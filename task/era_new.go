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

func (t *Task) EraNew(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakeManagerProgramID, _, err := utils.GetStakeManagerInfo(t.client, stakeManagerPubkey)
	if err != nil {
		return err
	}
	stake_manager.SetProgramID(stakeManagerProgramID)

	epochInfo, err := t.client.GetEpochInfo(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return err
	}
	if stakeManager.LatestEra >= uint64(epochInfo.Epoch) {
		return nil
	}

	if !utils.IsEmpty(stakeManager.EraProcessData) {
		return nil
	}

	eraNewInstruction := stake_manager.NewEraNewInstruction(stakeManagerPubkey).Build()

	latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("get recent block hash error: %w", err)
	}

	tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraNewInstruction}, t.feePayerAccount.PublicKey(), true)
	if err != nil {
		return err
	}

	err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
	if err != nil {
		return err
	}
	txHash := tx.Signatures[0].String()
	logrus.Infof("EraNew send tx hash: %s, newEra: %d", txHash, stakeManager.LatestEra+1)
	logrus.Infof("EraNew success")
	return nil
}
