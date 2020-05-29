package check

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/chislab/go-fiscobcos/accounts/abi/bind"
	"github.com/chislab/go-fiscobcos/common"
	"github.com/chislab/go-fiscobcos/core/types"
	"github.com/chislab/go-fiscobcos/crypto"
	"github.com/chislab/go-fiscobcos/ethclient"
	"github.com/chislab/go-fiscobcos/rpc"
)

var (
	callOpts = &bind.CallOpts{GroupId: 1, From: common.HexToAddress("0x115EFA2481ce10F0CEB043A6f150E7E728EC4a06")}
	GethCli  *ethclient.Client
	chanCli   *rpc.Client
	tx       *types.Transaction
	err 	error
	receipt  *types.Receipt
)

func init() {
	var err error
	GethCli, err = ethclient.Dial(&rpc.ClientConfig{
		Endpoint: "chan://127.0.0.1:20200",
		//Endpoint: "http://127.0.0.1:8545",
		CAFile: "./nodes/127.0.0.1/sdk/ca.crt",
		CertFile: "./nodes/127.0.0.1/sdk/node.crt",
		KeyFile: "./nodes/127.0.0.1/sdk/node.key",
	})
	GethCli.GroupId = 1
	if err != nil {
		println("error:", err.Error())
		os.Exit(1)
	}
	height, err := GethCli.BlockNumber(context.Background())
	if err != nil {
		println("error:", err.Error())
		os.Exit(1)
	}
	fmt.Println("Current block height is", height.String())
	// watch contract events.
	accessAdmin = NewAuthFromPriKey(height, AdminPrikey)
}

func str2Big(str string) *big.Int {
	return new(big.Int).SetBytes([]byte(str))
}

func WaitMinedByHash(txHash common.Hash) *types.Receipt {
	ctx := context.Background()
	queryTicker := time.NewTicker(time.Millisecond * 200)
	defer queryTicker.Stop()
	for {
		receipt, _ := GethCli.TransactionReceipt(ctx, txHash)
		if receipt != nil {
			return receipt
		}
		// Wait for the next round.
		select {
		case <-ctx.Done():
			return nil
		case <-queryTicker.C:
		}
	}
}

func NewAuthFromPriKey(height *big.Int, priKey... string) *bind.TransactOpts {
	var priv *ecdsa.PrivateKey
	if len(priKey) == 0 {
		priv, _ = crypto.GenerateKey()
	} else {
		priv, _ = crypto.HexToECDSA(priKey[0])
	}
	auth := bind.NewKeyedTransactor(priv, 1, 1)
	auth.BlockLimit = height.Uint64() + 100
	auth.Context = context.Background()
	return auth
}

func ilog(contract string, format string, v ...interface{}) {
	log.Printf("[%s | INFO]: %s", strings.ToUpper(contract), fmt.Sprintf(format, v...))
}

func getReceiptOutput(output string) string {
	if strings.HasPrefix(output, "0x") {
		output = output[2:]
	}
	b, err := hex.DecodeString(output)
	if err != nil || len(b) < 36 {
		return output
	}
	b = b[36:]
	tail := len(b) - 1
	for ; tail >= 0; tail-- {
		if b[tail] != 0 {
			break
		}
	}
	return string(b[:tail+1])
}

func checkTx(tx *types.Transaction, err error) {
	if err != nil {
		panic(err)
	}
	receipt, err = func(tx *types.Transaction, err error) (*types.Receipt, error) {
		receipt := WaitMinedByHash(tx.Hash())
		if receipt.Status != "0x0" {
			return receipt, fmt.Errorf("receipt.Status = %s\nTxHash = %s\nOutput = %s", receipt.Status, receipt.TxHash.String(), getReceiptOutput(receipt.Output))
		}
		return receipt, nil
	}(tx, err)
	if err != nil {
		panic(err)
	}
	return
}

//func WatchEvent() {
//	ch, err := chanCli.SubEventLogs(chanArg)
//	if err != nil {
//		panic(err)
//	}
//	for {
//		msg := <-ch
//		switch msg.Address {
//		case produceAddr:
//			parseProductEvt(msg)
//		case materialAddr:
//			parseMaterialEvt(msg)
//		case paymentAddr:
//			parsePaymentEvt(msg)
//		}
//	}
//}

