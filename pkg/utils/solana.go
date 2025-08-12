package utils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/decred/base58"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	computebudget "github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sirupsen/logrus"
)

var ErrExpired = fmt.Errorf("expired")

func SendAndWaitForConfirmation(rpcClient *rpc.Client, tx *solana.Transaction,
	lastValidBlockHeight uint64) (*rpc.GetTransactionResult, error) {

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	opts := rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: rpc.CommitmentConfirmed,
	}

	sig, err := rpcClient.SendTransactionWithOpts(ctx, tx, opts)
	if err != nil {
		return nil, fmt.Errorf("SendTransactionWithOpts failed, err: %s, tx: %s", err.Error(), tx.String())
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			nowBlockHeight, err := rpcClient.GetBlockHeight(ctx, rpc.CommitmentConfirmed)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			txRes, err := rpcClient.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
				Commitment:                     rpc.CommitmentConfirmed,
				MaxSupportedTransactionVersion: &rpc.MaxSupportedTransactionVersion1,
			})
			if err != nil {
				if nowBlockHeight > lastValidBlockHeight {
					return nil, ErrExpired
				}

				rpcClient.SendTransactionWithOpts(ctx, tx, opts)

				time.Sleep(time.Second)
				continue
			}

			if txRes.Meta.Err != nil {
				errString := ""
				for _, log := range txRes.Meta.LogMessages {
					if strings.Contains(log, "Error") || strings.Contains(log, "error") {
						errString += fmt.Sprintf(" log: %s", log)
					}
				}

				return nil, fmt.Errorf("tx execute failed: %v, logs: %s", txRes.Meta.Err, errString)
			}
			return txRes, nil
		}
	}
}

func SignAndSendTx(
	rpcClient *rpc.Client,
	tx *solana.Transaction,
	signFunc func(key solana.PublicKey) *solana.PrivateKey,
	lastValidBlockHeight uint64,
) error {
	if logrus.GetLevel() == logrus.DebugLevel || logrus.GetLevel() == logrus.TraceLevel {
		bytes, err := tx.Message.MarshalBinary()
		if err != nil {
			return fmt.Errorf("fail to marshal tx.Message: %w", err)
		}
		logrus.Debugf("raw base58 encoded transaction message: %s", base58.Encode(bytes))
	}

	_, err := tx.Sign(signFunc)
	if err != nil {
		return fmt.Errorf("sign failed, err: %s", err.Error())
	}

	_, err = SendAndWaitForConfirmation(rpcClient, tx, lastValidBlockHeight)
	if err != nil {
		return fmt.Errorf("waitForConfirmation error, err: %s", err.Error())
	}
	return nil
}

func NewSolanaTransaction(recentBlockHash solana.Hash, instructions []solana.Instruction,
	feePayerPublicKey solana.PublicKey, addUnitPriceIns bool) (*solana.Transaction, error) {
	if addUnitPriceIns {
		instructions = append([]solana.Instruction{computebudget.NewSetComputeUnitPriceInstruction(20000).Build()}, instructions...)
	}
	return solana.NewTransaction(instructions, recentBlockHash, solana.TransactionPayer(feePayerPublicKey))
}

func GetAndDecodeAccountInfo(rpcClient *rpc.Client, account solana.PublicKey, v any) error {
	accountInfo, err := rpcClient.GetAccountInfoWithOpts(context.Background(), account, &rpc.GetAccountInfoOpts{
		Encoding:   solana.EncodingBase64,
		Commitment: rpc.CommitmentConfirmed,
	})
	if err != nil {
		return err
	}

	return bin.NewBorshDecoder(accountInfo.Value.Data.GetBinary()).Decode(v)
}

func GetSignFunc(feePayerAccount solana.PrivateKey, otherAccounts ...solana.PrivateKey) func(key solana.PublicKey) *solana.PrivateKey {
	return func(key solana.PublicKey) *solana.PrivateKey {
		if feePayerAccount.PublicKey().Equals(key) {
			return &feePayerAccount
		}
		for _, account := range otherAccounts {
			if account.PublicKey().Equals(key) {
				return &account
			}
		}
		return nil
	}
}

func GetMinDelegationAmount(rpcClient *rpc.Client) (uint64, error) {
	res := struct {
		Value uint64 `json:"value"`
	}{}
	err := rpcClient.RPCCallForInto(context.Background(), &res, "getStakeMinimumDelegation", nil)
	return res.Value, err
}
