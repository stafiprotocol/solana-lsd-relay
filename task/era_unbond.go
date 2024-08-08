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

func (task *Task) EraUnbond(stakeManagerAddr common.PublicKey) error {
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	if !needUnbond(&stakeManager.EraProcessData) {
		return nil
	}

	stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerAddr.Bytes(), stakePoolSeed}, task.lsdProgramID)
	if err != nil {
		return err
	}

	stakeAccount := stakeManager.StakeAccounts[0] // use first
	stakeAccountInfo, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
	if err != nil {
		return err
	}
	validator := stakeAccountInfo.StakeAccount.Info.Stake.Delegation.Voter

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}
	splitStakeAccount := types.NewAccount() //random account

	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions: []types.Instruction{
			lsdprog.EraUnbond(
				task.lsdProgramID,
				stakeManagerAddr,
				stakePool,
				stakeAccount,
				splitStakeAccount.PublicKey,
				validator,
				task.feePayerAccount.PublicKey,
			),
		},
		Signers:         []types.Account{task.feePayerAccount, splitStakeAccount},
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

	logrus.Infof("EraUnbond send tx hash: %s, splitStakeAccount: %s, unbond: %d",
		txHash, splitStakeAccount.PublicKey.ToBase58(), stakeManager.EraProcessData.NeedBond)
	if err := task.waitTx(txHash); err != nil {
		stakeManagerNew, errInside := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
		if errInside != nil {
			return errInside
		}
		if stakeManagerNew.EraProcessData.NeedUnbond < stakeManager.EraProcessData.NeedUnbond {
			logrus.Info("EraUnbond success")
			return nil
		}

		return err
	}

	logrus.Info("EraUnbond success")
	return nil
}
