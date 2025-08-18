package stake_manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stake_manager"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

func DetailCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "detail",
		Short: "Get stake manager detail",

		RunE: func(cmd *cobra.Command, args []string) error {
			stakeManagerStr, err := cmd.Flags().GetString(common.FlagStakeManager)
			if err != nil {
				return err
			}
			stakeManagerPubkey := solana.MustPublicKeyFromBase58(stakeManagerStr)

			endpoint, err := cmd.Flags().GetString(common.FlagEndPoint)
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
			stakeManager := stake_manager.StakeManager{}
			err = utils.GetAndDecodeAccountInfo(rpcClient, stakeManagerPubkey, &stakeManager)
			if err != nil {
				return err
			}

			programId := getAccountInfoRes.Value.Owner
			fmt.Printf("programId: %s\n", programId)
			stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), utils.StakePoolSeed}, programId)
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
			switch lsdTokenMintAccount.Value.Owner {
			case solana.Token2022ProgramID:
				tokenProgramAccount = solana.Token2022ProgramID.String() + " (Token2022)"
			case solana.TokenProgramID:
				tokenProgramAccount = solana.TokenProgramID.String() + " (Token)"
			default:
				return fmt.Errorf("lsd token mint account owner is not token2022 or token program")
			}

			fmt.Printf("stakePool: %s\n", stakePool)
			fmt.Printf("stakeManager: \n%s\n", string(jsonBts))
			fmt.Printf("LSD Token Program ID: %s\n", tokenProgramAccount)
			return nil
		},
	}
	cmd.Flags().String(common.FlagStakeManager, "", "stake manager")
	cmd.Flags().String(common.FlagEndPoint, "", "solana rpc endpoint")
	return cmd
}
