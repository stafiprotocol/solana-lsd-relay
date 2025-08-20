package task

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

func (t *Task) EraWithdraw(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakePool, err := t.getStakeManagerAndPool(stakeManagerPubkey)
	if err != nil {
		return err
	}

	withdrawableAccounts := make([]solana.PublicKey, 0)
	for _, account := range stakeManager.SplitAccounts {
		accountInfo, err := t.client.GetStakeActivation(
			context.Background(),
			account,
			rpc.CommitmentConfirmed,
			nil,
		)
		if err != nil {
			return err
		}
		if accountInfo.State == rpc.ActivationStateInactive {
			withdrawableAccounts = append(withdrawableAccounts, account)
		}
	}

	if len(withdrawableAccounts) == 0 {
		return nil
	}

	for _, stakeAccount := range withdrawableAccounts {
		stakeAccountInfo := lsd_program.StakeAccount{}
		if err = utils.GetAndDecodeAccountInfo(t.client, stakeAccount, &stakeAccountInfo); err != nil {
			return fmt.Errorf("get stake account info error: %w", err)
		}

		eraWithdrawInstruction := lsd_program.NewEraWithdrawInstruction(
			stakeManagerPubkey,
			stakePool,
			stakeAccount,
			solana.SysVarClockPubkey,
			solana.SysVarStakeHistoryPubkey,
			solana.StakeProgramID,
		).Build()

		latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
		if err != nil {
			return fmt.Errorf("get recent block hash error: %w", err)
		}

		tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraWithdrawInstruction}, t.feePayerAccount.PublicKey(), true)
		if err != nil {
			return fmt.Errorf("new solana transaction error: %w", err)
		}
		err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
		logrus.Infof("EraWithdraw send tx hash: %s, stakeAccount: %s", tx.Signatures[0], stakeAccount)
		if err == nil {
			logrus.Info("EraWithdraw success")
			return nil
		}

		stakeAccountInfoNew := lsd_program.StakeAccount{}
		if verifyErr := utils.GetAndDecodeAccountInfo(t.client, stakeAccount, &stakeAccountInfoNew); verifyErr != nil && verifyErr == rpc.ErrNotFound {
			logrus.Info("EraWithdraw success")
			return nil
		}

		return fmt.Errorf("EraWithdraw failed err: %w", err)
	}

	return nil
}
