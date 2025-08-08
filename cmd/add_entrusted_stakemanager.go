package cmd

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"golang.org/x/time/rate"
)

func addEntrustedStakeManager() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "add-entrusted-stake-manager",
		Short: "Add entrusted stakeManager",

		RunE: func(cmd *cobra.Command, args []string) error {
			exportTxMessage, err := cmd.Flags().GetBool(flagExportTx)
			if err != nil {
				return err
			}

			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadInitStackConfig(configPath)
			if err != nil {
				return err
			}
			lsd_program.ProgramID = solana.MustPublicKeyFromBase58(cfg.LsdProgramID)

			stackPubkey := solana.MustPublicKeyFromBase58(cfg.StackAddress)
			adminPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			addEntrustedStakeManagerPubkey := solana.MustPublicKeyFromBase58(cfg.AddEntrustedStakeManagerAddress)

			fmt.Println("stack:", stackPubkey)
			fmt.Println("admin:", adminPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
			fmt.Println("addEntrustedStakeManagerPubkey:", addEntrustedStakeManagerPubkey)
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

			instruction, err := lsd_program.NewAddEntrustedStakeManagerInstruction(addEntrustedStakeManagerPubkey, stackPubkey, adminPubkey)
			if err != nil {
				return fmt.Errorf("NewAddEntrustedStakeManagerInstruction failed, err: %s", err.Error())
			}
			instructions := []solana.Instruction{instruction}

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.EndpointList[0],
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			_, err = AdminExecuteInstructions("add entrusted stake manager", rpcClient, instructions, cfg.KeystorePath, feePayerPubkey, adminPubkey, exportTxMessage)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	cmd.Flags().Bool(flagExportTx, false, "Export tx message")
	return cmd
}
