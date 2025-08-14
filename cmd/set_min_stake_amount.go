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

func stakeManagerSetMinStakeAmountCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "set-min-stake-amount",
		Short: "Set min stake amount",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadSetStakeManagerConfig(configPath)
			if err != nil {
				return err
			}
			lsd_program.SetProgramID(solana.MustPublicKeyFromBase58(cfg.LsdProgramID))

			stakeManagerPubkey := solana.MustPublicKeyFromBase58(cfg.StakeManagerAddress)
			adminPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)

			fmt.Println("stakeManager:", stakeManagerPubkey)
			fmt.Println("admin:", adminPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
			fmt.Println("minStakeAmount(lamports):", cfg.MinStakeAmount)
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

			setMinStakeAmountInstruction := lsd_program.NewSetMinStakeAmountInstruction(
				cfg.MinStakeAmount,
				stakeManagerPubkey,
				adminPubkey,
			).Build()

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			_, err = adminExecuteInstructions("set min stake amount", rpcClient, []solana.Instruction{setMinStakeAmountInstruction}, cfg.KeystorePath, feePayerPubkey, adminPubkey, cfg.ExportTx)
			return err
		},
	}
	cmd.Flags().String(flagConfigPath, "config_set_stakemanager.toml", "Config file path, example: config_set_stakemanager.example.toml")
	return cmd
}
