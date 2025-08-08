package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/decred/base58"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

var (
	appName = "solana-lsd-relay"
)

const (
	flagLogLevel     = "log_level"
	flagConfigPath   = "config"
	flagFeePayer     = "fee_payer"
	flagStakeManager = "stake_manager"
	flagEndPoint     = "endpoint"
	flagLsdProgramID = "lsd_program_id"
	flagKeystorePath = "keystore_path"
	flagExportTx     = "export"

	defaultKeystorePath = "./keys/solana_keys.json"
	defaultConfigPath   = "./config.toml"
)

// NewRootCmd returns the root command.
func NewRootCmd() *cobra.Command {
	// RootCmd represents the base command when called without any subcommands
	var rootCmd = &cobra.Command{
		Use:   appName,
		Short: "solana-lsd-relay",
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, segments []string) error {
		return nil
	}

	rootCmd.AddCommand(
		keysCmd(),
		stackCmd(),
		stakeManagerCmd(),
		startCmd(),
		versionCmd(),
	)

	return rootCmd
}

func keysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage keystore",
	}

	cmd.AddCommand(
		vaultImportCmd(),
		vaultGenCmd(),
		vaultExportCmd(),
		vaultListCmd(),
	)
	return cmd
}

func stakeManagerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stake-manager",
		Short: "Stake manager operation",
	}

	cmd.AddCommand(
		stakeManagerInitCmd(),
		nextStakeManagerCmd(),
		stakeManagerDetailCmd(),
		stakeManagerSetRateLimitCmd(),
		stakeManagerSetUnbondingDurationCmd(),
		stakeManagerAddValidator(),
		stakeManagerRemoveValidator(),
	)
	return cmd
}

func stackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Stack operation",
	}

	cmd.AddCommand(
		stackInitCmd(),
		addEntrustedStakeManager(),
		setStackFee(),
	)
	return cmd
}

func Execute() {
	rootCmd := NewRootCmd()
	rootCmd.SilenceUsage = true
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	ctx := context.Background()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func adminExecuteInstructions(
	action string,
	rpcClient *rpc.Client,
	instructions []solana.Instruction,
	keystorePath string,
	feePayerPubkey solana.PublicKey,
	adminPubkey solana.PublicKey,
	exportTxMessage bool,
) (*solana.Transaction, error) {
	latestBlockHashRes, err := rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return nil, fmt.Errorf("get recent block hash error: %w", err)
	}

	if exportTxMessage {
		tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, instructions, feePayerPubkey, false)
		if err != nil {
			return nil, fmt.Errorf("NewTransaction failed, err: %s, tx: %s", err.Error(), tx.String())
		}
		bytes, err := tx.Message.MarshalBinary()
		if err != nil {
			return tx, fmt.Errorf("fail to marshal tx.Message: %w", err)
		}
		fmt.Println(action, "tx(base58):")
		fmt.Println(base58.Encode(bytes))
		return nil, nil
	} else {
		tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, instructions, feePayerPubkey, true)
		if err != nil {
			return nil, fmt.Errorf("NewTransaction failed, err: %s, tx: %s", err.Error(), tx.String())
		}

		privateKeyMap, err := utils.LoadPrivateKeysFromKeystore(keystorePath)
		if err != nil {
			return nil, err
		}

		feePayerAccount, exist := privateKeyMap[feePayerPubkey.String()]
		if !exist {
			return nil, fmt.Errorf("fee payer not exit in vault")
		}

		adminAccount, exist := privateKeyMap[adminPubkey.String()]
		if !exist {
			return nil, fmt.Errorf("admin not exit in vault")
		}

		if err = utils.SignAndSendTx(rpcClient, tx, utils.GetSignFunc(feePayerAccount, adminAccount), latestBlockHashRes.Value.LastValidBlockHeight); err != nil {
			return nil, fmt.Errorf("sign and send tx failed: %w", err)
		}
		fmt.Println(action, "tx hash:", tx.Signatures[0].String())
		return tx, nil
	}
}
