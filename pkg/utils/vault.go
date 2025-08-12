package utils

import (
	"fmt"
	"os"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/cli"
	"github.com/gagliardetto/solana-go/vault"
)

func OpenVault(walletFile string) (*vault.Vault, vault.SecretBoxer, error) {
	v, err := vault.NewVaultFromWalletFile(walletFile)
	if err != nil {
		return nil, nil, fmt.Errorf("loading vault: %w", err)
	}

	boxer, err := vault.SecretBoxerForType(v.SecretBoxWrap, "")
	if err != nil {
		return nil, nil, fmt.Errorf("secret boxer: %w", err)
	}

	if err := v.Open(boxer); err != nil {
		return nil, nil, fmt.Errorf("opening: %w", err)
	}

	return v, boxer, nil
}

func LoadPrivateKeysFromKeystore(keystorePath string) (map[string]solana.PrivateKey, error) {
	v, _, err := OpenVault(keystorePath)
	if err != nil {
		return nil, fmt.Errorf("could not open keystore file '%s': %w.\nWARN: or do you miss --export flag?", keystorePath, err)
	}

	privateKeyMap := make(map[string]solana.PrivateKey)
	for _, privKey := range v.KeyBag {
		privateKeyMap[privKey.PublicKey().String()] = solana.PrivateKey(privKey)
	}

	return privateKeyMap, nil
}

func CreateBoxerIfNeeded(boxer vault.SecretBoxer) vault.SecretBoxer {
	if boxer != nil {
		return boxer
	}

	// create secret boxer
	fmt.Println("")
	fmt.Println("You will be asked to provide a passphrase to secure your newly created vault.")
	fmt.Println("Make sure you make it long and strong.")
	fmt.Println("")
	if envVal := os.Getenv("SLNC_GLOBAL_INSECURE_VAULT_PASSPHRASE"); envVal != "" {
		boxer = vault.NewPassphraseBoxer(envVal)
	} else {
		password, err := cli.GetEncryptPassphrase()
		if err != nil {
			fmt.Printf("ERROR: get password: %s\n", err)
			os.Exit(1)
		}

		boxer = vault.NewPassphraseBoxer(password)
	}

	return boxer
}
