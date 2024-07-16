package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-go-sdk/common"
)

func getStakePoolCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "get-stake-pool",
		Short: "Get stake pool address",

		RunE: func(cmd *cobra.Command, args []string) error {

			stakeManager, err := cmd.Flags().GetString(flagStakeManager)
			if err != nil {
				return err
			}
			lsdProgramIDStr, err := cmd.Flags().GetString(flagLsdProgramID)
			if err != nil {
				return err
			}

			stakeManagerPubkey := common.PublicKeyFromString(stakeManager)
			lsdProgramID := common.PublicKeyFromString(lsdProgramIDStr)

			stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, lsdProgramID)
			if err != nil {
				return err
			}

			fmt.Println("lsdProgramID:", lsdProgramID.ToBase58())
			fmt.Println("stakeManager:", stakeManagerPubkey.ToBase58())
			fmt.Println("stakePool:", stakePool.ToBase58())

			return nil
		},
	}
	cmd.Flags().String(flagStakeManager, "", "stake manager")
	cmd.Flags().String(flagLsdProgramID, "", "lsd program id")
	return cmd
}
