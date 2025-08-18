// Copyright 2020 tpkeeper
// SPDX-License-Identifier: LGPL-3.0-only

package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stake_manager"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

func TestStake(t *testing.T) {
	// endpoint := "https://api.devnet.solana.com"
	// endpoint := "https://api.testnet.sonic.game"
	endpoint := "https://solana-dev-rpc.stafi.io"
	rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		endpoint,
		rate.Every(time.Second), // time frame
		5,                       // limit of requests per time frame
	))

	stakeManager := solana.MustPublicKeyFromBase58("5T3iezF2qUNfHGSWKLm6SVfZYVE8yipRbmUzGgiQaRs4")
	lsdTokenMint := solana.MustPublicKeyFromBase58("2qCaXMkiEujzNjA9s9xRn6t8tYtoHxg7QJQHbw27d5yZ")
	stakeManagerProgramID := solana.MustPublicKeyFromBase58("HQHnPC158TLWsENn1TLvk3LG4wwkfM2QTxm9fvrmW26F")

	user, err := solana.PrivateKeyFromSolanaKeygenFile("/Users/tpkeeper/.config/solana/fee_payer.json")
	if err != nil {
		t.Fatal(err)
	}
	stake_manager.SetProgramID(stakeManagerProgramID)

	stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManager.Bytes(), utils.StakePoolSeed}, stakeManagerProgramID)
	if err != nil {
		t.Fatal(err)
	}

	userLsdTokenAccount, _, err := solana.FindAssociatedTokenAddress(user.PublicKey(), lsdTokenMint)
	if err != nil {
		t.Fatal(err)
	}

	instructions := []solana.Instruction{}

	instructions = append(instructions, stake_manager.NewStakeInstruction(
		2000000000,
		stakeManager,
		stakePool,
		user.PublicKey(),
		user.PublicKey(),
		lsdTokenMint,
		userLsdTokenAccount,
		solana.SystemProgramID,
		solana.SPLAssociatedTokenAccountProgramID,
		solana.TokenProgramID,
	).Build())

	latestBlockHashRes, err := rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := solana.NewTransaction(
		instructions,
		latestBlockHashRes.Value.Blockhash,
		solana.TransactionPayer(user.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if user.PublicKey().Equals(key) {
			return &user
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Stake will send tx: %s", tx.Signatures[0])

	_, err = utils.SendAndWaitForConfirmation(rpcClient, tx, latestBlockHashRes.Value.LastValidBlockHeight)
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnStake(t *testing.T) {
	// endpoint := "https://api.devnet.solana.com"
	// endpoint := "https://api.testnet.sonic.game"
	endpoint := "https://solana-dev-rpc.stafi.io"
	user, err := solana.PrivateKeyFromSolanaKeygenFile("/Users/tpkeeper/.config/solana/fee_payer.json")
	if err != nil {
		t.Fatal(err)
	}

	stakeManager := solana.MustPublicKeyFromBase58("5T3iezF2qUNfHGSWKLm6SVfZYVE8yipRbmUzGgiQaRs4")
	lsdTokenMint := solana.MustPublicKeyFromBase58("2qCaXMkiEujzNjA9s9xRn6t8tYtoHxg7QJQHbw27d5yZ")
	stakeManagerProgramID := solana.MustPublicKeyFromBase58("HQHnPC158TLWsENn1TLvk3LG4wwkfM2QTxm9fvrmW26F")

	stake_manager.SetProgramID(stakeManagerProgramID)

	userLsdTokenAccount, _, err := solana.FindAssociatedTokenAddress(user.PublicKey(), lsdTokenMint)
	if err != nil {
		t.Fatal(err)
	}

	unstakeAccount, _ := solana.NewRandomPrivateKey()

	t.Logf("unstakeAccount: %s", unstakeAccount.String())

	instructions := []solana.Instruction{
		stake_manager.NewUnstakeInstruction(
			1000000,
			stakeManager,
			lsdTokenMint,
			userLsdTokenAccount,
			user.PublicKey(),
			unstakeAccount.PublicKey(),
			user.PublicKey(),
			solana.SystemProgramID,
			solana.TokenProgramID,
		).Build(),
	}

	rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		endpoint,
		rate.Every(time.Second), // time frame
		5,                       // limit of requests per time frame
	))

	latestBlockHashRes, err := rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := solana.NewTransaction(
		instructions,
		latestBlockHashRes.Value.Blockhash,
		solana.TransactionPayer(user.PublicKey()))
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if user.PublicKey().Equals(key) {
			return &user
		}
		if unstakeAccount.PublicKey().Equals(key) {
			return &unstakeAccount
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Unstake will send tx: %s", tx.Signatures[0])

	_, err = utils.SendAndWaitForConfirmation(rpcClient, tx, latestBlockHashRes.Value.LastValidBlockHeight)
	if err != nil {
		t.Fatal(err)
	}
}
