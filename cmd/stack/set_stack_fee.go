package stack

import (
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stake_manager"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

func SetStackFee() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "set-stack-fee",
		Short: "Set stack fee",

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

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))
			_, stakeManagerProgramID, _, err := utils.GetStakeManagerInfo(rpcClient, stakeManagerPubkey)
			stake_manager.SetProgramID(stakeManagerProgramID)

			instruction := stake_manager.NewSetPlatformStackFeeCommissionInstruction(cfg.StackFeeCommission, stakeManagerPubkey, stackPubkey, adminPubkey).Build()
			instructions := []solana.Instruction{instruction}

			_, err = common.AdminExecuteInstructions("set stack fee", rpcClient, instructions, cfg.KeystorePath, feePayerPubkey, adminPubkey, exportTxMessage)
			return err
		},
	}
	cmd.Flags().String(common.FlagConfigPath, common.DefaultConfigPath, "Config file path")
	cmd.Flags().Bool(common.FlagExportTx, false, "Export tx message")
	return cmd
}
