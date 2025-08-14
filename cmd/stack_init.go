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
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
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
			lsd_program.SetProgramID(lsdProgramID)
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
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			latestBlockHashRes, err := rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
			if err != nil {
				return fmt.Errorf("get recent block hash error: %w", err)
			}

			tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, []solana.Instruction{initializeStackInstruction}, feePayerAccountPubkey, true)
			if err != nil {
				return fmt.Errorf("NewTransaction failed, err: %s, tx: %s", err.Error(), tx.String())
			}

			privateKeyMap, err := utils.LoadPrivateKeysFromKeystore(cfg.KeystorePath)
			if err != nil {
				return err
			}

			feePayerAccount, exist := privateKeyMap[feePayerAccountPubkey.String()]
			if !exist {
				return fmt.Errorf("fee payer not exit in vault")
			}

			adminAccount, exist := privateKeyMap[adminAccountPubkey.String()]
			if !exist {
				return fmt.Errorf("admin not exit in vault")
			}

			if err = utils.SignAndSendTx(rpcClient, tx, utils.GetSignFunc(feePayerAccount, adminAccount, stackAccount), latestBlockHashRes.Value.LastValidBlockHeight); err != nil {
				return fmt.Errorf("sign and send tx failed: %w", err)
			}
			fmt.Println("tx hash:", tx.Signatures[0].String())

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
