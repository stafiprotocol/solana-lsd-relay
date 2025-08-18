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

func (t *Task) EraBond(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakeManagerProgramID, stakePool, err := utils.GetStakeManagerInfo(t.client, stakeManagerPubkey)
	if err != nil {
		return err
	}
	stake_manager.SetProgramID(stakeManagerProgramID)

	minDelegationAmount, err := utils.GetMinDelegationAmount(t.client)
	if err != nil {
		return err
	}

	if !utils.IsNeedBond(stakeManager.EraProcessData, minDelegationAmount) {
		return nil
	}

	stakeAccount, err := solana.NewRandomPrivateKey()
	if err != nil {
		return err
	}

	eraBondInstruction := stake_manager.NewEraBondInstruction(
		stakeManagerPubkey,
		stakeManager.Validators[0],
		stakePool,
		stakeAccount.PublicKey(),
		t.feePayerAccount.PublicKey(),
		solana.SysVarClockPubkey,
		solana.SysVarRentPubkey,
		solana.SysVarStakeConfigPubkey,
		solana.SysVarStakeHistoryPubkey,
		solana.StakeProgramID,
		solana.SystemProgramID,
	).Build()

	latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("get recent block hash error: %w", err)
	}

	tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraBondInstruction}, t.feePayerAccount.PublicKey(), false)
	if err != nil {
		return fmt.Errorf("NewTransaction failed, err: %s, tx: %s", err.Error(), tx.String())
	}
	signFunc := utils.GetSignFunc(t.feePayerAccount, stakeAccount)
	if err = utils.SignAndSendTx(t.client, tx, signFunc, latestBlockHashRes.Value.LastValidBlockHeight); err != nil {
		return fmt.Errorf("SignAndSendTx failed, err: %s", err.Error())
	}

	logrus.Infof("EraBond send tx hash: %s, stakeAccount: %s, bond: %d",
		tx.Signatures[0], stakeAccount.PublicKey(), stakeManager.EraProcessData.NeedBond)
	logrus.Info("EraBond success")
	return nil
}
