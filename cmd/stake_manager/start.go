package stake_manager

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/stafiprotocol/solana-lsd-relay/cmd/common"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/log"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"github.com/stafiprotocol/solana-lsd-relay/task"
)

func StartCmd() *cobra.Command {

	var cmd = &cobra.Command{
		Use:   "start",
		Short: "Start solana lsd relay",

		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := cmd.Flags().GetString(common.FlagConfigPath)
			if err != nil {
				return err
			}
			fmt.Printf("Config path: %s\n", configPath)

			cfg, err := config.LoadConfig[config.ConfigStartStakeManager](configPath)
			if err != nil {
				return err
			}

			bts, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Printf("Config: \n%s\n", string(bts))
		Out:
			for {
				fmt.Println("\nCheck config info, then press (y/n) to continue:")
				var input string
				fmt.Scanln(&input)
				switch input {
				case "y":
					break Out
				case "n":
					return nil
				default:
					fmt.Println("press `y` or `n`")
					continue
				}
			}

			logLevelStr, err := cmd.Flags().GetString(common.FlagLogLevel)
			if err != nil {
				return err
			}
			logLevel, err := logrus.ParseLevel(logLevelStr)
			if err != nil {
				return err
			}
			logrus.SetLevel(logLevel)
			if len(cfg.LogFilePath) == 0 {
				cfg.LogFilePath = common.DefaultLogDir
			}
			err = log.InitLogFile(cfg.LogFilePath + "/relay")
			if err != nil {
				return fmt.Errorf("InitLogFile failed: %w", err)
			}

			ctx := utils.ShutdownListener()

			privateKeyMap, err := utils.LoadPrivateKeysFromKeystore(cfg.KeystorePath)
			if err != nil {
				return err
			}
			feePayerAccount, exist := privateKeyMap[cfg.FeePayerAccount]
			if !exist {
				return fmt.Errorf("fee payer not exit in vault")
			}

			t := task.NewTask(cfg.Endpoint, cfg.StakeManagerAddress, "", feePayerAccount)
			err = t.Start()
			if err != nil {
				return err
			}
			defer func() {
				logrus.Infof("shutting down task ...")
				t.Stop()
			}()

			<-ctx.Done()

			return nil
		},
	}
	cmd.Flags().String(common.FlagConfigPath, common.DefaultConfigPath, "Config file path")
	cmd.Flags().String(common.FlagLogLevel, logrus.InfoLevel.String(), "The logging level (trace|debug|info|warn|error|fatal|panic)")
	return cmd
}
