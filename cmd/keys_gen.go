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
		Short: "Gen new keys to an existing vault",
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

			fmt.Println("Loading existing vault from file:", walletFile)
			v, err := vault.NewVaultFromWalletFile(walletFile)
			if err != nil {
				fmt.Printf("unable to load vault file: %s", err)
				return err
			}

			boxer, err := vault.SecretBoxerForType(v.SecretBoxWrap)
			if err != nil {
				fmt.Printf("unable to intiate boxer: %s", err)
				return err
			}

			err = v.Open(boxer)
			if err != nil {
				fmt.Printf("unable to open vault: %s", err)
				return err
			}

			v.PrintPublicKeys()

			privateKeys := make([]vault.PrivateKey, 0)
			for i := 0; i < numKeys; i++ {
				_, privKey, err := vault.NewRandomPrivateKey()
				if err != nil {
					return err
				}
				privateKeys = append(privateKeys, privKey)
			}
			fmt.Printf("Created %d keys. They will be shown when encrypted and written to disk successfully.\n", len(privateKeys))

			var newKeys []vault.PublicKey
			for _, privateKey := range privateKeys {
				v.AddPrivateKey(privateKey)
				newKeys = append(newKeys, privateKey.PublicKey())
			}

			err = v.Seal(boxer)
			if err != nil {
				fmt.Printf("failed to seal vault: %s", err)
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
	cmd.Flags().StringP(flagKeystorePath, "", defaultKeystorePath, "Wallet file that contains encrypted key material")
	return cmd
}
