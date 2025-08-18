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

func AcceptAdminCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "accept-admin",
		Short: "Accept admin",

		RunE: func(cmd *cobra.Command, args []string) error {
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
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			newAdminPubkey := solana.MustPublicKeyFromBase58(cfg.NewAdminAddress)

			fmt.Println("stack:", stackPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
			fmt.Println("newAdminAddress:", newAdminPubkey)
			fmt.Println("exportTx:", cfg.ExportTx)
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

			account, err := rpcClient.GetAccountInfo(context.Background(), stackPubkey)
			if err != nil {
				return err
			}
			stack.SetProgramID(account.Value.Owner)
			transferAdminInstruction := stack.NewAcceptStackAdminInstruction(
				newAdminPubkey,
				stackPubkey,
			).Build()

			_, err = common.AdminExecuteInstructions("accept admin", rpcClient,
				[]solana.Instruction{transferAdminInstruction}, cfg.KeystorePath, feePayerPubkey, newAdminPubkey, cfg.ExportTx)
			return err
		},
	}
	cmd.Flags().String(common.FlagConfigPath, "config_set_stack.toml", "Config file path, example: config_set_stack.example.toml")
	return cmd
}
