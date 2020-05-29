package check

import (
	"context"
	"fisco/build/access"
	"fisco/build/material"
	"fisco/build/payment"
	"fisco/build/produce"
	"fmt"
	"github.com/chislab/go-fiscobcos/accounts/abi/bind"
	"github.com/chislab/go-fiscobcos/common"
	"github.com/urfave/cli/v2"
	"math/big"
)

var (
	accessAddr       common.Address
	accessContract   *access.Access
	produceAddr      common.Address
	produceContract  *produce.Produce
	materialAddr     common.Address
	materialContract *material.Material
	paymentAddr      common.Address
	paymentContract  *payment.Payment
)

var (
	AdminPrikey = "526ccb243b5e279a3ce30c08e4d091a0eb2c3bb5a700946d4da47b28df8fe6d5"
	ProducePrikey = "7499b53affec1e5d31bf736bc30b2e39bcae6838047c662aa4df359cc6672416"
	MaterialPrikey = "3f4157a56042544439d7c3baa94594eec26bf3ee680f7351897fc017a68f80cc"
	PaymentPrikey = "8b25edc8b392f134ee29c5442f34b039dbc7efb93bfbf8979438bd06ce60d5f1"
	MProducer1Prikey = "ea5917d4c7f289ce0fa590edd0e5bff3bd931649fcfc918e8a83b9ad879a31dc"
	MProducer2Prikey = "c29372135a5583e79f6c12893a96ce24060b6ecb3fbb7487dbea61d1df8b83a1"
	MProducer3Prikey = "3034b6f9ccdd446db372c9df92f082aca15ade1cbb5101eac0194dc7f5d306d3"
	PProducerPrikey = "7a8b52e315a5fd5c42e4f3ee0df141bee1b7298a939ffb78e40050ae833d8b84"
)

var (
	accessAdmin *bind.TransactOpts
	produceAdmin *bind.TransactOpts
	materialAdmin *bind.TransactOpts
	paymentAdmin *bind.TransactOpts

	mProducerAdmins []*bind.TransactOpts
	pProducerAdmins []*bind.TransactOpts

	customAuth *bind.TransactOpts
)

var (
	productType = []string{"TV", "PC"}
	materialType = []string{"LCD", "Audio", "CPU"}
)

func deloyContracts() {
	fmt.Println("deploay contracts...")
	// 方便bsn_backend对该模块进行复用
	accessAddr, tx, accessContract, err = access.DeployAccess(accessAdmin, GethCli)
	checkTx(tx, err)
	fmt.Println("accessAddr", accessAddr.String())

	produceAddr, tx, produceContract, err = produce.DeployProduce(produceAdmin, GethCli, accessAddr, big.NewInt(int64(len(materialType))))
	checkTx(tx, err)
	fmt.Println("produceAddr", produceAddr.String())

	materialAddr, tx, materialContract, err = material.DeployMaterial(materialAdmin, GethCli, accessAddr)
	checkTx(tx, err)
	fmt.Println("materialAddr", materialAddr.String())

	paymentAddr, tx, paymentContract, err = payment.DeployPayment(paymentAdmin, GethCli, big.NewInt(50))
	checkTx(tx, err)
	fmt.Println("paymentAddr", paymentAddr.String())

	// 设置权限
	tx, err = accessContract.GrantPayment(accessAdmin, paymentAddr)
	checkTx(tx, err)
	for _, auth := range pProducerAdmins {
		tx, err = accessContract.GrantProductProducer(accessAdmin, auth.From)
		checkTx(tx, err)
	}
	for _, auth := range mProducerAdmins {
		tx, err = accessContract.GrantMaterialProducer(accessAdmin, auth.From)
		checkTx(tx, err)
	}

	//init 合约
	tx, err = produceContract.SetMaterialContract(produceAdmin, materialAddr)
	checkTx(tx, err)
	tx, err = produceContract.SetPaymentContract(produceAdmin, paymentAddr)
	checkTx(tx, err)
	tx, err = paymentContract.SetProductProducer(paymentAdmin, produceAddr)
	checkTx(tx, err)
	tx, err = paymentContract.SetMaterialProducer(paymentAdmin, materialAddr)
	checkTx(tx, err)
}

func initKeys() {
	fmt.Println("Init keys...")
	height, _ := GethCli.BlockNumber(context.Background())

	accessAdmin = NewAuthFromPriKey(height, AdminPrikey)
	produceAdmin = NewAuthFromPriKey(height, ProducePrikey)
	materialAdmin = NewAuthFromPriKey(height, MaterialPrikey)
	paymentAdmin = NewAuthFromPriKey(height, PaymentPrikey)
	mProducerAdmins = make([]*bind.TransactOpts, len(materialType))
	mProducerAdmins[0] = NewAuthFromPriKey(height, MProducer1Prikey)
	mProducerAdmins[1] = NewAuthFromPriKey(height, MProducer2Prikey)
	mProducerAdmins[2] = NewAuthFromPriKey(height, MProducer3Prikey)

	pProducerAdmins = make([]*bind.TransactOpts, len(productType))
	pProducerAdmins[0] = NewAuthFromPriKey(height, PProducerPrikey)
	pProducerAdmins[1] = NewAuthFromPriKey(height)

	customAuth = NewAuthFromPriKey(height)
}

func TestFull(ctx *cli.Context) error {
	fmt.Println("Init contracts, please be patient...")
	initKeys()
	deloyContracts()
	fmt.Println("================== check full procedures ==================")

	tx, err = materialContract.SetPrice(mProducerAdmins[0], str2Big("LCD"), big.NewInt(int64(100)))
	checkTx(tx, err)
	tx, err = materialContract.SetPrice(mProducerAdmins[1], str2Big("Audio"), big.NewInt(int64(50)))
	checkTx(tx, err)
	tx, err = materialContract.SetPrice(mProducerAdmins[2], str2Big("CPU"), big.NewInt(int64(200)))
	checkTx(tx, err)
	tx, err = produceContract.UpdateProductPrice(pProducerAdmins[0], str2Big("TV"), big.NewInt(3000))
	checkTx(tx, err)
	tx, err = produceContract.UpdateProductPrice(pProducerAdmins[1], str2Big("PC"), big.NewInt(5000))

	// 充值
	mintAmount := big.NewInt(100000)
	for _, auth := range mProducerAdmins {
		tx, err = paymentContract.Mint(paymentAdmin, auth.From, mintAmount)
		checkTx(tx, err)
	}
	for _, auth := range pProducerAdmins {
		tx, err = paymentContract.Mint(paymentAdmin, auth.From, mintAmount)
		checkTx(tx, err)
	}
	tx, err = paymentContract.Mint(paymentAdmin, customAuth.From, mintAmount)
	checkTx(tx, err)

	// 普通用户下产品订单
	tx, err = paymentContract.MakeOrder(customAuth, false, pProducerAdmins[0].From, str2Big("TV"), big.NewInt(5), big.NewInt(3000))
	checkTx(tx, err)
	// 产品厂家下原材料订单
	tx, err = paymentContract.MakeOrder(pProducerAdmins[0], true, mProducerAdmins[0].From, str2Big("LCD"), big.NewInt(100), big.NewInt(100))
	checkTx(tx, err)
	tx, err = paymentContract.MakeOrder(pProducerAdmins[0], true, mProducerAdmins[1].From, str2Big("Audio"), big.NewInt(100), big.NewInt(50))
	checkTx(tx, err)
	tx, err = paymentContract.MakeOrder(pProducerAdmins[0], true, mProducerAdmins[2].From, str2Big("CPU"), big.NewInt(100), big.NewInt(200))
	checkTx(tx, err)

	// 检查原料, 如果数量充足, 就进行订单确认，但是这里假定订单数量都不充足。进行生产后交付。理应检查原件
	// 线下准备原料, 在这里进行注册
	tx, err = materialContract.NewMaterial(mProducerAdmins[0], str2Big("LCD"), big.NewInt(300), str2Big("LCD_1"))
	checkTx(tx, err)
	tx, err = materialContract.NewMaterial(mProducerAdmins[1], str2Big("Audio"), big.NewInt(300), str2Big("Audio_1"))
	checkTx(tx, err)
	tx, err = materialContract.NewMaterial(mProducerAdmins[2], str2Big("CPU"), big.NewInt(300), str2Big("CPU_1"))
	checkTx(tx, err)

	// 生产厂家确认原料订单
	tx, err = paymentContract.ConfirmOrder(pProducerAdmins[0], big.NewInt(0)) // LCD=
	checkTx(tx, err)
	tx, err = paymentContract.ConfirmOrder(pProducerAdmins[0], big.NewInt(2)) // Audio
	checkTx(tx, err)
	tx, err = paymentContract.ConfirmOrder(pProducerAdmins[0], big.NewInt(4)) // CPU
	checkTx(tx, err)

	// 进行产品生产
	// 1. 首先消耗零件
	tx, err = materialContract.ConsumeMaterial(pProducerAdmins[0], str2Big("LCD"), big.NewInt(20))
	checkTx(tx, err)
	tx, err = materialContract.ConsumeMaterial(pProducerAdmins[0], str2Big("Audio"), big.NewInt(20))
	checkTx(tx, err)
	tx, err = materialContract.ConsumeMaterial(pProducerAdmins[0], str2Big("CPU"), big.NewInt(20))
	checkTx(tx, err)

	// 2. 产品进行注册, 注册不需要具体知道零件消耗了多少
	for i := 0; i < 10; i++ {
		productID := fmt.Sprintf("ProductID_%d", i)
		tx, err = produceContract.RegisterProduct(pProducerAdmins[0], str2Big("TV"), str2Big(productID), str2Big("2020-05-20"),
			[]*big.Int{str2Big("LCD_1"), str2Big("Audio_1"), str2Big("CPU_1")})
		checkTx(tx, err)
	}

	tx, err = paymentContract.ConfirmOrder(customAuth, big.NewInt(1))
	checkTx(tx, err)

	return nil
}
