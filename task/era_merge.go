package task

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/lsdprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

func (task *Task) EraMerge(stakeManagerAddr common.PublicKey) error {
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	if !isEmpty(&stakeManager.EraProcessData) {
		return nil
	}

	stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerAddr.Bytes(), stakePoolSeed}, task.lsdProgramID)
	if err != nil {
		return err
	}

	valToAccount := make(map[string]map[uint64][]common.PublicKey) // voter -> credit -> []stakeAccount
	for _, stakeAccount := range stakeManager.StakeAccounts {
		accountInfo, err := task.client.GetStakeActivation(
			context.Background(),
			stakeAccount.ToBase58(),
			client.GetStakeActivationConfig{})
		if err != nil {
			return err
		}
		if accountInfo.State != client.StakeActivationStateActive {
			continue
		}

		account, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
		if err != nil {
			return err
		}
		voter := account.StakeAccount.Info.Stake.Delegation.Voter.ToBase58()
		credit := account.StakeAccount.Info.Stake.CreditsObserved
		if valToAccount[voter] == nil {
			valToAccount[voter] = make(map[uint64][]common.PublicKey)
		}
		if valToAccount[voter][credit] == nil {
			valToAccount[voter][credit] = make([]common.PublicKey, 0)
		}

		valToAccount[voter][credit] = append(valToAccount[voter][credit], stakeAccount)
	}

	for _, creditToAccounts := range valToAccount {
		for _, accounts := range creditToAccounts {
			if len(accounts) < 2 {
				continue
			}
			res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
				Commitment: client.CommitmentConfirmed,
			})
			if err != nil {
				fmt.Printf("get recent block hash error, err: %v\n", err)
			}
			srcStakeAccount := accounts[1]
			dstStakeAccount := accounts[0]
			rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
				Instructions: []types.Instruction{
					lsdprog.EraMerge(
						task.lsdProgramID,
						stakeManagerAddr,
						srcStakeAccount,
						dstStakeAccount,
						stakePool,
					),
				},
				Signers:         []types.Account{task.feePayerAccount},
				FeePayer:        task.feePayerAccount.PublicKey,
				RecentBlockHash: res.Blockhash,
			})
			if err != nil {
				fmt.Printf("generate tx error, err: %v\n", err)
			}
			txHash, err := task.client.SendRawTransaction(context.Background(), rawTx)
			if err != nil {
				fmt.Printf("send tx error, err: %v\n", err)
			}

			logrus.Infof("EraMerge send tx hash: %s, srcStakeAccount: %s, dstStakeAccount: %s",
				txHash, srcStakeAccount.ToBase58(), dstStakeAccount.ToBase58())
			if err := task.waitTx(txHash); err != nil {
				stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
				if err != nil {
					return err
				}
				stakeAccountExist := make(map[string]bool)
				for _, stakeAccount := range stakeManagerNew.StakeAccounts {
					stakeAccountExist[stakeAccount.ToBase58()] = true
				}

				if !stakeAccountExist[srcStakeAccount.ToBase58()] || !stakeAccountExist[dstStakeAccount.ToBase58()] {
					logrus.Info("EraMerge success")
					continue
				}

				return err
			}

			logrus.Info("EraMerge success")

		}
	}

	return nil
}
