package task

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/stack"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

type Task struct {
	stop                chan struct{}
	endpoint            string
	stakeManagerAddress string
	stackAddress        string

	stackProgramID     solana.PublicKey
	stackAccountPubkey solana.PublicKey

	stakeManagerPubkey solana.PublicKey

	feePayerAccount solana.PrivateKey
	entrustedMode   bool

	client   *rpc.Client
	handlers []Handler
}

type Handler struct {
	method func(solana.PublicKey) error
	name   string
}

func NewTask(endpoint, stakeManagerAddress, stackAddress string, feePayer solana.PrivateKey) *Task {
	s := &Task{
		stop:                make(chan struct{}),
		endpoint:            endpoint,
		stakeManagerAddress: stakeManagerAddress,
		stackAddress:        stackAddress,
		feePayerAccount:     feePayer,
		entrustedMode:       false,
	}
	return s
}

func (t *Task) Start() error {
	t.client = rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		t.endpoint,
		rate.Every(time.Second), // time frame
		5,                       // limit of requests per time frame
	))

	if len(t.stakeManagerAddress) > 0 {
		t.entrustedMode = false
		t.stakeManagerPubkey = solana.MustPublicKeyFromBase58(t.stakeManagerAddress)

		stakeManagerAccount, _, _, err := utils.GetStakeManagerInfo(t.client, t.stakeManagerPubkey)
		if err != nil {
			return err
		}

		t.stackAccountPubkey = stakeManagerAccount.Stack
	} else {
		t.entrustedMode = true
		t.stackAccountPubkey = solana.MustPublicKeyFromBase58(t.stackAddress)
	}
	stackAccount, err := t.client.GetAccountInfo(context.Background(), t.stackAccountPubkey)
	if err != nil {
		return err
	}
	t.stackProgramID = stackAccount.Value.Owner
	stack.SetProgramID(t.stackProgramID)

	t.appendHandlers(t.EraNew, t.EraSkipBond, t.EraBond, t.EraUnbond, t.EraUpdateActive, t.EraUpdateRate, t.EraMerge, t.EraWithdraw)
	SafeGoWithRestart(t.handler)
	return nil
}

func (t *Task) Stop() {
	close(t.stop)
}

func (t *Task) appendHandlers(handlers ...func(solana.PublicKey) error) {
	for _, handler := range handlers {

		funcNameRaw := runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()

		splits := strings.Split(funcNameRaw, "/")
		funcName := splits[len(splits)-1]
		funcName = strings.Split(funcName, ".")[2]
		funcName = strings.Split(funcName, "-")[0]

		t.handlers = append(t.handlers, Handler{
			method: handler,
			name:   funcName,
		})
	}
}

func (t *Task) handler() {
	logrus.Info("start handlers")
	retry := 0

	for {
		if retry > 200 {
			utils.ShutdownRequestChannel <- struct{}{}
			return
		}
		select {
		case <-t.stop:
			logrus.Info("task has stopped")
			return
		default:
			err := t.handleEra()
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
		stackAccount := stack.Stack{}
		err := utils.GetAndDecodeAccountInfo(t.client, t.stackAccountPubkey, &stackAccount)
		if err != nil {
			return err
		}

		for _, stakeManager := range stackAccount.EntrustedStakeManagers {
			for _, handler := range t.handlers {
				funcName := handler.name
				logrus.Debugf("stakeManager: %s, handler %s start...", stakeManager, funcName)
				err := handler.method(stakeManager)
				if err != nil {
					return fmt.Errorf("handler %s failed: %s, will retry", funcName, err)
				}
				logrus.Debugf("stakeManager: %s, handler %s end", stakeManager, funcName)
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
