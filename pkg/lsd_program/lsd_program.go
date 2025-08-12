package lsd_program

import "github.com/gagliardetto/solana-go"

type StakeAccount struct {
	Type uint32 // 0 uninitialized 1 initialized 2 delegated 3 rewardspool
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

// era process data helper functions

func (data EraProcessData) IsEmpty() bool {
	return data.NeedBond == 0 && data.NeedUnbond == 0 && data.NewActive == 0 && data.OldActive == 0 && len(data.PendingStakeAccounts) == 0
}

func (data EraProcessData) IsNeedSkipBond(minDelegationAmount uint64) bool {
	return data.NeedBond > 0 && data.NeedBond < minDelegationAmount
}

func (data EraProcessData) IsNeedBond(minDelegationAmount uint64) bool {
	return data.NeedBond >= minDelegationAmount
}

func (data EraProcessData) IsNeedUnbond() bool {
	return data.NeedUnbond > 0
}

func (data EraProcessData) IsNeedUpdateActive() bool {
	return data.NeedUnbond == 0 && data.NeedBond == 0 && len(data.PendingStakeAccounts) > 0
}

func (data EraProcessData) IsNeedUpdateRate() bool {
	return data.NeedUnbond == 0 && data.NeedBond == 0 && len(data.PendingStakeAccounts) == 0 && data.NewActive != 0 && data.OldActive != 0
}
