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

package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const defaultRPCURL = "http://api.mainnet-beta.solana.com/rpc"

func GetPassword(input string) (string, error) {
	fd := os.Stdin.Fd()
	fmt.Print(input)
	pass, err := term.ReadPassword(int(fd))
	fmt.Println("")
	return string(pass), err
}

func errorCheck(prefix string, err error) {
	if err != nil {
		fmt.Printf("ERROR: %s: %s\n", prefix, err)
		if strings.HasSuffix(err.Error(), "connection refused") && strings.Contains(err.Error(), defaultRPCURL) {
			fmt.Println("Have you selected a valid Solana JSON-RPC endpoint ? You can use the --rpc-url flag or SLNC_GLOBAL_RPC_URL environment variable.")
		}
		os.Exit(1)
	}
}

func MustGetWallet(cmd *cobra.Command, create bool) (*Vault, SecretBoxer) {
	if create {
		walletFile, err := cmd.Flags().GetString("keystore_path")
		if err != nil {
			errorCheck("wallet create", err)
		}
		if _, err := os.Stat(walletFile); err != nil {
			vault, err := createVault(walletFile)
			errorCheck("wallet create", err)
			return vault, nil
		}
	}
	vault, boxer, err := openWallet(cmd)
	errorCheck("wallet open", err)
	return vault, boxer
}

func createVault(walletFile string) (*Vault, error) {
	// create directory if not exist
	dir := filepath.Dir(walletFile)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create directory %s error: %w", dir, err)
	}

	return NewVault(), nil
}

func openWallet(cmd *cobra.Command) (*Vault, SecretBoxer, error) {
	walletFile, err := cmd.Flags().GetString("keystore_path")
	if err != nil {
		return nil, nil, err
	}
	if _, err := os.Stat(walletFile); err != nil {
		return nil, nil, fmt.Errorf("wallet file %q missing: %w", walletFile, err)
	}

	v, err := NewVaultFromWalletFile(walletFile)
	if err != nil {
		return nil, nil, fmt.Errorf("loading vault: %w", err)
	}

	boxer, err := SecretBoxerForType(v.SecretBoxWrap)
	if err != nil {
		return nil, nil, fmt.Errorf("secret boxer: %w", err)
	}

	if err := v.Open(boxer); err != nil {
		return nil, nil, fmt.Errorf("opening: %w", err)
	}

	return v, boxer, nil
}

func GetDecryptPassphrase() (string, error) {
	if envVal := os.Getenv("SLNC_GLOBAL_INSECURE_VAULT_PASSPHRASE"); envVal != "" {
		return envVal, nil
	}

	passphrase, err := GetPassword("Enter passphrase to decrypt your solana vault: ")
	if err != nil {
		return "", fmt.Errorf("reading password: %s", err)
	}

	return passphrase, nil
}

func GetEncryptPassphrase() (string, error) {
	passphrase, err := GetPassword("Enter passphrase to encrypt your solana vault: ")
	if err != nil {
		return "", fmt.Errorf("reading password: %s", err)
	}

	passphraseConfirm, err := GetPassword("Confirm passphrase: ")
	if err != nil {
		return "", fmt.Errorf("reading confirmation password: %s", err)
	}

	if passphrase != passphraseConfirm {
		fmt.Println()
		return "", errors.New("passphrase mismatch")
	}
	return passphrase, nil

}

func WrittenReport(walletFile string, newKeys []PublicKey, totalKeys int) {
	fmt.Println("")
	fmt.Printf("Wallet file %q written to disk.\n", walletFile)
	if totalKeys > 0 {
		fmt.Println("Here are the keys that were ADDED during this operation (use `list` to see them all):")
		for _, pub := range newKeys {
			fmt.Printf("- %s\n", pub.String())
		}

		fmt.Printf("Total keys stored: %d\n", totalKeys)
	}
}

func CreateBoxerIfNeeded(boxer SecretBoxer) SecretBoxer {
	if boxer != nil {
		return boxer
	}

	// create secret boxer
	fmt.Println("")
	fmt.Println("You will be asked to provide a passphrase to secure your newly created vault.")
	fmt.Println("Make sure you make it long and strong.")
	fmt.Println("")
	if envVal := os.Getenv("SLNC_GLOBAL_INSECURE_VAULT_PASSPHRASE"); envVal != "" {
		boxer = NewPassphraseBoxer(envVal)
	} else {
		password, err := GetEncryptPassphrase()
		errorCheck("get password: ", err)
		boxer = NewPassphraseBoxer(password)
	}

	return boxer
}
