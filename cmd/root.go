package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/stack"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/stake_manager"
)

var (
	appName = "solana-lsd-relay"
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
		stake_manager.InitCmd(),
		stake_manager.StartCmd(),
		stake_manager.DetailCmd(),
		stake_manager.TransferAdminCmd(),
		stake_manager.AcceptAdminCmd(),
		stake_manager.SetStakeManagerCmd(),
	)
	return cmd
}

func stackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stack",
		Short: "Stack operation",
	}

	cmd.AddCommand(
		stack.InitCmd(),
		stack.StartCmd(),
		stack.AddEntrustedStakeManager(),
		stack.RemoveEntrustedStakeManager(),
		stack.SetStackFee(),
		stack.TransferAdminCmd(),
		stack.AcceptAdminCmd(),
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
