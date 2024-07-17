package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/lsdprog"
	"github.com/stafiprotocol/solana-go-sdk/sysprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/vault"
)

var stakePoolSeed = []byte("pool_seed")
var stakeManagerSeed = "stake_manager_seed_%d"

func stakeManagerInitCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "init",
		Short: "Init stake manager",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(flagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("config path: %s\n", configPath)

			cfg, err := config.LoadInitStakeManagerConfig(configPath)
			if err != nil {
				return err
			}
			v, err := vault.NewVaultFromWalletFile(cfg.KeystorePath)
			if err != nil {
				return err
			}
			boxer, err := vault.SecretBoxerForType(v.SecretBoxWrap)
			if err != nil {
				return fmt.Errorf("secret boxer: %w", err)
			}

			if err := v.Open(boxer); err != nil {
				return fmt.Errorf("opening: %w", err)
			}

			privateKeyMap := make(map[string]vault.PrivateKey)
			accountMap := make(map[string]types.Account)
			for _, privKey := range v.KeyBag {
				privateKeyMap[privKey.PublicKey().String()] = privKey
				accountMap[privKey.PublicKey().String()] = types.AccountFromPrivateKeyBytes(privKey)
			}

			c := client.NewClient(cfg.EndpointList)

			res, err := c.GetLatestBlockhash(context.Background(), client.GetLatestBlockhashConfig{
				Commitment: client.CommitmentConfirmed,
			})
			if err != nil {
				fmt.Printf("get recent block hash error, err: %v\n", err)
			}

			lsdTokenMintPubkey := common.PublicKeyFromString(cfg.LsdTokenMintAddress)
			lsdProgramID := common.PublicKeyFromString(cfg.LsdProgramID)
			validatorPubkey := common.PublicKeyFromString(cfg.ValidatorAddress)
			stackPubkey := common.PublicKeyFromString(cfg.StackAddress)

			feePayerAccount, exist := accountMap[cfg.FeePayerAccount]
			if !exist {
				return fmt.Errorf("fee payer not exit in vault")
			}
			adminAccount, exist := accountMap[cfg.AdminAccount]
			if !exist {
				return fmt.Errorf("admin not exit in vault")
			}

			var stakeManagerPubkey common.PublicKey
			var seed string
			for i := 0; ; i++ {
				seed = fmt.Sprintf(stakeManagerSeed, i)
				stakeManagerPubkey = common.CreateWithSeed(feePayerAccount.PublicKey, seed, lsdProgramID)
				_, err := c.GetAccountInfo(context.Background(), stakeManagerPubkey.ToBase58(), client.GetAccountInfoConfig{
					Encoding:  client.GetAccountInfoConfigEncodingBase64,
					DataSlice: client.GetAccountInfoConfigDataSlice{},
				})
				if err != nil {
					if err == client.ErrAccountNotFound {
						break
					} else {
						return err
					}
				}
			}
			if cfg.StakeManagerAddress != stakeManagerPubkey.ToBase58() {
				return fmt.Errorf("stake manager not match: cfg: %s, gen: %s", cfg.StakeManagerAddress, stakeManagerPubkey.ToBase58())
			}

			stackFeeAccountPubkey, _, err := common.FindProgramAddress([][]byte{stackPubkey.Bytes(), lsdTokenMintPubkey.Bytes()}, lsdProgramID)
			if err != nil {
				return err
			}
			stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, lsdProgramID)
			if err != nil {
				return err
			}

			stakePoolRent, err := c.GetMinimumBalanceForRentExemption(context.Background(), 0)
			if err != nil {
				return err
			}

			stakeManagerRent, err := c.GetMinimumBalanceForRentExemption(context.Background(), lsdprog.StakeManagerAccountLengthDefault)
			if err != nil {
				return err
			}

			fmt.Println("lsdProgramID:", lsdProgramID.ToBase58())
			fmt.Println("stack:", stackPubkey.ToBase58())
			fmt.Println("lsdTokenMint:", lsdTokenMintPubkey.ToBase58())
			fmt.Println("stakeManager:", stakeManagerPubkey.ToBase58())
			fmt.Println("stakePool:", stakePool.ToBase58())
			fmt.Println("stackFeeAccount(determinately generated):", stackFeeAccountPubkey.ToBase58())
			fmt.Println("admin", adminAccount.PublicKey.ToBase58())
			fmt.Println("feePayer:", feePayerAccount.PublicKey.ToBase58())
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

			rawTx, err := types.CreateRawTransaction(types.CreateRawTransactionParam{
				Instructions: []types.Instruction{
					sysprog.Transfer(
						feePayerAccount.PublicKey,
						stakePool,
						stakePoolRent,
					),
					sysprog.CreateAccountWithSeed(
						feePayerAccount.PublicKey,
						stakeManagerPubkey,
						feePayerAccount.PublicKey,
						lsdProgramID,
						seed,
						stakeManagerRent,
						lsdprog.StakeManagerAccountLengthDefault,
					),
					lsdprog.InitializeStakeManager(
						lsdProgramID,
						stakeManagerPubkey,
						stackPubkey,
						stakePool,
						stackFeeAccountPubkey,
						lsdTokenMintPubkey,
						validatorPubkey,
						feePayerAccount.PublicKey,
						adminAccount.PublicKey,
					),
				},
				Signers:         []types.Account{feePayerAccount, adminAccount},
				FeePayer:        feePayerAccount.PublicKey,
				RecentBlockHash: res.Blockhash,
			})
			if err != nil {
				fmt.Printf("generate tx error, err: %v\n", err)
			}
			txHash, err := c.SendRawTransaction(context.Background(), rawTx)
			if err != nil {
				fmt.Printf("send tx error, err: %v\n", err)
			}

			fmt.Println("initializeStakeManager txHash:", txHash)

			retry := 0
			for {
				if retry > 60 {
					return fmt.Errorf("tx %s failed", txHash)
				}
				_, err = c.GetAccountInfo(context.Background(), stakeManagerPubkey.ToBase58(), client.GetAccountInfoConfig{
					Encoding:  client.GetAccountInfoConfigEncodingBase64,
					DataSlice: client.GetAccountInfoConfigDataSlice{},
				})
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
