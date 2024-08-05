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

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/vault"
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

			walletFile, err := cmd.Flags().GetString(flagKeystorePath)
			if err != nil {
				return err
			}

			v, boxer := vault.MustGetWallet(cmd, true)
			if len(v.KeyBag) > 0 {
				v.PrintPublicKeys()
			}

			privateKeys := make([]vault.PrivateKey, 0)
			for i := 0; i < numKeys; i++ {
				_, privKey, err := vault.NewRandomPrivateKey()
				if err != nil {
					return err
				}
				privateKeys = append(privateKeys, privKey)
			}

			var newKeys []vault.PublicKey
			for _, privateKey := range privateKeys {
				v.AddPrivateKey(privateKey)
				newKeys = append(newKeys, privateKey.PublicKey())
			}

			if err = v.Seal(vault.CreateBoxerIfNeeded(boxer)); err != nil {
				fmt.Printf("seal err: %s", err)
				return err
			}

			err = v.WriteToFile(walletFile)
			if err != nil {
				fmt.Printf("failed to write vault file: %s", err)
				return err
			}

			vault.WrittenReport(walletFile, newKeys, len(v.KeyBag))
			return nil
		},
	}
	cmd.Flags().IntP("keys", "k", 0, "Number of keypairs to create")
	cmd.Flags().StringP(flagKeystorePath, "", defaultKeystorePath, "Wallet file that contains encrypted key material")
	return cmd
}
