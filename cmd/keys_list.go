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
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/vault"
)

func vaultListCmd() *cobra.Command {

	// vaultListCmd represents the list command
	var cmd = &cobra.Command{
		Use:   "list",
		Short: "List public keys inside a Solana vault.",
		Long: `List public keys inside a Solana vault.

The wallet file contains a lits of public keys for easy reference, but
you cannot trust that these public keys have their counterpart in the
wallet, unless you check with the "list" command.
`,
		Run: func(cmd *cobra.Command, args []string) {
			vault, _ := vault.MustGetWallet(cmd, false)

			vault.PrintPublicKeys()
		},
	}
	cmd.Flags().StringP(flagKeystorePath, "", defaultKeystorePath, "Wallet file that contains encrypted key material")
	return cmd
}
