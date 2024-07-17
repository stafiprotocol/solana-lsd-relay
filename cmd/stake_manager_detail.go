package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/lsdprog"
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

			stakeManagerPubkey := common.PublicKeyFromString(stakeManager)

			endpoint, err := cmd.Flags().GetString(flagEndPoint)
			if err != nil {
				return err
			}

			c := client.NewClient([]string{endpoint})
			accountInfo, err := c.GetAccountInfo(context.Background(), stakeManager, client.GetAccountInfoConfig{
				Encoding: client.GetAccountInfoConfigEncodingBase64,
				DataSlice: client.GetAccountInfoConfigDataSlice{
					Offset: 0,
					Length: lsdprog.StakeManagerAccountLengthDefault,
				},
			})
			if err != nil {
				return err
			}
			stakeManagerDetail, err := c.GetLsdStakeManager(context.Background(), stakeManager)
			if err != nil {
				return err
			}

			programId := common.PublicKeyFromString(accountInfo.Owner)
			stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, programId)
			if err != nil {
				return err
			}

			jsonBts, err := json.MarshalIndent(stakeManagerDetail, "", "  ")
			if err != nil {
				return err
			}

			fmt.Printf("stakeManager: \n%s\n", string(jsonBts))
			fmt.Printf("stakePool: \n%s\n", stakePool.ToBase58())
			return nil
		},
	}
	cmd.Flags().String(flagStakeManager, "", "stake manager")
	cmd.Flags().String(flagEndPoint, "", "solana rpc endpoint")
	return cmd
}
