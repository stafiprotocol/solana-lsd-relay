package task

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stake_manager"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

func (t *Task) EraWithdraw(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakeManagerProgramID, stakePool, err := utils.GetStakeManagerInfo(t.client, stakeManagerPubkey)
	if err != nil {
		return err
	}
	stake_manager.SetProgramID(stakeManagerProgramID)

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
		stakeAccountInfo := utils.StakeAccount{}
		if err = utils.GetAndDecodeAccountInfo(t.client, stakeAccount, &stakeAccountInfo); err != nil {
			return fmt.Errorf("get stake account info error: %w", err)
		}

		eraWithdrawInstruction := stake_manager.NewEraWithdrawInstruction(
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
		if err != nil {
			return err
		}
		logrus.Infof("EraWithdraw send tx hash: %s, stakeAccount: %s", tx.Signatures[0], stakeAccount)
		logrus.Info("EraWithdraw success")
	}

	return nil
}
