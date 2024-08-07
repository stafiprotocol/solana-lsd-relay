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

func (task *Task) EraSkipBond(stakeManagerAddr common.PublicKey) error {
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	minDelegationAmount, err := task.client.GetMinDelegationAmount(context.Background())
	if err != nil {
		return err
	}

	if !needSkipBond(&stakeManager.EraProcessData, minDelegationAmount) {
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
			lsdprog.EraSkipBond(
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

	logrus.Infof("EraSkipBond send tx hash: %s,  skipBondAmount: %d",
		txHash, stakeManager.EraProcessData.NeedBond)
	if err := task.waitTx(txHash); err != nil {
		stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
		if err != nil {
			return err
		}
		if !needSkipBond(&stakeManagerNew.EraProcessData, minDelegationAmount) {
			logrus.Info("EraSkipBond success")
			return nil
		}
		return err
	}
	logrus.Info("EraSkipBond success")

	return nil
}
