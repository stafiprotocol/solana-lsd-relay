package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"golang.org/x/time/rate"
)

func stackInitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "init",
		Short: "Init stack",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadInitStackConfig(configPath)
			if err != nil {
				return err
			}
			lsdProgramID := solana.MustPublicKeyFromBase58(cfg.LsdProgramID)
			lsd_program.ProgramID = lsdProgramID
			feePayerAccountPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			adminAccountPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)

			stackAccount, err := solana.NewRandomPrivateKey()
			if err != nil {
				return err
			}

			fmt.Println("lsdProgramID:", lsd_program.ProgramID)
			fmt.Println("admin:", adminAccountPubkey)
			fmt.Println("feePayer:", feePayerAccountPubkey)
			fmt.Println("stack(randomly generated):", stackAccount.PublicKey())
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

			initializeStackInstruction := lsd_program.NewInitializeStackInstruction(
				stackAccount.PublicKey(),
				feePayerAccountPubkey,
				adminAccountPubkey,
				solana.SystemProgramID,
			).Build()

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.EndpointList[0],
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			_, err = adminExecuteInstructions(
				"initialize stack",
				rpcClient,
				[]solana.Instruction{initializeStackInstruction},
				cfg.KeystorePath,
				feePayerAccountPubkey,
				adminAccountPubkey,
				false)
			if err != nil {
				return err
			}

			retry := 0
			for {
				if retry > 60 {
					return fmt.Errorf("tx failed")
				}
				_, err := rpcClient.GetAccountInfo(context.Background(), stackAccount.PublicKey())
				if err != nil {
					retry++
					time.Sleep(time.Second)
					continue
				}

				break
			}

			return nil
		},
	}
	cmd.Flags().String(flagConfigPath, defaultConfigPath, "Config file path")
	return cmd
}
