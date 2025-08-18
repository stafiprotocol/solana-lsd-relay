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
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

func InitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "init",
		Short: "Init stack",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(common.FlagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadConfig[config.ConfigInitStack](configPath)
			if err != nil {
				return err
			}
			stackProgramID := solana.MustPublicKeyFromBase58(cfg.StackProgramID)
			stack.SetProgramID(stackProgramID)

			feePayerAccountPubkey := solana.MustPublicKeyFromBase58(cfg.FeePayerAccount)
			adminAccountPubkey := solana.MustPublicKeyFromBase58(cfg.AdminAccount)

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				cfg.Endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			var stackAccount solana.PublicKey
			var stackIndex uint8
			for i := range uint8(255) {
				stackAccount, _, err = solana.FindProgramAddress([][]byte{[]byte("stack_seed"), adminAccountPubkey.Bytes(), {i}}, stackProgramID)
				if err != nil {
					return err
				}
				_, err = rpcClient.GetAccountInfo(context.Background(), stackAccount)
				if err != nil {
					if err == rpc.ErrNotFound {
						stackIndex = i
						break
					} else {
						return err
					}
				}
			}

			fmt.Println("stackProgramID:", stack.ProgramID)
			fmt.Println("stack:", stackAccount)
			fmt.Println("admin:", adminAccountPubkey)
			fmt.Println("feePayer:", feePayerAccountPubkey)
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

			initializeStackInstruction := stack.NewInitializeStackInstruction(
				stackIndex,
				stackAccount,
				feePayerAccountPubkey,
				adminAccountPubkey,
				solana.SystemProgramID,
			).Build()

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

			if err = utils.SignAndSendTx(rpcClient, tx, utils.GetSignFunc(feePayerAccount, adminAccount), latestBlockHashRes.Value.LastValidBlockHeight); err != nil {
				return fmt.Errorf("sign and send tx failed: %w", err)
			}
			fmt.Println("tx hash:", tx.Signatures[0].String())

			return nil
		},
	}
	cmd.Flags().String(common.FlagConfigPath, common.DefaultConfigPath, "Config file path")
	return cmd
}
