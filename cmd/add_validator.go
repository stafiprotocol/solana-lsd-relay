package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/lsdprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/vault"
)

func stakeManagerAddValidator() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "add-validator",
		Short: "Add validator",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadInitStakeManagerConfig(configPath)
			if err != nil {
				return err
			}
			v, err := vault.NewVaultFromWalletFile(cfg.KeystorePath)
			if err != nil {
				return err
			}
			boxer, err := vault.SecretBoxerForType(v.SecretBoxWrap)
			if err != nil {
				return fmt.Errorf("secret boxer: %w", err)
			}

			if err := v.Open(boxer); err != nil {
				return fmt.Errorf("opening: %w", err)
			}

			privateKeyMap := make(map[string]vault.PrivateKey)
			accountMap := make(map[string]types.Account)
			for _, privKey := range v.KeyBag {
				privateKeyMap[privKey.PublicKey().String()] = privKey
				accountMap[privKey.PublicKey().String()] = types.AccountFromPrivateKeyBytes(privKey)
			}

			c := client.NewClient(cfg.EndpointList)

			res, err := c.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
				Commitment: client.CommitmentConfirmed,
			})
			if err != nil {
				fmt.Printf("get recent block hash error, err: %v\n", err)
			}

			lsdProgramID := common.PublicKeyFromString(cfg.LsdProgramID)

			feePayerAccount, exist := accountMap[cfg.FeePayerAccount]
			if !exist {
				return fmt.Errorf("fee payer not exit in vault")
			}
			adminAccount, exist := accountMap[cfg.AdminAccount]
			if !exist {
				return fmt.Errorf("admin not exit in vault")
			}

			addValidatorPubkey := common.PublicKeyFromString(cfg.AddValidatorAddress)
			stakeManagerPubkey := common.PublicKeyFromString(cfg.StakeManagerAddress)

			fmt.Println("stakeManager:", stakeManagerPubkey.ToBase58())
			fmt.Println("admin", adminAccount.PublicKey.ToBase58())
			fmt.Println("feePayer:", feePayerAccount.PublicKey.ToBase58())
			fmt.Println("addValidatorAddress:", cfg.AddValidatorAddress)
		Out:
			for {
				fmt.Println("\ncheck config info, then press (y/n) to continue:")
				var input string
				fmt.Scanln(&input)
				switch input {
				case "y":
					break Out
				case "n":
					return nil
				default:
					fmt.Println("press `y` or `n`")
					continue
				}
			}

			rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
				Instructions: []types.Instruction{
					lsdprog.AddValidator(
						lsdProgramID,
						stakeManagerPubkey,
						adminAccount.PublicKey,
						addValidatorPubkey,
					),
				},
				Signers:         []types.Account{feePayerAccount, adminAccount},
				FeePayer:        feePayerAccount.PublicKey,
				RecentBlockHash: res.Blockhash,
			})
			if err != nil {
				fmt.Printf("generate tx error, err: %v\n", err)
			}
			txHash, err := c.SendRawTransaction(context.Background(), rawTx)
			if err != nil {
				fmt.Printf("send tx error, err: %v\n", err)
			}

			fmt.Println("AddValidator txHash:", txHash)

			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	return cmd
}
