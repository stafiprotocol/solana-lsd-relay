package task

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-go-sdk/client"
	"github.com/stafiprotocol/solana-go-sdk/common"
	"github.com/stafiprotocol/solana-go-sdk/lsdprog"
	"github.com/stafiprotocol/solana-go-sdk/types"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
)

var stakePoolSeed = []byte("pool_seed")

type Task struct {
	stop        chan struct{}
	cfg         config.ConfigStart
	accountsMap map[string]types.Account

	lsdProgramID       common.PublicKey
	stackAccountPubkey common.PublicKey
	stakeManagerPubkey common.PublicKey

	feePayerAccount types.Account
	entrustedMode   bool

	client   *client.Client
	handlers []Handler
}

type Handler struct {
	method func(common.PublicKey) error
	name   string
}

func NewTask(cfg config.ConfigStart, accouts map[string]types.Account) *Task {
	s := &Task{
		stop:          make(chan struct{}),
		cfg:           cfg,
		accountsMap:   accouts,
		entrustedMode: true,
	}
	return s
}

func (task *Task) Start() error {
	task.client = client.NewClient(task.cfg.EndpointList)

	lsdProgramID := common.PublicKeyFromString(task.cfg.LsdProgramID)
	stackAccountPubkey := common.PublicKeyFromString(task.cfg.StackAddress)

	feePayerAccount, exist := task.accountsMap[task.cfg.FeePayerAccount]
	if !exist {
		return fmt.Errorf("fee payer not exit in vault")
	}

	task.lsdProgramID = lsdProgramID
	task.stackAccountPubkey = stackAccountPubkey
	task.feePayerAccount = feePayerAccount
	if len(task.cfg.StakeManagerAddress) > 0 {
		task.stakeManagerPubkey = common.PublicKeyFromString(task.cfg.StakeManagerAddress)
		task.entrustedMode = false
	}

	task.appendHandlers(task.EraNew, task.EraBond, task.EraUnbond, task.EraUpdateActive, task.EraUpdateRate, task.EraMerge, task.EraWithdraw)
	SafeGoWithRestart(task.handler)
	return nil
}

func (task *Task) Stop() {
	close(task.stop)
}

func (s *Task) appendHandlers(handlers ...func(common.PublicKey) error) {
	for _, handler := range handlers {

		funcNameRaw := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

		splits := strings.Split(funcNameRaw, "/")
		funcName := splits[len(splits)-1]
		funcName = strings.Split(funcName, ".")[2]
		funcName = strings.Split(funcName, "-")[0]

		s.handlers = append(s.handlers, Handler{
			method: handler,
			name:   funcName,
		})
	}
}

func (s *Task) handler() {
	logrus.Info("start handlers")
	retry := 0

	for {
		if retry > 200 {
			utils.ShutdownRequestChannel <- struct{}{}
			return
		}
		select {
		case <-s.stop:
			logrus.Info("task has stopped")
			return
		default:
			err := s.handleEra()
			if err != nil {
				logrus.Warnf("era handle failed: %s, will retry.", err)
				time.Sleep(time.Second * 6)
				retry++
				continue
			}

			retry = 0
		}

		time.Sleep(30 * time.Second)
	}
}

func (t *Task) handleEra() error {
	if t.entrustedMode {
		stackAccount, err := t.client.GetLsdStack(context.Background(), t.stackAccountPubkey.ToBase58())
		if err != nil {
			return err
		}

		for _, stakeManager := range stackAccount.EntrustedStakeManagers {
			for _, handler := range t.handlers {
				funcName := handler.name
				logrus.Debugf("stakeManager: %s, handler %s start...", stakeManager.ToBase58(), funcName)
				err := handler.method(stakeManager)
				if err != nil {
					return fmt.Errorf("handler %s failed: %s, will retry", funcName, err)
				}
				logrus.Debugf("stakeManager: %s, handler %s end", stakeManager.ToBase58(), funcName)
			}
		}
	} else {
		for _, handler := range t.handlers {
			funcName := handler.name
			logrus.Debugf("handler %s start...", funcName)
			err := handler.method(t.stakeManagerPubkey)
			if err != nil {
				return fmt.Errorf("handler %s failed: %s, will retry", funcName, err)
			}
			logrus.Debugf("handler %s end", funcName)
		}
	}
	return nil
}

func isEmpty(data *lsdprog.EraProcessData) bool {
	return data.NeedBond == 0 && data.NeedUnbond == 0 && data.NewActive == 0 && data.OldActive == 0 && len(data.PendingStakeAccounts) == 0
}
func needBond(data *lsdprog.EraProcessData) bool {
	return data.NeedBond > 0
}

func needUnbond(data *lsdprog.EraProcessData) bool {
	return data.NeedUnbond > 0
}

func needUpdateActive(data *lsdprog.EraProcessData) bool {
	return data.NeedUnbond == 0 && data.NeedBond == 0 && len(data.PendingStakeAccounts) > 0
}

func needUpdateRate(data *lsdprog.EraProcessData) bool {
	return data.NeedUnbond == 0 && data.NeedBond == 0 && len(data.PendingStakeAccounts) == 0 && data.NewActive != 0 && data.OldActive != 0
}

func (t *Task) waitTx(txHash string) error {
	retry := 0
	for {
		if retry > 50 {
			return fmt.Errorf("waitTx %s reach retry limit", txHash)
		}

		tx, err := t.client.GetTransactionV2(context.Background(), txHash)
		if err != nil {
			logrus.Debugf("query tx %s failed: %s", txHash, err.Error())
			time.Sleep(time.Second * 6)
			retry++
			continue
		}

		if tx.Meta.Err != nil {
			return fmt.Errorf("%v", tx.Meta.Err)
		}
		return nil
	}
}
