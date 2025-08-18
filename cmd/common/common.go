package common

import (
	"context"
	"fmt"

	"github.com/decred/base58"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

const (
	FlagLogLevel     = "log_level"
	FlagConfigPath   = "config"
	FlagFeePayer     = "fee_payer"
	FlagStakeManager = "stake_manager"
	FlagEndPoint     = "endpoint"
	FlagLsdProgramID = "lsd_program_id"
	FlagKeystorePath = "keystore_path"
	FlagExportTx     = "export"

	DefaultKeystorePath = "./keys/solana_keys.json"
	DefaultConfigPath   = "./config.toml"
	DefaultLogDir       = "./log_data"
)

func AdminExecuteInstructions(
	action string,
	rpcClient *rpc.Client,
	instructions []solana.Instruction,
	keystorePath string,
	feePayerPubkey solana.PublicKey,
	adminPubkey solana.PublicKey,
	exportTxMessage bool,
) (*solana.Transaction, error) {
	latestBlockHashRes, err := rpcClient.GetLatestBlockhash(context.Background(), rpc.CommitmentConfirmed)
	if err != nil {
		return nil, fmt.Errorf("get recent block hash error: %w", err)
	}

	if exportTxMessage {
		tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, instructions, feePayerPubkey, false)
		if err != nil {
			return nil, fmt.Errorf("NewTransaction failed, err: %s, tx: %s", err.Error(), tx.String())
		}
		bytes, err := tx.Message.MarshalBinary()
		if err != nil {
			return tx, fmt.Errorf("fail to marshal tx.Message: %w", err)
		}
		fmt.Println(action, "tx(base58):")
		fmt.Println(base58.Encode(bytes))
		return nil, nil
	} else {
		tx, err := utils.NewSolanaTransaction(latestBlockHashRes.Value.Blockhash, instructions, feePayerPubkey, true)
		if err != nil {
			return nil, fmt.Errorf("NewTransaction failed, err: %s, tx: %s", err.Error(), tx.String())
		}

		privateKeyMap, err := utils.LoadPrivateKeysFromKeystore(keystorePath)
		if err != nil {
			return nil, err
		}

		feePayerAccount, exist := privateKeyMap[feePayerPubkey.String()]
		if !exist {
			return nil, fmt.Errorf("fee payer not exit in vault")
		}

		adminAccount, exist := privateKeyMap[adminPubkey.String()]
		if !exist {
			return nil, fmt.Errorf("admin not exit in vault")
		}

		if err = utils.SignAndSendTx(rpcClient, tx, utils.GetSignFunc(feePayerAccount, adminAccount), latestBlockHashRes.Value.LastValidBlockHeight); err != nil {
			return nil, fmt.Errorf("sign and send tx failed: %w", err)
		}
		fmt.Println(action, "tx hash:", tx.Signatures[0].String())
		return tx, nil
	}
}
