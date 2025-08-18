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
	"path/filepath"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/cli"
	"github.com/gagliardetto/solana-go/vault"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

func vaultImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import private keys taking input from the shell",
		RunE: func(cmd *cobra.Command, args []string) error {
			walletFile, err := cmd.Flags().GetString(common.FlagKeystorePath)
			if err != nil {
				return err
			}

			v, boxer := mustGetWallet(cmd, true)
			if len(v.KeyBag) > 0 {
				v.PrintPublicKeys()
			}

			privateKeys, err := capturePrivateKeys()
			if err != nil {
				fmt.Printf("failed to enter private keys: %s", err)
				return err
			}
			if len(privateKeys) == 0 {
				fmt.Println("quit: no private keys")
				return nil
			}

			var newKeys []solana.PublicKey
			for _, privateKey := range privateKeys {
				v.AddPrivateKey(privateKey)
				newKeys = append(newKeys, privateKey.PublicKey())
			}

			if err = v.Seal(utils.CreateBoxerIfNeeded(boxer)); err != nil {
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

	cmd.Flags().StringP(common.FlagKeystorePath, "", common.DefaultKeystorePath, "Wallet file that contains encrypted key material")
	return cmd
}

func mustGetWallet(cmd *cobra.Command, create bool) (*vault.Vault, vault.SecretBoxer) {
	if create {
		walletFile, err := cmd.Flags().GetString(common.FlagKeystorePath)
		exitOnError("wallet create", err)

		if _, err := os.Stat(walletFile); err != nil {
			// create directory if not exist
			dir := filepath.Dir(walletFile)
			err = os.MkdirAll(dir, 0750)
			exitOnError(fmt.Sprintf("create directory %s error", dir), err)

			return vault.NewVault(), nil
		}
	}

	walletFile, err := cmd.Flags().GetString(common.FlagKeystorePath)
	if err != nil {
		exitOnError("get keystore path", err)
	}
	_, err = os.Stat(walletFile)
	exitOnError(fmt.Sprintf("wallet file %q missing", walletFile), err)

	vault, boxer, err := utils.OpenVault(walletFile)
	exitOnError("wallet open", err)
	return vault, boxer
}

func exitOnError(msg string, err error) {
	if err != nil {
		fmt.Printf("ERROR: %s: %s\n", msg, err.Error())
		os.Exit(1)
	}
}

func capturePrivateKeys() (out []solana.PrivateKey, err error) {
	fmt.Println("")
	fmt.Println("PLEASE READ:")
	fmt.Println("We are now going to ask you to paste your private keys, one at a time.")
	fmt.Println("They will not be shown on screen.")
	fmt.Println("Please verify that the public keys printed on screen correspond to what you have noted")
	fmt.Println("")

	first := true
	for {
		privKey, err := capturePrivateKey(first)
		if err != nil {
			return out, fmt.Errorf("capture privkeys: %s", err)
		}
		first = false

		if privKey == nil {
			return out, nil
		}
		out = append(out, privKey)
	}
}

func capturePrivateKey(isFirst bool) (privateKey solana.PrivateKey, err error) {
	prompt := "Paste your first private key: "
	if !isFirst {
		prompt = "Paste your next private key or hit ENTER if you are done: "
	}

	enteredKey, err := cli.GetPassword(prompt)
	if err != nil {
		return nil, fmt.Errorf("get private key: %s", err)
	}

	if enteredKey == "" {
		return nil, nil
	}

	key, err := solana.PrivateKeyFromBase58(enteredKey)
	if err != nil {
		return nil, fmt.Errorf("import private key: %s", err)
	}

	fmt.Printf("- Scanned private key corresponding to %s\n", key.PublicKey().String())

	return key, nil
}

func vaultWrittenReport(walletFile string, newKeys []solana.PublicKey, totalKeys int) {
	fmt.Println("")
	fmt.Printf("Wallet file %q written to disk.\n", walletFile)
	fmt.Println("Here are the keys that were ADDED during this operation (use `list` to see them all):")
	for _, pub := range newKeys {
		fmt.Printf("- %s\n", pub.String())
	}

	fmt.Printf("Total keys stored: %d\n", totalKeys)
}
