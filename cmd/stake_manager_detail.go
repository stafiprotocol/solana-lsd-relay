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

			lsdTokenMintAccount, err := rpcClient.GetAccountInfo(context.Background(), stakeManager.LsdTokenMint)
			if err != nil {
				return err
			}

			var tokenProgramAccount string
			if lsdTokenMintAccount.Value.Owner == solana.Token2022ProgramID {
				tokenProgramAccount = solana.Token2022ProgramID.String() + " (Token2022)"
			} else if lsdTokenMintAccount.Value.Owner == solana.TokenProgramID {
				tokenProgramAccount = solana.TokenProgramID.String() + " (Token)"
			} else {
				return fmt.Errorf("lsd token mint account owner is not token2022 or token program")
			}

			fmt.Printf("stakePool: %s\n", stakePool)
			fmt.Printf("stakeManager: \n%s\n", string(jsonBts))
			fmt.Printf("LSD Token Program ID: %s\n", tokenProgramAccount)
			return nil
		},
	}
	cmd.Flags().String(flagStakeManager, "", "stake manager")
	cmd.Flags().String(flagEndPoint, "", "solana rpc endpoint")
	return cmd
}
