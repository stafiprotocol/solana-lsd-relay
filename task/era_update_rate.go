package task

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-go-sdk/assotokenprog"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/lsdprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
)

func (task *Task) EraUpdataRate(stakeManagerAddr common.PublicKey) error {
	stakeManager, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	if !needUpdataRate(&stakeManager.EraProcessData) {
		return nil
	}
	stackAccount, err := task.client.GetLsdStack(context.Background(), task.stackAccountPubkey.ToBase58())
	if err != nil {
		return err
	}

	stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerAddr.Bytes(), stakePoolSeed}, task.lsdProgramID)
	if err != nil {
		return err
	}

	instructions := make([]types.Instruction, 0)

	platformFeeRecipient, _, err := common.FindAssociatedTokenAddress(stakeManager.Admin, stakeManager.LsdTokenMint)
	if err != nil {
		return err
	}

	_, err = task.client.GetTokenAccountInfo(context.Background(), platformFeeRecipient.ToBase58())
	if err != nil {
		if err == client.ErrAccountNotFound {
			instructions = append(instructions, assotokenprog.CreateAssociatedTokenAccount(
				task.feePayerAccount.PublicKey, stakeManager.Admin, stakeManager.LsdTokenMint))
		} else {
			return err
		}
	}

	stackFeeRecipient, _, err := common.FindAssociatedTokenAddress(stackAccount.Admin, stakeManager.LsdTokenMint)
	if err != nil {
		return err
	}

	if platformFeeRecipient != stackFeeRecipient {
		_, err = task.client.GetTokenAccountInfo(context.Background(), stackFeeRecipient.ToBase58())
		if err != nil {
			if err == client.ErrAccountNotFound {
				instructions = append(instructions, assotokenprog.CreateAssociatedTokenAccount(
					task.feePayerAccount.PublicKey, stackAccount.Admin, stakeManager.LsdTokenMint))
			} else {
				return err
			}
		}
	}

	stackFeeAccount, _, err := common.FindProgramAddress([][]byte{task.stackAccountPubkey.Bytes(), stakeManager.LsdTokenMint.Bytes()}, task.lsdProgramID)
	if err != nil {
		return err
	}

	res, err := task.client.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
		Commitment: client.CommitmentConfirmed,
	})
	if err != nil {
		fmt.Printf("get recent block hash error, err: %v\n", err)
	}

	lsdTokenMint := stakeManager.LsdTokenMint

	instructions = append(instructions, lsdprog.EraUpdateRate(
		task.lsdProgramID,
		stakeManagerAddr,
		task.stackAccountPubkey,
		stakePool,
		lsdTokenMint,
		platformFeeRecipient,
		stackFeeRecipient,
		stackFeeAccount,
	))

	rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
		Instructions:    instructions,
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

	logrus.Infof("EraUpdateRate send tx hash: %s, pipelineActive: %d, eraSnapshotActive: %d, eraProcessActive: %d, rate(old): %d",
		txHash, stakeManager.Active, stakeManager.EraProcessData.OldActive, stakeManager.EraProcessData.NewActive, stakeManager.Rate)
	if err := task.waitTx(txHash); err != nil {
		stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
		if err != nil {
			return err
		}

		if !needUpdataRate(&stakeManagerNew.EraProcessData) {
			logrus.Infof("EraUpdateRate success, rate(new): %d", stakeManagerNew.Rate)
			return nil
		}
		return err
	}
	stakeManagerNew, err := task.client.GetLsdStakeManager(context.Background(), stakeManagerAddr.ToBase58())
	if err != nil {
		return err
	}

	logrus.Infof("EraUpdateRate success, rate(new): %d", stakeManagerNew.Rate)
	return nil
}
