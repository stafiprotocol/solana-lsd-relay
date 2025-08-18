package stake_manager

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stake_manager"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

func SetStakeManagerCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "set-stake-manager",
		Short: "Set stake manager",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(common.FlagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg := config.ConfigSetStakeManager{
				RateChangeLimit:       -1,
				UnbondingDuration:     -1,
				MinStakeAmount:        -1,
				PlatformFeeCommission: -1,
			}
			if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
				return err
			}

			stakeManagerPubkey := solana.MustPublicKeyFromBase58(cfg.StakeManagerAddress)
			adminPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)

			fmt.Println("stakeManager:", stakeManagerPubkey)
			fmt.Println("admin:", adminPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
			fmt.Println("exportTx:", cfg.ExportTx)

			var minStakeAmount *uint64
			if cfg.MinStakeAmount >= 0 {
				amount := uint64(cfg.MinStakeAmount)
				minStakeAmount = &amount
				fmt.Printf("minStakeAmount: %d\n", *minStakeAmount)
			}
			var platformFeeCommission *uint64
			if cfg.PlatformFeeCommission >= 0 {
				amount := uint64(cfg.PlatformFeeCommission)
				platformFeeCommission = &amount
				fmt.Printf("platformFeeCommission: %d\n", *platformFeeCommission)
			}
			var unbondingDuration *uint64
			if cfg.UnbondingDuration >= 0 {
				amount := uint64(cfg.UnbondingDuration)
				unbondingDuration = &amount
				fmt.Printf("unbondingDuration: %d\n", *unbondingDuration)
			}
			var rateChangeLimit *uint64
			if cfg.RateChangeLimit >= 0 {
				amount := uint64(cfg.RateChangeLimit)
				rateChangeLimit = &amount
				fmt.Printf("rateChangeLimit: %d\n", *rateChangeLimit)
			}
			var newBalancerAddress *solana.PublicKey
			if len(cfg.NewBalancerAddress) > 0 {
				address := solana.MustPublicKeyFromBase58(cfg.NewBalancerAddress)
				newBalancerAddress = &address
				fmt.Printf("newBalancerAddress: %s\n", newBalancerAddress.String())
			}
			var addValidatorAddress *solana.PublicKey
			if len(cfg.AddValidatorAddress) > 0 {
				address := solana.MustPublicKeyFromBase58(cfg.AddValidatorAddress)
				addValidatorAddress = &address
				fmt.Printf("addValidatorAddress: %s\n", addValidatorAddress.String())
			}
			var removeValidatorAddress *solana.PublicKey
			if len(cfg.RemoveValidatorAddress) > 0 {
				address := solana.MustPublicKeyFromBase58(cfg.RemoveValidatorAddress)
				removeValidatorAddress = &address
				fmt.Printf("removeValidatorAddress: %s\n", removeValidatorAddress.String())
			}

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

			setStakeManagerInstruction := stake_manager.NewConfigStakeManagerInstruction(
				stake_manager.ConfigStakeManagerParams{
					MinStakeAmount:        minStakeAmount,
					PlatformFeeCommission: platformFeeCommission,
					UnbondingDuration:     unbondingDuration,
					RateChangeLimit:       rateChangeLimit,
					Balancer:              newBalancerAddress,
					AddValidator:          addValidatorAddress,
					RemoveValidator:       removeValidatorAddress,
				},
				stakeManagerPubkey,
				adminPubkey,
			).Build()

			_, err = common.AdminExecuteInstructions("set stake manager", rpcClient,
				[]solana.Instruction{setStakeManagerInstruction}, cfg.KeystorePath, feePayerPubkey, adminPubkey, cfg.ExportTx)
			return err
		},
	}
	cmd.Flags().String(common.FlagConfigPath, "config_set_stakemanager.toml", "Config file path, example: config_set_stakemanager.example.toml")
	return cmd
}
