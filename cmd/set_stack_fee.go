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

func setStackFee() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "set-stack-fee",
		Short: "Set stack fee",

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
			lsd_program.SetProgramID(solana.MustPublicKeyFromBase58(cfg.LsdProgramID))

			adminPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			stackPubkey := solana.MustPublicKeyFromBase58(cfg.StackAddress)
			stakeManagerPubkey := solana.MustPublicKeyFromBase58(cfg.StakeManagerAddress)

			fmt.Println("stack:", stackPubkey)
			fmt.Println("stakeManager:", stakeManagerPubkey)
			fmt.Println("stackFeeCommission:", cfg.StackFeeCommission)
			fmt.Println("admin:", adminPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
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
			instruction := lsd_program.NewSetPlatformStackFeeCommissionInstruction(cfg.StackFeeCommission, stakeManagerPubkey, stackPubkey, adminPubkey).Build()
			instructions := []solana.Instruction{instruction}

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.EndpointList[0],
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			_, err = adminExecuteInstructions("set stack fee", rpcClient, instructions, cfg.KeystorePath, feePayerPubkey, adminPubkey, exportTxMessage)
			return err
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	cmd.Flags().Bool(flagExportTx, false, "Export tx message")
	return cmd
}
