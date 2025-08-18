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

func (t *Task) EraSkipBond(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakeManagerProgramID, _, err := utils.GetStakeManagerInfo(t.client, stakeManagerPubkey)
	if err != nil {
		return err
	}
	stake_manager.SetProgramID(stakeManagerProgramID)

	minDelegationAmount, err := utils.GetMinDelegationAmount(t.client)
	if err != nil {
		return err
	}

	if !utils.IsNeedSkipBond(stakeManager.EraProcessData, minDelegationAmount) {
		return nil
	}

	eraSkipBondInstruction := stake_manager.NewEraSkipBondInstruction(stakeManagerPubkey, solana.StakeProgramID).Build()

	latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("get recent block hash error: %w", err)
	}

	tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraSkipBondInstruction}, t.feePayerAccount.PublicKey(), true)
	if err != nil {
		return err
	}

	err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
	if err != nil {
		return err
	}
	txHash := tx.Signatures[0].String()
	logrus.Infof("EraSkipBond send tx hash: %s, skipBondAmount: %d", txHash, stakeManager.EraProcessData.NeedBond)
	logrus.Infof("EraSkipBond success")
	return nil
}
