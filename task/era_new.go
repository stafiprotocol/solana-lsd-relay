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

func (t *Task) EraNew(stakeManagerPubkey solana.PublicKey) error {
	stakeManager := lsd_program.StakeManager{}
	err := utils.GetAndDecodeAccountInfo(t.client, stakeManagerPubkey, &stakeManager)
	if err != nil {
		return err
	}

	epochInfo, err := t.client.GetEpochInfo(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return err
	}
	if stakeManager.LatestEra >= uint64(epochInfo.Epoch) {
		return nil
	}

	if !stakeManager.EraProcessData.IsEmpty() {
		return nil
	}

	eraNewInstruction, err := lsd_program.NewEraNewInstruction(stakeManagerPubkey, solana.SysVarClockPubkey)
	if err != nil {
		return err
	}

	latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("get recent block hash error: %w", err)
	}

	tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraNewInstruction}, t.feePayerAccount.PublicKey(), true)
	if err != nil {
		return err
	}

	err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
	txHash := tx.Signatures[0].String()
	logrus.Infof("EraNew send tx hash: %s, newEra: %d", txHash, stakeManager.LatestEra+1)
	if err == nil {
		logrus.Infof("EraNew success")
		return nil
	}

	// verify tx success
	stakeManagerNew, _, getErr := t.getStakeManagerAndPool(stakeManagerPubkey)
	if getErr != nil {
		return getErr
	}

	if stakeManagerNew.LatestEra > stakeManager.LatestEra {
		logrus.Infof("EraNew success")
		return nil
	}

	return fmt.Errorf("EraNew failed err: %w", err)
}
