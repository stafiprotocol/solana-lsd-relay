// Copyright 2020 tpkeeper
// SPDX-License-Identifier: LGPL-3.0-only

package utils_test

import (
	"context"
	"testing"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

var stakePoolSeed = []byte("pool_seed")

func TestStake(t *testing.T) {
	// endpoint := "https://api.devnet.solana.com"
	// endpoint := "https://api.testnet.sonic.game"
	endpoint := "https://solana-dev-rpc.stafi.io"
	rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		endpoint,
		rate.Every(time.Second), // time frame
		5,                       // limit of requests per time frame
	))

	stakeManager := solana.MustPublicKeyFromBase58("JAAGMA3nXSFq3QhSMC9Trkf5hneMGoGRLaGtEkmL1Nmj")
	lsdTokenMint := solana.MustPublicKeyFromBase58("9aCVwV3SkrZfqiL5ShjC8hHy61TV7ufUquh3NyAwJ5gA")
	stakeManagerProgramID := solana.MustPublicKeyFromBase58("795MBfkwwtAX4fWiFqZcJK8D91P9tqqtiSRrSNhBvGzq")

	user, err := solana.PrivateKeyFromSolanaKeygenFile("/Users/tpkeeper/.config/solana/fee_payer.json")
	if err != nil {
		t.Fatal(err)
	}
	lsd_program.SetProgramID(stakeManagerProgramID)

	stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManager.Bytes(), stakePoolSeed}, stakeManagerProgramID)
	if err != nil {
		t.Fatal(err)
	}

	userLsdTokenAccount, _, err := solana.FindAssociatedTokenAddress(user.PublicKey(), lsdTokenMint)
	if err != nil {
		t.Fatal(err)
	}

	instructions := []solana.Instruction{}

	instructions = append(instructions, lsd_program.NewStakeInstruction(
		20000000000,
		stakeManager,
		stakePool,
		user.PublicKey(),
		lsdTokenMint,
		userLsdTokenAccount,
		solana.SystemProgramID,
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

	stakeManager := solana.MustPublicKeyFromBase58("JAAGMA3nXSFq3QhSMC9Trkf5hneMGoGRLaGtEkmL1Nmj")
	lsdTokenMint := solana.MustPublicKeyFromBase58("9aCVwV3SkrZfqiL5ShjC8hHy61TV7ufUquh3NyAwJ5gA")
	stakeManagerProgramID := solana.MustPublicKeyFromBase58("795MBfkwwtAX4fWiFqZcJK8D91P9tqqtiSRrSNhBvGzq")

	lsd_program.SetProgramID(stakeManagerProgramID)

	userLsdTokenAccount, _, err := solana.FindAssociatedTokenAddress(user.PublicKey(), lsdTokenMint)
	if err != nil {
		t.Fatal(err)
	}

	unstakeAccount, _ := solana.NewRandomPrivateKey()

	t.Logf("unstakeAccount: %s", unstakeAccount.String())

	instructions := []solana.Instruction{
		lsd_program.NewUnstakeInstruction(
			2e9,
			stakeManager,
			lsdTokenMint,
			userLsdTokenAccount,
			user.PublicKey(),
			unstakeAccount.PublicKey(),
			user.PublicKey(),
			solana.SystemProgramID,
			solana.TokenProgramID,
			solana.SysVarClockPubkey,
			solana.SysVarRentPubkey,
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

func TestCalStakeActivation(t *testing.T) {
	endpoint := "https://api.mainnet-beta.solana.com"
	rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		endpoint,
		rate.Every(time.Second), // time frame
		5,                       // limit of requests per time frame
	))

	stakeAccounts := []string{
		"8uzkBgKn5NwFwqwbwwHaNUcRwb4D73BGmvn8zczuEyeF",
		"FV3AgEGwA6mXrrRgaVeNaPASQwCHfVgm2nma1ehDgnfY",
		"FV7q9oMEAzziqQrD51gMR1xt12MfHMEjvebyknVLNJLp"}
	for _, address := range stakeAccounts {

		res, err := utils.CalStakeActivation(context.Background(), rpcClient, solana.MustPublicKeyFromBase58(address))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("%+v", res)
	}
}
