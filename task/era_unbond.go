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

func (t *Task) EraUnbond(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakeManagerProgramID, stakePool, err := utils.GetStakeManagerInfo(t.client, stakeManagerPubkey)
	if err != nil {
		return err
	}
	stake_manager.SetProgramID(stakeManagerProgramID)

	if !utils.IsNeedUnbond(stakeManager.EraProcessData) {
		return nil
	}

	stakeAccount := stakeManager.StakeAccounts[0] // use first

	stakeAccountInfo := utils.StakeAccount{}
	if err = utils.GetAndDecodeAccountInfo(t.client, stakeAccount, &stakeAccountInfo); err != nil {
		return fmt.Errorf("get stake account info error: %w", err)
	}

	validator := stakeAccountInfo.Info.Stake.Delegation.Voter
	splitStakeAccount, err := solana.NewRandomPrivateKey()
	if err != nil {
		return fmt.Errorf("new random private key for split stake account error: %w", err)
	}

	eraUnbondInstruction := stake_manager.NewEraUnbondInstruction(
		stakeManagerPubkey,
		stakePool,
		stakeAccount,
		splitStakeAccount.PublicKey(),
		validator,
		t.feePayerAccount.PublicKey(),
		solana.SysVarClockPubkey,
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
	if err != nil {
		return err
	}
	logrus.Infof("EraUnbond send tx hash: %s, unbondAmount: %d", tx.Signatures[0], stakeManager.EraProcessData.NeedUnbond)
	logrus.Infof("EraUnbond success")

	return nil
}
