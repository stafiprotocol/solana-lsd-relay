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

func (t *Task) EraMerge(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakePool, err := t.getStakeManagerAndPool(stakeManagerPubkey)
	if err != nil {
		return err
	}

	if !stakeManager.EraProcessData.IsEmpty() {
		return nil
	}

	valToAccount := make(map[string]map[uint64][]solana.PublicKey) // voter -> credit -> []stakeAccount
	for _, stakeAccount := range stakeManager.StakeAccounts {
		accountInfo, err := t.client.GetStakeActivation(
			context.Background(),
			stakeAccount,
			rpc.CommitmentConfirmed,
			nil,
		)
		if err != nil {
			return err
		}
		if accountInfo.State != rpc.ActivationStateActive {
			continue
		}

		stakeAccountInfo := lsd_program.StakeAccount{}
		if err = utils.GetAndDecodeAccountInfo(t.client, stakeAccount, &stakeAccountInfo); err != nil {
			return fmt.Errorf("get stake account info error: %w", err)
		}

		voter := stakeAccountInfo.Info.Stake.Delegation.Voter.String()
		credit := stakeAccountInfo.Info.Stake.CreditsObserved
		if valToAccount[voter] == nil {
			valToAccount[voter] = make(map[uint64][]solana.PublicKey)
		}
		if valToAccount[voter][credit] == nil {
			valToAccount[voter][credit] = make([]solana.PublicKey, 0)
		}

		valToAccount[voter][credit] = append(valToAccount[voter][credit], stakeAccount)
	}

	for _, creditToAccounts := range valToAccount {
		for _, accounts := range creditToAccounts {
			if len(accounts) < 2 {
				continue
			}

			srcStakeAccount := accounts[1]
			dstStakeAccount := accounts[0]

			eraMergeInstruction, err := lsd_program.NewEraMergeInstruction(
				stakeManagerPubkey,
				srcStakeAccount,
				dstStakeAccount,
				stakePool,
				solana.SysVarClockPubkey,
				solana.SysVarStakeHistoryPubkey,
				solana.StakeProgramID,
			)
			if err != nil {
				return fmt.Errorf("new era merge instruction error: %w", err)
			}

			latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
			if err != nil {
				return fmt.Errorf("get recent block hash error: %w", err)
			}

			tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{eraMergeInstruction}, t.feePayerAccount.PublicKey(), true)
			if err != nil {
				return fmt.Errorf("new solana transaction error: %w", err)
			}

			logrus.Infof("EraMerge send tx hash: %s, srcStakeAccount: %s, dstStakeAccount: %s", tx.Signatures[0], srcStakeAccount, dstStakeAccount)
			err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
			if err == nil {
				logrus.Info("EraMerge success")
				return nil
			}

			stakeManagerNew, _, verifyErr := t.getStakeManagerAndPool(stakeManagerPubkey)
			if verifyErr != nil {
				return verifyErr
			}

			stakeAccountExist := make(map[string]bool)
			for _, stakeAccount := range stakeManagerNew.StakeAccounts {
				stakeAccountExist[stakeAccount.String()] = true
			}

			if !stakeAccountExist[srcStakeAccount.String()] || !stakeAccountExist[dstStakeAccount.String()] {
				logrus.Info("EraMerge success")
				continue
			}

			return fmt.Errorf("EraMerge failed err: %w", err)
		}
	}

	return nil
}
