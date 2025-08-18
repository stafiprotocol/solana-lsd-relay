// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"

	"github.com/gagliardetto/solana-go"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

func vaultGenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gen",
		Short: "Gen new keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			numKeys, err := cmd.Flags().GetInt("keys")
			if err != nil {
				return err
			}

			if numKeys == 0 {
				return fmt.Errorf("specify --keys")
			}

			walletFile, err := cmd.Flags().GetString(common.FlagKeystorePath)
			if err != nil {
				return err
			}

			v, boxer := mustGetWallet(cmd, true)
			if len(v.KeyBag) > 0 {
				v.PrintPublicKeys()
			}

			privateKeys := make([]solana.PrivateKey, 0)
			for i := 0; i < numKeys; i++ {
				privKey, err := solana.NewRandomPrivateKey()
				if err != nil {
					return err
				}
				privateKeys = append(privateKeys, privKey)
			}

			var newKeys []solana.PublicKey
			for _, privateKey := range privateKeys {
				v.AddPrivateKey(privateKey)
				newKeys = append(newKeys, privateKey.PublicKey())
			}

			if err = v.Seal(utils.CreateBoxerIfNeeded(boxer)); err != nil {
				fmt.Printf("seal err: %s", err)
				return err
			}

			err = v.WriteToFile(walletFile)
			if err != nil {
				fmt.Printf("failed to write vault file: %s", err)
				return err
			}

			vaultWrittenReport(walletFile, newKeys, len(v.KeyBag))
			return nil
		},
	}
	cmd.Flags().IntP("keys", "k", 0, "Number of keypairs to create")
	cmd.Flags().StringP(common.FlagKeystorePath, "", common.DefaultKeystorePath, "Wallet file that contains encrypted key material")
	return cmd
}
