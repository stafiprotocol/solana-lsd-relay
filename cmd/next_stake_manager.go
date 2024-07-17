package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
)

func nextStakeManagerCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "next-stake-manager",
		Short: "Get next stake manager info",

		RunE: func(cmd *cobra.Command, args []string) error {

			feePayer, err := cmd.Flags().GetString(flagFeePayer)
			if err != nil {
				return err
			}
			endpoint, err := cmd.Flags().GetString(flagEndPoint)
			if err != nil {
				return err
			}
			lsdProgramIDStr, err := cmd.Flags().GetString(flagLsdProgramID)
			if err != nil {
				return err
			}

			c := client.NewClient([]string{endpoint})

			feePayerPubkey := common.PublicKeyFromString(feePayer)
			lsdProgramID := common.PublicKeyFromString(lsdProgramIDStr)

			var stakeManagerPubkey common.PublicKey
			var seed string
			index := 0
			for i := 0; ; i++ {
				index = i
				seed = fmt.Sprintf(stakeManagerSeed, index)
				stakeManagerPubkey = common.CreateWithSeed(feePayerPubkey, seed, lsdProgramID)
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

			stakePool, _, err := common.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, lsdProgramID)
			if err != nil {
				return err
			}

			fmt.Println("lsdProgramID:", lsdProgramID.ToBase58())
			fmt.Println("stakeManager:", stakeManagerPubkey.ToBase58())
			fmt.Println("stakePool:", stakePool.ToBase58())
			fmt.Println("index:", index)

			return nil
		},
	}
	cmd.Flags().String(flagFeePayer, "", "fee payer")
	cmd.Flags().String(flagEndPoint, "", "solana rpc endpoint")
	cmd.Flags().String(flagLsdProgramID, "", "lsd program id")
	return cmd
}
