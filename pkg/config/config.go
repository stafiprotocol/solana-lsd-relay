// Copyright 2021 stafiprotocol
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ConfigInitStakeManager struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string

	LsdProgramID        string
	StackAddress        string
	LsdTokenMintAddress string
	ValidatorAddress    string
	StakeManagerAddress string

	FeePayerAccount string
	AdminAccount    string
}

func LoadInitStakeManagerConfig(configFilePath string) (*ConfigInitStakeManager, error) {
	var cfg = ConfigInitStakeManager{}
	if err := loadConfigFromFile(configFilePath, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type ConfigInitStack struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string

	LsdProgramID string

	FeePayerAccount string
	AdminAccount    string

	// setting
	StackAddress                    string
	AddEntrustedStakeManagerAddress string

	StakeManagerAddress string
	StackFeeCommission  uint64
}

func LoadInitStackConfig(configFilePath string) (*ConfigInitStack, error) {
	var cfg = ConfigInitStack{}
	if err := loadConfigFromFile(configFilePath, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

type ConfigStart struct {
	Endpoint     string // url for  rpc endpoint
	LogFilePath  string
	KeystorePath string

	LsdProgramID string

	StackAddress        string
	StakeManagerAddress string

	FeePayerAccount string
}

func LoadStartConfig(configFilePath string) (*ConfigStart, error) {
	var cfg = ConfigStart{}
	if err := loadConfigFromFile(configFilePath, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.LogFilePath) == 0 {
		cfg.LogFilePath = "./log_data"
	}

	return &cfg, nil
}

type ConfigSetStakeManager struct {
	Endpoint     string // url for  rpc endpoint
	KeystorePath string
	ExportTx     bool

	LsdProgramID        string
	StakeManagerAddress string
	FeePayerAccount     string
	AdminAccount        string

	// setting
	AddValidatorAddress    string
	RemoveValidatorAddress string
	RateChangeLimit        uint64
	UnbondingDuration      uint64
	MinStakeAmount         uint64
	PlatformFeeCommission  uint64
	NewBalancerAddress     string
	NewAdminAddress        string
}

func LoadSetStakeManagerConfig(configFilePath string) (*ConfigSetStakeManager, error) {
	var cfg = ConfigSetStakeManager{}
	if err := loadConfigFromFile(configFilePath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

type ConfigCreateMetadata struct {
	Endpoint     string // rpc endpoint
	KeystorePath string
	ExportTx     bool

	LsdProgramID        string
	StakeManagerAddress string
	StakePoolAddress    string
	LsdTokenMintAddress string

	FeePayerAccount string
	AdminAccount    string

	// Metadata parameters
	TokenName   string
	TokenSymbol string
	TokenUri    string
}

func LoadCreateMetadataConfig(configFilePath string) (*ConfigCreateMetadata, error) {
	var cfg = ConfigCreateMetadata{}
	if err := loadConfigFromFile(configFilePath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadConfigFromFile(path string, config any) error {
	_, err := os.Open(path)
	if err != nil {
		return err
	}
	if _, err := toml.DecodeFile(path, config); err != nil {
		return err
	}
	fmt.Println("load config success")
	return nil
}
