package utils

import (
	"context"
	"fmt"
	"math"
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

type StakeActivationState string

const (
	StakeActivationStateActive       StakeActivationState = "active"
	StakeActivationStateInactive     StakeActivationState = "inactive"
	StakeActivationStateActivating   StakeActivationState = "activating"
	StakeActivationStateDeactivating StakeActivationState = "deactivating"
)

type GetStakeActivationResponse struct {
	State    StakeActivationState `json:"state"`
	Active   uint64               `json:"active"`
	Inactive uint64               `json:"inactive"`
}

type StakeAccount struct {
	Type uint32 //0 uninitialized 1 initialized 2 delegated 3 rewardspool
	Info struct {
		Meta struct {
			RentExemptReserve int64
			Authorized        struct {
				Staker     solana.PublicKey
				Withdrawer solana.PublicKey
				Lockup     struct {
					UnixTimeStamp int64
					Epoch         uint64
					Custodian     solana.PublicKey
				}
			}
		}
		Stake struct {
			Delegation      Delegation
			CreditsObserved uint64
		}
	}
}

type Delegation struct {
	Voter              solana.PublicKey
	Stake              uint64
	ActivationEpoch    uint64 //epoch when delegate
	DeactivationEpoch  uint64 //epoch when deactive
	WarmupCooldownRate float64
}

const NEW_WARMUP_COOLDOWN_RATE = float64(0.09)

func getHistory(h []StakeHistory, epoch uint64) *StakeHistoryEntry {
	for _, his := range h {
		if his.Epoch == epoch {
			return &his.Entry
		}
	}
	return nil
}

// returned tuple is (effective, activating) stake
func (d *Delegation) StakeAndActivating(targetEpoch uint64, histories []StakeHistory) (uint64, uint64) {
	if d.ActivationEpoch == math.MaxUint64 {
		return d.Stake, 0
	}
	if d.ActivationEpoch == d.DeactivationEpoch {
		return 0, 0
	}
	if targetEpoch == d.ActivationEpoch {
		return 0, d.Stake
	}
	if targetEpoch < d.ActivationEpoch {
		return 0, 0
	}

	targetEntry := getHistory(histories, targetEpoch)
	if targetEntry != nil {

		prev_epoch := d.ActivationEpoch
		prev_cluster_stake := targetEntry

		current_epoch := uint64(0)
		current_effective_stake := uint64(0)
		for {
			current_epoch = prev_epoch + 1
			// if there is no activating stake at prev epoch, we should have been
			// fully effective at this moment
			if prev_cluster_stake.Activating == 0 {
				break
			}

			// how much of the growth in stake this account is
			//  entitled to take
			remaining_activating_stake := d.Stake - current_effective_stake
			weight := float64(remaining_activating_stake) / float64(prev_cluster_stake.Activating)
			warmup_cooldown_rate := NEW_WARMUP_COOLDOWN_RATE

			// // portion of newly effective cluster stake I'm entitled to at current epoch
			newly_effective_cluster_stake := float64(prev_cluster_stake.Effective) * warmup_cooldown_rate
			newly_effective_stake := uint64((weight * newly_effective_cluster_stake))
			if newly_effective_stake < 1 {
				newly_effective_stake = 1
			}

			current_effective_stake += newly_effective_stake
			if current_effective_stake >= d.Stake {
				current_effective_stake = d.Stake
				break
			}

			if current_epoch >= targetEpoch || current_epoch >= d.DeactivationEpoch {
				break
			}

			current_cluster_stake := getHistory(histories, current_epoch)
			if current_cluster_stake != nil {
				prev_epoch = current_epoch
				prev_cluster_stake = current_cluster_stake

			} else {
				break
			}
		}
		return current_effective_stake, d.Stake - current_effective_stake

	} else {
		return d.Stake, 0
	}
}

func (d *Delegation) StakeActivatingAndDeactivating(target_epoch uint64, histories []StakeHistory) StakeHistoryEntry {
	effective_stake, activating_stake := d.StakeAndActivating(target_epoch, histories)
	if target_epoch < d.DeactivationEpoch {
		// not deactivated
		if activating_stake == 0 {
			// StakeActivationStatus::with_effective(effective_stake)
			return StakeHistoryEntry{
				Effective: effective_stake,
			}
		} else {
			return StakeHistoryEntry{
				Effective:  effective_stake,
				Activating: activating_stake,
			}
		}
	}
	if target_epoch == d.DeactivationEpoch {
		return StakeHistoryEntry{
			Deactivating: effective_stake,
		}
	}

	targetEntry := getHistory(histories, d.DeactivationEpoch)
	if targetEntry != nil {
		prev_epoch := d.DeactivationEpoch
		prev_cluster_stake := targetEntry

		current_epoch := uint64(0)
		current_effective_stake := effective_stake
		for {
			current_epoch = prev_epoch + 1
			// if there is no deactivating stake at prev epoch, we should have been
			// fully undelegated at this moment
			if prev_cluster_stake.Deactivating == 0 {
				break
			}

			// I'm trying to get to zero, how much of the deactivation in stake
			//   this account is entitled to take
			weight := float64(current_effective_stake) / float64(prev_cluster_stake.Deactivating)
			warmup_cooldown_rate := NEW_WARMUP_COOLDOWN_RATE

			// portion of newly not-effective cluster stake I'm entitled to at current epoch
			newly_not_effective_cluster_stake := float64(prev_cluster_stake.Effective) * warmup_cooldown_rate
			newly_not_effective_stake := uint64(weight * newly_not_effective_cluster_stake)
			if newly_not_effective_stake < 1 {
				newly_not_effective_stake = 1
			}

			if current_effective_stake > newly_not_effective_stake {
				current_effective_stake = current_effective_stake - newly_not_effective_stake
			} else {
				current_effective_stake = 0
			}

			if current_effective_stake == 0 {
				break
			}

			if current_epoch >= target_epoch {
				break
			}
			current_cluster_stake := getHistory(histories, current_epoch)
			if current_cluster_stake != nil {
				prev_epoch = current_epoch
				prev_cluster_stake = current_cluster_stake
			} else {
				break
			}
		}

		return StakeHistoryEntry{
			Deactivating: current_effective_stake,
		}
	} else {
		return StakeHistoryEntry{}
	}
}

type StakeHistory struct {
	Epoch uint64
	Entry StakeHistoryEntry
}

type StakeHistoryEntry struct {
	Effective    uint64
	Activating   uint64
	Deactivating uint64
}

func CalStakeActivation(ctx context.Context, client *rpc.Client, address solana.PublicKey) (*GetStakeActivationResponse, error) {
	stakeAccountInfo, err := client.GetAccountInfo(ctx, address)
	if err != nil {
		return nil, err
	}
	stakeAccount := StakeAccount{}
	err = bin.NewBinDecoder(stakeAccountInfo.Value.Data.GetBinary()).Decode(&stakeAccount)
	if err != nil {
		return nil, err
	}

	if stakeAccount.Type == 0 {
		return nil, fmt.Errorf("stake account not init")
	}

	rentExemptReserve := stakeAccount.Info.Meta.RentExemptReserve
	delegation := stakeAccount.Info.Stake.Delegation
	if delegation == (Delegation{}) {
		return &GetStakeActivationResponse{
			State:    StakeActivationStateInactive,
			Active:   0,
			Inactive: stakeAccountInfo.Value.Lamports - uint64(rentExemptReserve),
		}, nil
	}

	stakeHistoriesAccountInfo, err := client.GetAccountInfo(ctx, solana.SysVarStakeHistoryPubkey)
	if err != nil {
		return nil, err
	}
	stakeHistories := []StakeHistory{}
	err = bin.NewBinDecoder(stakeHistoriesAccountInfo.Value.Data.GetBinary()).Decode(&stakeHistories)
	if err != nil {
		return nil, err
	}

	epochInfo, err := client.GetEpochInfo(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}
	stakeActivationStatus := delegation.StakeActivatingAndDeactivating(uint64(epochInfo.Epoch), stakeHistories)

	stake_activation_state := StakeActivationStateInactive
	if stakeActivationStatus.Deactivating > 0 {
		stake_activation_state = StakeActivationStateDeactivating
	} else if stakeActivationStatus.Activating > 0 {
		stake_activation_state = StakeActivationStateActivating
	} else if stakeActivationStatus.Effective > 0 {
		stake_activation_state = StakeActivationStateActive
	} else {
		stake_activation_state = StakeActivationStateInactive
	}

	inactive_stake := uint64(0)
	if stakeAccountInfo.Value.Lamports > stakeActivationStatus.Effective+uint64(rentExemptReserve) {
		inactive_stake = stakeAccountInfo.Value.Lamports - (stakeActivationStatus.Effective + uint64(rentExemptReserve))
	}

	return &GetStakeActivationResponse{
		State:    stake_activation_state,
		Active:   stakeActivationStatus.Effective,
		Inactive: inactive_stake,
	}, nil
}
