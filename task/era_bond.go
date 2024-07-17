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

func (task *Task) EraBond(stakeManagerAddr common.PublicKey) error {
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	if !needBond(&stakeManager.EraProcessData) {
		return nil
	}

	stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerAddr.Bytes(), stakePoolSeed}, task.lsdProgramID)
	if err != nil {
		return err
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	stakeAccount := types.NewAccount() //random account

	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions: []types.Instruction{
			lsdprog.EraBond(
				task.lsdProgramID,
				stakeManagerAddr,
				stakeManager.Validators[0], // use first validator
				stakePool,
				stakeAccount.PublicKey,
				task.feePayerAccount.PublicKey,
			),
		},
		Signers:         []types.Account{task.feePayerAccount, stakeAccount},
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

	logrus.Infof("EraBond send tx hash: %s, stakeAccount: %s, bond: %d",
		txHash, stakeAccount.PublicKey.ToBase58(), stakeManager.EraProcessData.NeedBond)
	if err := task.waitTx(txHash); err != nil {
		stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
		if err != nil {
			return err
		}
		if !needBond(&stakeManagerNew.EraProcessData) {
			logrus.Info("EraBond success")
			return nil
		}
		return err
	}
	logrus.Info("EraBond success")

	return nil
}
