package task

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/sirupsen/logrus"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/config"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/lsd_program"
	"github.com/stafiprotocol/solana-lsd-relay/pkg/utils"
	"golang.org/x/time/rate"
)

var stakePoolSeed = []byte("pool_seed")

type Task struct {
	stop chan struct{}
	cfg  config.ConfigStart

	lsdProgramID       solana.PublicKey
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

func NewTask(cfg config.ConfigStart, feePayer solana.PrivateKey) *Task {
	s := &Task{
		stop:            make(chan struct{}),
		cfg:             cfg,
		feePayerAccount: feePayer,
		entrustedMode:   true,
	}
	return s
}

func (t *Task) Start() error {
	t.client = rpc.NewWithCustomRPCClient(rpc.NewWithLimiter(
		t.cfg.Endpoint,
		rate.Every(time.Second), // time frame
		5,                       // limit of requests per time frame
	))

	lsdProgramID := solana.MustPublicKeyFromBase58(t.cfg.LsdProgramID)
	stackAccountPubkey := solana.MustPublicKeyFromBase58(t.cfg.StackAddress)

	t.lsdProgramID = lsdProgramID
	t.stackAccountPubkey = stackAccountPubkey
	if len(t.cfg.StakeManagerAddress) > 0 {
		t.stakeManagerPubkey = solana.MustPublicKeyFromBase58(t.cfg.StakeManagerAddress)
		t.entrustedMode = false
	}

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
		stackAccount := lsd_program.Stack{}
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

func (t *Task) getStakeManagerAndPool(stakeManagerPubkey solana.PublicKey) (*lsd_program.StakeManager, solana.PublicKey, error) {
	stakeManager := lsd_program.StakeManager{}
	err := utils.GetAndDecodeAccountInfo(t.client, stakeManagerPubkey, &stakeManager)
	if err != nil {
		return nil, solana.PublicKey{}, err
	}

	stakePool, _, err := solana.FindProgramAddress([][]byte{stakeManagerPubkey.Bytes(), stakePoolSeed}, t.lsdProgramID)
	if err != nil {
		return nil, solana.PublicKey{}, err
	}

	return &stakeManager, stakePool, nil
}
