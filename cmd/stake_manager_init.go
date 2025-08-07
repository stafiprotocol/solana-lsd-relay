package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"golang.org/x/time/rate"
)

var stakePoolSeed = []byte("pool_seed")
var stakeManagerSeed = "stake_manager_seed_%d"

var StakeManagerAccountLengthDefault = uint64(100000)
var StackAccountLengthDefault = uint64(1000)
var StackFeeAccountLengthDefault = uint64(17)

func stakeManagerInitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "init",
		Short: "Init stake manager",

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

			cfg, err := config.LoadInitStakeManagerConfig(configPath)
			if err != nil {
				return err
			}
			lsdProgramID := solana.MustPublicKeyFromBase58(cfg.LsdProgramID)

			stackPubkey := solana.MustPublicKeyFromBase58(cfg.StackAddress)
			lsdTokenMintPubkey := solana.MustPublicKeyFromBase58(cfg.LsdTokenMintAddress)
			validatorPubkey := solana.MustPublicKeyFromBase58(cfg.ValidatorAddress)
			feePayerPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			adminPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.EndpointList[0],
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))
			var stakeManagerPubkey solana.PublicKey
			var seed string
			for i := 0; ; i++ {
				seed = fmt.Sprintf(stakeManagerSeed, i)

				stakeManagerPubkey, err = solana.CreateWithSeed(feePayerPubkey, seed, lsdProgramID)
				if err != nil {
					return err
				}
				_, err := rpcClient.GetAccountInfo(context.Background(), stakeManagerPubkey)
				if err != nil {
					if err == rpc.ErrNotFound {
						break
					} else {
						return err
					}
				}
			}
			if cfg.StakeManagerAddress != stakeManagerPubkey.String() {
				return fmt.Errorf("stake manager not match: cfg: %s, avaiable create stake manager: %s",
					cfg.StakeManagerAddress, stakeManagerPubkey.String())
			}

			stackFeeAccountPubkey, _, err := solana.FindProgramAddress([][]byte{stackPubkey.Bytes(), lsdTokenMintPubkey.Bytes()}, lsdProgramID)
			if err != nil {
				return err
			}
			stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, lsdProgramID)
			if err != nil {
				return err
			}

			stakePoolRent, err := rpcClient.GetMinimumBalanceForRentExemption(context.Background(), 0, rpc.CommitmentConfirmed)
			if err != nil {
				return err
			}

			stakeManagerRent, err := rpcClient.GetMinimumBalanceForRentExemption(context.Background(), StakeManagerAccountLengthDefault, rpc.CommitmentConfirmed)
			if err != nil {
				return err
			}

			fmt.Println("lsdProgramID:", lsdProgramID)
			fmt.Println("stack:", stackPubkey)
			fmt.Println("lsdTokenMint:", lsdTokenMintPubkey)
			fmt.Println("stakeManager:", stakeManagerPubkey)
			fmt.Println("stakePool:", stakePool)
			fmt.Println("stackFeeAccount(determinately generated):", stackFeeAccountPubkey)
			fmt.Println("admin:", adminPubkey)
			fmt.Println("feePayer:", feePayerPubkey)
			fmt.Println("stakePool rent:", stakePoolRent)
			fmt.Println("stakeManager rent:", stakeManagerRent)
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

			transferInstruction := system.NewTransferInstructionBuilder().
				SetLamports(stakePoolRent).
				SetFundingAccount(feePayerPubkey).
				SetRecipientAccount(stakePool).
				Build()

			createAccountInstruction := system.NewCreateAccountWithSeedInstruction(
				feePayerPubkey,
				seed,
				stakeManagerRent,
				StakeManagerAccountLengthDefault,
				lsdProgramID,
				feePayerPubkey,
				stakeManagerPubkey,
				feePayerPubkey,
			).Build()

			initInstruction, err := lsd_program.NewInitializeStakeManagerInstruction(
				stakeManagerPubkey, stackPubkey, stakePool, stackFeeAccountPubkey,
				lsdTokenMintPubkey, validatorPubkey, feePayerPubkey, adminPubkey,
				solana.SPLAssociatedTokenAccountProgramID, system.ProgramID, solana.SysVarClockPubkey, solana.SysVarRentPubkey)
			if err != nil {
				return fmt.Errorf("NewInitializeStakeManagerInstruction failed, err: %s", err.Error())
			}
			instructions := []solana.Instruction{
				transferInstruction,
				createAccountInstruction,
				initInstruction,
			}

			tx, err := AdminExecuteInstructions(rpcClient, instructions, cfg.KeystorePath, feePayerPubkey, adminPubkey, exportTxMessage)
			if err != nil {
				return err
			}
			fmt.Println("initializeStakeManager txHash:", tx.Signatures[0].String())
			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	cmd.Flags().Bool(flagExportTx, false, "Export tx message")
	return cmd
}
