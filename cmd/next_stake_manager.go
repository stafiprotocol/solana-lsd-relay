package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"golang.org/x/time/rate"
)

func nextStakeManagerCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "next-stake-manager",
		Short: "Get next stake manager info",

		RunE: func(cmd *cobra.Command, args []string) error {
			feePayerStr, err := cmd.Flags().GetString(flagFeePayer)
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

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			feePayerPubkey := solana.MustPublicKeyFromBase58(feePayerStr)
			lsdProgramID := solana.MustPublicKeyFromBase58(lsdProgramIDStr)

			var stakeManagerPubkey solana.PublicKey
			var seed string
			index := 0
			for i := 0; ; i++ {
				index = i
				seed = fmt.Sprintf(stakeManagerSeed, index)
				stakeManagerPubkey, err = solana.CreateWithSeed(feePayerPubkey, seed, lsdProgramID)
				if err != nil {
					return fmt.Errorf("CreateWithSeed failed, err: %w", err)
				}
				_, err = rpcClient.GetAccountInfo(context.Background(), stakeManagerPubkey)
				if err != nil {
					if err == rpc.ErrNotFound {
						break
					} else {
						return err
					}
				}
			}

			stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, lsdProgramID)
			if err != nil {
				return err
			}

			fmt.Println("lsdProgramID:", lsdProgramID)
			fmt.Println("stakeManager:", stakeManagerPubkey)
			fmt.Println("stakePool:", stakePool)
			fmt.Println("index:", index)

			return nil
		},
	}
	cmd.Flags().String(flagFeePayer, "", "fee payer")
	cmd.Flags().String(flagEndPoint, "", "solana rpc endpoint")
	cmd.Flags().String(flagLsdProgramID, "", "lsd program id")
	return cmd
}
