package stack

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stack"
	"golang.org/x/time/rate"
)

func AddEntrustedStakeManager() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "add-entrusted-stake-manager",
		Short: "Add entrusted stakeManager",

		RunE: func(cmd *cobra.Command, args []string) error {
			exportTxMessage, err := cmd.Flags().GetBool(common.FlagExportTx)
			if err != nil {
				return err
			}

			configPath, err := cmd.Flags().GetString(common.FlagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadConfig[config.ConfigSetStack](configPath)
			if err != nil {
				return err
			}

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

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))
			stackAccount, err := rpcClient.GetAccountInfo(context.Background(), stackPubkey)
			if err != nil {
				return err
			}
			stack.SetProgramID(stackAccount.Value.Owner)

			instruction := stack.NewAddEntrustedStakeManagerInstruction(addEntrustedStakeManagerPubkey, stackPubkey, adminPubkey).Build()
			instructions := []solana.Instruction{instruction}

			_, err = common.AdminExecuteInstructions("add entrusted stake manager", rpcClient, instructions, cfg.KeystorePath, feePayerPubkey, adminPubkey, exportTxMessage)
			if err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().String(common.FlagConfigPath, common.DefaultConfigPath, "Config file path")
	cmd.Flags().Bool(common.FlagExportTx, false, "Export tx message")
	return cmd
}
