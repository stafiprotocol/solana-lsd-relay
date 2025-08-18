package stake_manager

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stake_manager"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

var StakeManagerAccountLengthDefault = uint64(100000)
var StackAccountLengthDefault = uint64(1000)
var StackFeeAccountLengthDefault = uint64(17)

func InitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "init",
		Short: "Init stake manager",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(common.FlagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadConfig[config.ConfigInitStakeManager](configPath)
			if err != nil {
				return err
			}
			stakeManagerProgramID := solana.MustPublicKeyFromBase58(cfg.StakeManagerProgramID)
			stake_manager.SetProgramID(stakeManagerProgramID)

			stackPubkey := solana.MustPublicKeyFromBase58(cfg.StackAddress)
			validatorPubkey := solana.MustPublicKeyFromBase58(cfg.ValidatorAddress)
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			adminPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			var stakeManagerPubkey solana.PublicKey
			var stakeManagerIndex uint8
			for i := range uint8(255) {
				stakeManagerPubkey, _, err = solana.FindProgramAddress([][]byte{
					[]byte("stake_manager_seed"), adminPubkey.Bytes(), {i}}, stakeManagerProgramID)
				if err != nil {
					return err
				}
				_, err = rpcClient.GetAccountInfo(context.Background(), stakeManagerPubkey)
				if err != nil {
					if err == rpc.ErrNotFound {
						stakeManagerIndex = uint8(i)
						break
					} else {
						return err
					}
				}

			}

			stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), utils.StakePoolSeed}, stakeManagerProgramID)
			if err != nil {
				return err
			}
			lsdTokenMintPubkey, _, err := solana.FindProgramAddress([][]byte{[]byte(utils.TokenMintSeed), adminPubkey.Bytes(), []byte{stakeManagerIndex}}, stakeManagerProgramID)
			if err != nil {
				return err
			}
			stackFeeAccountPubkey, _, err := solana.FindProgramAddress([][]byte{stackPubkey.Bytes(), lsdTokenMintPubkey.Bytes()}, stakeManagerProgramID)
			if err != nil {
				return err
			}

			fmt.Println("stakeManagerProgramID:", stakeManagerProgramID)
			fmt.Println("stakeManager:", stakeManagerPubkey)
			fmt.Println("stack:", stackPubkey)
			fmt.Println("lsdTokenMint:", lsdTokenMintPubkey)
			fmt.Println("stakePool:", stakePool)
			fmt.Println("admin:", adminPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
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

			stakePoolRent, err := rpcClient.GetMinimumBalanceForRentExemption(context.Background(), 0, rpc.CommitmentConfirmed)
			if err != nil {
				return err
			}
			transferInstruction := system.NewTransferInstructionBuilder().
				SetLamports(stakePoolRent).
				SetFundingAccount(feePayerPubkey).
				SetRecipientAccount(stakePool).
				Build()

			initInstruction := stake_manager.NewInitializeStakeManagerInstruction(
				stakeManagerIndex, stakeManagerPubkey, stackPubkey, stakePool, stackFeeAccountPubkey,
				lsdTokenMintPubkey, validatorPubkey, feePayerPubkey, adminPubkey,
				solana.TokenProgramID, system.ProgramID, solana.SysVarRentPubkey,
			).Build()

			instructions := []solana.Instruction{
				transferInstruction,
				initInstruction,
			}

			_, err = common.AdminExecuteInstructions("initialize stake manager", rpcClient, instructions, cfg.KeystorePath, feePayerPubkey, adminPubkey, false)
			return err
		},
	}
	cmd.Flags().String(common.FlagConfigPath, common.DefaultConfigPath, "Config file path")
	return cmd
}
