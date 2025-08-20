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

func (t *Task) EraUpdateActive(stakeManagerPubkey solana.PublicKey) error {
	for {
		stakeManager, _, err := t.getStakeManagerAndPool(stakeManagerPubkey)
		if err != nil {
			return err
		}

		if !stakeManager.EraProcessData.IsNeedUpdateActive() {
			return nil
		}

		eraActive := stakeManager.EraProcessData.OldActive
		eraProcessActive := stakeManager.EraProcessData.NewActive

		stakeAccountPubkey := stakeManager.EraProcessData.PendingStakeAccounts[0]
		stakeAccount := lsd_program.StakeAccount{}
		if err = utils.GetAndDecodeAccountInfo(t.client, stakeAccountPubkey, &stakeAccount); err != nil {
			return fmt.Errorf("get stake account info error: %w", err)
		}

		eraUpdateActiveInstruction := lsd_program.NewEraUpdateActiveInstruction(stakeManagerPubkey, stakeAccountPubkey).Build()

		latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
		if err != nil {
			return fmt.Errorf("get recent block hash error: %w", err)
		}

		tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraUpdateActiveInstruction}, t.feePayerAccount.PublicKey(), true)
		if err != nil {
			return fmt.Errorf("new solana transaction error: %w", err)
		}

		err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
		logrus.Infof("EraUpdateActive send tx hash: %s, stakeAccount: %s, stakeAccoutActive: %d, eraSnapshotActive: %d, eraProcessActive(old): %d, eraProcessActive(new): %d",
			tx.Signatures[0], stakeAccountPubkey, stakeAccount.Info.Stake.Delegation.Stake, eraActive, eraProcessActive, uint64(eraProcessActive)+stakeAccount.Info.Stake.Delegation.Stake)
		if err == nil {
			logrus.Info("EraUpdateActive success")
			return nil
		}

		stakeManagerNew, _, verifyErr := t.getStakeManagerAndPool(stakeManagerPubkey)
		if verifyErr != nil {
			return verifyErr
		}
		if !stakeManagerNew.EraProcessData.IsNeedUpdateActive() {
			logrus.Info("EraUpdateActive success")
			return nil
		}
		if stakeManagerNew.EraProcessData.PendingStakeAccounts[0] != stakeAccountPubkey {
			logrus.Info("EraUpdateActive success")
		}

		return fmt.Errorf("EraUpdateActive failed err: %w", err)
	}
}
