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

func (task *Task) EraUpdateActive(stakeManagerAddr common.PublicKey) error {
	for {
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	if !needUpdateActive(&stakeManager.EraProcessData) {
		return nil
	}

	eraActive := stakeManager.EraProcessData.OldActive
	eraProcessActive := stakeManager.EraProcessData.NewActive

	stakeAccount := stakeManager.EraProcessData.PendingStakeAccounts[0]
	stakeAccountInfo, err := task.client.GetStakeAccountInfo(context.Background(), stakeAccount.ToBase58())
	if err != nil {
		return err
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}
	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions: []types.Instruction{
			lsdprog.EraUpdateActive(
				task.lsdProgramID,
				stakeManagerAddr,
				stakeAccount,
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

	logrus.Infof("EraUpdateActive send tx hash: %s, stakeAccount: %s, stakeAccoutActive: %d, eraSnapshotActive: %d, eraProcessActive(old): %d, eraProcessActive(new): %d",
		txHash, stakeAccount.ToBase58(), stakeAccountInfo.StakeAccount.Info.Stake.Delegation.Stake, eraActive, eraProcessActive, int64(eraProcessActive)+stakeAccountInfo.StakeAccount.Info.Stake.Delegation.Stake)

	if err := task.waitTx(txHash); err != nil {
		stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
		if err != nil {
			return err
		}
		if !needUpdateActive(&stakeManagerNew.EraProcessData) {
			logrus.Info("EraUpdateActive success")
			return nil
		}
		if stakeManagerNew.EraProcessData.PendingStakeAccounts[0] != stakeAccount {
			logrus.Info("EraUpdateActive success")
		}
		return err
	}

	logrus.Info("EraUpdateActive success")
	}
}
