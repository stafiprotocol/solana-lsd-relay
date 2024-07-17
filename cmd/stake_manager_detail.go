package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-go-sdk/client"
)

func stakeManagerDetailCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "detail",
		Short: "Get stake manager detail",

		RunE: func(cmd *cobra.Command, args []string) error {

			stakeManager, err := cmd.Flags().GetString(flagStakeManager)
			if err != nil {
				return err
			}
			endpoint, err := cmd.Flags().GetString(flagEndPoint)
			if err != nil {
				return err
			}

			c := client.NewClient([]string{endpoint})

			stakeManagerDetail, err := c.GetLsdStakeManager(context.Background(), stakeManager)
			if err != nil {
				return err
			}

			jsonBts, err := json.MarshalIndent(stakeManagerDetail, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(jsonBts))
			return nil
		},
	}
	cmd.Flags().String(flagStakeManager, "", "stake manager")
	cmd.Flags().String(flagEndPoint, "", "solana rpc endpoint")
	return cmd
}
