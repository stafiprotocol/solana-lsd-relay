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

func (t *Task) EraSkipBond(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, _, err := t.getStakeManagerAndPool(stakeManagerPubkey)
	if err != nil {
		return err
	}

	minDelegationAmount, err := utils.GetMinDelegationAmount(t.client)
	if err != nil {
		return err
	}

	if !stakeManager.EraProcessData.IsNeedSkipBond(minDelegationAmount) {
		return nil
	}

	eraSkipBondInstruction := lsd_program.NewEraSkipBondInstruction(stakeManagerPubkey, solana.StakeProgramID).Build()

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
		return fmt.Errorf("EraSkipBond failed err: %w", err)
	}
	logrus.Infof("EraSkipBond send tx hash: %s, skipBondAmount: %d", tx.Signatures[0].String(), stakeManager.EraProcessData.NeedBond)
	logrus.Infof("EraSkipBond success")
	return nil

}
