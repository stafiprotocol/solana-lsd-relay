package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"golang.org/x/time/rate"
	"time"
)

// TokenMetadataProgramID is the program ID for the token metadata program
var TokenMetadataProgramID = solana.MustPublicKeyFromBase58("metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s")

func CreateTokenMetadataCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "create-token-metadata",
		Short: "Create metadata for LSD token",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadCreateMetadataConfig(configPath)
			if err != nil {
				return err
			}
			lsd_program.SetProgramID(solana.MustPublicKeyFromBase58(cfg.LsdProgramID))

			bts, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Printf("Config: \n%s\n", string(bts))
		Out:
			for {
				fmt.Println("\ncheck account info, then press (y/n) to continue:")
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

			feePayerAccountPublicKey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			adminAccountPublicKey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)

			stakeManagerPubkey := solana.MustPublicKeyFromBase58(cfg.StakeManagerAddress)
			stakePoolPubkey := solana.MustPublicKeyFromBase58(cfg.StakePoolAddress)
			lsdTokenMintPubkey := solana.MustPublicKeyFromBase58(cfg.LsdTokenMintAddress)

			// Derive metadata account
			metadataAccount, _, err := solana.FindProgramAddress(
				[][]byte{
					[]byte("metadata"),
					TokenMetadataProgramID.Bytes(),
					lsdTokenMintPubkey.Bytes(),
				},
				TokenMetadataProgramID,
			)
			if err != nil {
				return fmt.Errorf("failed to derive metadata account: %w", err)
			}

			createMetdataInstruction := lsd_program.NewCreateMetadataV1Instruction(
				lsd_program.CreateMetadataParams{
					TokenName:   cfg.TokenName,
					TokenSymbol: cfg.TokenSymbol,
					TokenUri:    cfg.TokenUri,
				},

				feePayerAccountPublicKey,
				adminAccountPublicKey,
				stakeManagerPubkey,
				stakePoolPubkey,
				lsdTokenMintPubkey,
				metadataAccount,
				TokenMetadataProgramID,
				solana.SystemProgramID,
				solana.SysVarInstructionsPubkey,
				solana.Token2022ProgramID,
			).Build()
			instructions := []solana.Instruction{createMetdataInstruction}

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			_, err = adminExecuteInstructions("create token metadata", rpcClient, instructions, cfg.KeystorePath,
				feePayerAccountPublicKey, adminAccountPublicKey, cfg.ExportTx)
			return err
		},
	}
	cmd.Flags().String(flagConfigPath, "config_create_metadata.toml", "Config file path, example: config_create_metadata.example.toml")
	return cmd
}
