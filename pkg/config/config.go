// Copyright 2021 stafiprotocol
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"

	"github.com/BurntSushi/toml"
)

type ConfigInitStakeManager struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string

	StakeManagerProgramID string
	StackAddress          string
	ValidatorAddress      string

	FeePayerAccount string
	AdminAccount    string
}

type ConfigInitStack struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string

	StackProgramID string

	FeePayerAccount string
	AdminAccount    string
}

type ConfigStartStakeManager struct {
	Endpoint     string // url for  rpc endpoint
	LogFilePath  string
	KeystorePath string

	StakeManagerAddress string

	FeePayerAccount string
}

type ConfigStartStack struct {
	Endpoint     string // url for  rpc endpoint
	LogFilePath  string
	KeystorePath string

	StackAddress string

	FeePayerAccount string
}

type ConfigSetStakeManager struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string
	ExportTx     bool

	StakeManagerAddress string
	FeePayerAccount     string
	AdminAccount        string

	// setting
	AddValidatorAddress    string
	RemoveValidatorAddress string
	RateChangeLimit        int64
	UnbondingDuration      int64
	MinStakeAmount         int64
	PlatformFeeCommission  int64
	NewBalancerAddress     string

	NewAdminAddress string
}

type ConfigSetStack struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string
	ExportTx     bool

	FeePayerAccount string
	AdminAccount    string
	StackAddress    string

	AddEntrustedStakeManagerAddress    string
	RemoveEntrustedStakeManagerAddress string
	StakeManagerAddress                string
	StackFeeCommission                 uint64
	NewAdminAddress                    string
}

func LoadConfig[conf any](path string) (*conf, error) {
	ret := new(conf)
	if _, err := toml.DecodeFile(path, ret); err != nil {
		return nil, err
	}
	fmt.Println("load config success")

	return ret, nil
}
