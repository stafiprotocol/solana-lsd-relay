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

func (task *Task) EraNew(stakeManagerAddr common.PublicKey) error {
	epochInfo, err := task.client.GetEpochInfo(context.Background(), client.CommitmentFinalized)
	if err != nil {
		return err
	}
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}
	if stakeManager.LatestEra >= uint64(epochInfo.Epoch) {
		return nil
	}

	if !isEmpty(&stakeManager.EraProcessData) {
		return nil
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions: []types.Instruction{
			lsdprog.EraNew(
				task.lsdProgramID,
				stakeManagerAddr,
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

	logrus.Infof("EraNew send tx hash: %s, newEra: %d", txHash, stakeManager.LatestEra+1)
	if err := task.waitTx(txHash); err != nil {
		stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
		if err != nil {
			return err
		}
		if stakeManagerNew.LatestEra > stakeManager.LatestEra {
			logrus.Infof("EraNew success")
			return nil
		}

		return err
	}
	logrus.Infof("EraNew success")

	return nil
}
