// Copyright 2021 stafiprotocol
// SPDX-License-Identifier: LGPL-3.0-only

package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type ConfigInitStakeManager struct {
	EndpointList []string // url for  rpc endpoint
	KeystorePath string

	LsdProgramID        string
	StackAddress        string
	LsdTokenMintAddress string
	ValidatorAddress    string
	StakeManagerAddress string

	FeePayerAccount string
	AdminAccount    string

	// setting
	AddValidatorAddress    string
	RemoveValidatorAddress string
	RateChangeLimit        uint64
	UnbondingDuration      uint64
}

func LoadInitStakeManagerConfig(configFilePath string) (*ConfigInitStakeManager, error) {
	var cfg = ConfigInitStakeManager{}
	if err := loadSysConfigInitStakeManager(configFilePath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadSysConfigInitStakeManager(path string, config *ConfigInitStakeManager) error {
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

type ConfigInitStack struct {
	EndpointList []string // url for  rpc endpoint
	KeystorePath string

	LsdProgramID string

	FeePayerAccount string
	AdminAccount    string

	// setting
	StackAddress                    string
	AddEntrustedStakeManagerAddress string
}

func LoadInitStackConfig(configFilePath string) (*ConfigInitStack, error) {
	var cfg = ConfigInitStack{}
	if err := loadSysConfigInitStack(configFilePath, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func loadSysConfigInitStack(path string, config *ConfigInitStack) error {
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

type ConfigStart struct {
	EndpointList []string // url for  rpc endpoint
	LogFilePath  string
	KeystorePath string

	LsdProgramID string

	StackAddress        string
	StakeManagerAddress string

	FeePayerAccount string
}

func LoadStartConfig(configFilePath string) (*ConfigStart, error) {
	var cfg = ConfigStart{}
	if err := loadSysConfigStart(configFilePath, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.LogFilePath) == 0 {
		cfg.LogFilePath = "./log_data"
	}

	return &cfg, nil
}

func loadSysConfigStart(path string, config *ConfigStart) error {
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
