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
	"os"

	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/vault"
)

func vaultCreateCmd() *cobra.Command {
	// vaultCreateCmd represents the create command
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new encrypted Solana keys vault",
		Long: `Create a new encrypted Solana keys vault.

A vault contains encrypted private keys, and with 'cmd', can be used to
securely sign transactions.

You can create a passphrase protected vault with:

    solana-lsd-relay keys create --keys=2

This uses the default --vault-type=passphrase

You can then use this vault for the different cmd operations.`,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			walletFile, err := cmd.Flags().GetString(flagKeystorePath)
			if err != nil {
				return err
			}

			if _, err := os.Stat(walletFile); err == nil {
				fmt.Printf("Wallet file %q already exists, rename it before running `cmd vault create`.\n", walletFile)
				os.Exit(1)
			}

			var boxer vault.SecretBoxer

			v := vault.NewVault()
			v.Comment, err = cmd.Flags().GetString("comment")
			if err != nil {
				return err
			}

			var newKeys []vault.PublicKey

			doImport, err := cmd.Flags().GetBool("import")
			if err != nil {
				return err
			}
			if doImport {
				privateKeys, err := capturePrivateKeys()
				if err != nil {
					return fmt.Errorf("failed enterign private key: %w", err)
				}

				for _, privateKey := range privateKeys {
					v.AddPrivateKey(privateKey)
					newKeys = append(newKeys, privateKey.PublicKey())
				}

				fmt.Printf("Imported %d keys.\n", len(newKeys))

			} else {
				numKeys, err := cmd.Flags().GetInt("keys")
				if err != nil {
					return err
				}

				if numKeys == 0 {
					return fmt.Errorf("specify either --keys or --import: create a vault with 0 keys?")
				}

				for i := 0; i < numKeys; i++ {
					pubKey, err := v.NewKeyPair()
					if err != nil {
						return fmt.Errorf("unable ot create new keypair: %w", err)
					}

					newKeys = append(newKeys, pubKey)
				}
				fmt.Printf("Created %d keys. They will be shown when encrypted and written to disk successfully.\n", len(newKeys))
			}

			fmt.Println("")
			fmt.Println("You will be asked to provide a passphrase to secure your newly created vault.")
			fmt.Println("Make sure you make it long and strong.")
			fmt.Println("")
			if envVal := os.Getenv("SLNC_GLOBAL_INSECURE_VAULT_PASSPHRASE"); envVal != "" {
				boxer = vault.NewPassphraseBoxer(envVal)
			} else {
				password, err := vault.GetEncryptPassphrase()
				if err != nil {
					return fmt.Errorf("failed to get password input: %w", err)
				}

				boxer = vault.NewPassphraseBoxer(password)
			}

			if err = v.Seal(boxer); err != nil {
				return fmt.Errorf("failed to seal the vault: %w", err)
			}

			if err = v.WriteToFile(walletFile); err != nil {
				return fmt.Errorf("failed to write vault file: %w", err)
			}

			vaultWrittenReport(walletFile, newKeys, len(v.KeyBag))
			return nil
		},
	}

	cmd.Flags().IntP("keys", "k", 0, "Number of keypairs to create")
	cmd.Flags().BoolP("import", "i", false, "Whether to import keys instead of creating them. This takes precedence over --keys, and private keys will be inputted on the command line.")
	cmd.Flags().StringP("comment", "", "", "Comment field in the vault's json file.")
	cmd.Flags().StringP(flagKeystorePath, "", defaultKeystorePath, "Wallet file that contains encrypted key material")

	return cmd
}
