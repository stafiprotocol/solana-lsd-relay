package task

import (
	"context"
	"fmt"

	"github.com/gagliardetto/solana-go"
	associatedtokenaccount "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

func (t *Task) EraUpdateRate(stakeManagerPubkey solana.PublicKey) error {
	stakeManager, stakePool, err := t.getStakeManagerAndPool(stakeManagerPubkey)
	if err != nil {
		return err
	}

	if !stakeManager.EraProcessData.IsNeedUpdateRate() {
		return nil
	}

	stackAccount := lsd_program.Stack{}
	if err = utils.GetAndDecodeAccountInfo(t.client, t.stackAccountPubkey, &stackAccount); err != nil {
		return fmt.Errorf("get stack account info error: %w", err)
	}

	instructions := make([]solana.Instruction, 0)

	platformFeeRecipient, _, err := solana.FindAssociatedTokenAddress(stakeManager.Admin, stakeManager.LsdTokenMint)
	if err != nil {
		return err
	}

	if _, err = t.client.GetAccountInfo(context.Background(), platformFeeRecipient); err != nil {
		if err == rpc.ErrNotFound {
			// create platform fee recipient account if not exist
			associatedTokenAccountInstruction := associatedtokenaccount.NewCreateInstruction(
				t.feePayerAccount.PublicKey(),
				stakeManager.Admin,
				stakeManager.LsdTokenMint,
			).Build()
			instructions = append(instructions, associatedTokenAccountInstruction)
		} else {
			return err
		}
	}

	stackFeeRecipient, _, err := solana.FindAssociatedTokenAddress(stackAccount.Admin, stakeManager.LsdTokenMint)
	if err != nil {
		return err
	}

	if platformFeeRecipient != stackFeeRecipient {
		if _, err = t.client.GetAccountInfo(context.Background(), stackFeeRecipient); err != nil {
			if err == rpc.ErrNotFound {
				// create stack fee recipient account if not exist
				associatedTokenAccountInstruction := associatedtokenaccount.NewCreateInstruction(
					t.feePayerAccount.PublicKey(),
					stackAccount.Admin,
					stakeManager.LsdTokenMint,
				).Build()
				instructions = append(instructions, associatedTokenAccountInstruction)
			} else {
				return err
			}
		}
	}

	stackFeeAccount, _, err := solana.FindProgramAddress([][]byte{t.stackAccountPubkey.Bytes(), stakeManager.LsdTokenMint.Bytes()}, t.lsdProgramID)
	if err != nil {
		return err
	}

	lsdTokenMintAccount, err := t.client.GetAccountInfo(context.Background(), stakeManager.LsdTokenMint)
	if err != nil {
		return err
	}

	var tokenProgramAccount solana.PublicKey
	if lsdTokenMintAccount.Value.Owner == solana.Token2022ProgramID {
		tokenProgramAccount = solana.Token2022ProgramID
	} else if lsdTokenMintAccount.Value.Owner == solana.TokenProgramID {
		tokenProgramAccount = solana.TokenProgramID
	} else {
		return fmt.Errorf("lsd token mint account owner is not token2022 or token program")
	}

	eraUpdateRateInstruction := lsd_program.NewEraUpdateRateInstruction(
		stakeManagerPubkey,
		t.stackAccountPubkey,
		stakePool,
		stakeManager.LsdTokenMint,
		platformFeeRecipient,
		stackFeeRecipient,
		stackFeeAccount,
		associatedtokenaccount.ProgramID,
		tokenProgramAccount,
	).Build()

	instructions = append(instructions, eraUpdateRateInstruction)

	latestBlockHashRes, err := t.client.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return fmt.Errorf("get recent block hash error: %w", err)
	}

	tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, instructions, t.feePayerAccount.PublicKey(), true)
	if err != nil {
		return fmt.Errorf("new solana transaction error: %w", err)
	}

	err = utils.SignAndSendTx(t.client, tx, utils.GetSignFunc(t.feePayerAccount), latestBlockHashRes.Value.LastValidBlockHeight)
	logrus.Infof("EraUpdateRate send tx hash: %s, pipelineActive: %d, eraSnapshotActive: %d, eraProcessActive: %d, rate(old): %d",
		tx.Signatures[0], stakeManager.Active, stakeManager.EraProcessData.OldActive, stakeManager.EraProcessData.NewActive, stakeManager.Rate)
	if err == nil {
		logrus.Info("EraUpdateActive success")
		return nil
	}

	stakeManagerNew, _, verifyErr := t.getStakeManagerAndPool(stakeManagerPubkey)
	if verifyErr != nil {
		return verifyErr
	}
	if !stakeManagerNew.EraProcessData.IsNeedUpdateRate() {
		logrus.Infof("EraUpdateRate success, rate(new): %d", stakeManagerNew.Rate)
		return nil
	}

	return fmt.Errorf("EraUpdateRate failed err: %w", err)
}
