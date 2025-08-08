package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

func stakeManagerDetailCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "detail",
		Short: "Get stake manager detail",

		RunE: func(cmd *cobra.Command, args []string) error {
			stakeManagerStr, err := cmd.Flags().GetString(flagStakeManager)
			if err != nil {
				return err
			}
			stakeManagerPubkey := solana.MustPublicKeyFromBase58(stakeManagerStr)

			endpoint, err := cmd.Flags().GetString(flagEndPoint)
			if err != nil {
				return err
			}

			rpcClient := rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
				endpoint,
				rate.Every(time.Second), // time frame
				5,                       // limit of requests per time frame
			))

			getAccountInfoRes, err := rpcClient.GetAccountInfo(context.Background(), stakeManagerPubkey)
			if err != nil {
				return err
			}
			stakeManager := lsd_program.StakeManager{}
			err = utils.GetAndDecodeAccountInfo(rpcClient, stakeManagerPubkey, &stakeManager)
			if err != nil {
				return err
			}

			programId := getAccountInfoRes.Value.Owner
			fmt.Printf("programId: %s\n", programId)
			stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, programId)
			if err != nil {
				return err
			}

			jsonBts, err := json.MarshalIndent(stakeManager, "", "  ")
			if err != nil {
				return err
			}

			fmt.Printf("stakePool: %s\n", stakePool)
			fmt.Printf("stakeManager: \n%s\n", string(jsonBts))
			return nil
		},
	}
	cmd.Flags().String(flagStakeManager, "", "stake manager")
	cmd.Flags().String(flagEndPoint, "", "solana rpc endpoint")
	return cmd
}
