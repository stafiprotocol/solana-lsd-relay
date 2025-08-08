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

func (t *Task) EraUnbond(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakePool, err := t.getStakeManagerAndPool(stakeManagerPubkey)
	if err != nil {
		return err
	}

	if !stakeManager.EraProcessData.IsNeedUnbond() {
		return nil
	}

	stakeAccount := stakeManager.StakeAccounts[0] // use first

	stakeAccountInfo := lsd_program.StakeAccount{}
	if err = utils.GetAndDecodeAccountInfo(t.client, stakeAccount, &stakeAccountInfo); err != nil {
		return fmt.Errorf("get stake account info error: %w", err)
	}

	validator := stakeAccountInfo.Info.Stake.Delegation.Voter
	splitStakeAccount, err := solana.NewRandomPrivateKey()
	if err != nil {
		return fmt.Errorf("new random private key for split stake account error: %w", err)
	}

	eraUnbondInstruction := lsd_program.NewEraUnbondInstruction(
		stakeManagerPubkey,
		stakePool,
		stakeAccount,
		splitStakeAccount.PublicKey(),
		validator,
		t.feePayerAccount.PublicKey(),
		solana.SysVarClockPubkey,
		solana.SysVarRentPubkey,
		solana.SysVarStakeHistoryPubkey,
		solana.StakeProgramID,
		solana.SystemProgramID,
	).Build()

	latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("get recent block hash error: %w", err)
	}

	tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraUnbondInstruction}, t.feePayerAccount.PublicKey(), false)
	if err != nil {
		return err
	}

	err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount, splitStakeAccount), latestBlockHashRes.Value.LastValidBlockHeight)
	txHash := tx.Signatures[0].String()
	logrus.Infof("EraUnbond send tx hash: %s, unbondAmount: %d", txHash, stakeManager.EraProcessData.NeedUnbond)
	if err == nil {
		logrus.Infof("EraUnbond success")
		return nil
	}

	stakeManagerNew, _, getErr := t.getStakeManagerAndPool(stakeManagerPubkey)
	if getErr != nil {
		return getErr
	} else if !stakeManagerNew.EraProcessData.IsNeedUnbond() {
		logrus.Info("EraUnbond success")
		return nil
	}

	return fmt.Errorf("EraUnbond failed err: %w", err)
}
