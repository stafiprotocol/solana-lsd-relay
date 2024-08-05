package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
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
