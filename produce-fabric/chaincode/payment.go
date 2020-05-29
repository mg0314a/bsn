package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/cid"
	"github.com/hyperledger/fabric/protos/peer"
)

// Order 订单
type Order struct {
	OrderID   string    `json:"orderID"`   //订单ID
	Payer     string    `json:"payer"`     //下单者
	Producer  string    `json:"producer"`  //供货商
	Amount    uint64    `json:"amount"`    //订单金额
	Count     uint64    `json:"count"`     //下单数量
	Type      string    `json:"type"`      //产品类型
	OrderType int       `json:"orderType"` //订单类型(物料订单0，产品订单1)
	CreatedAt time.Time `json:"createdAt"` //下单时间
	Status    byte      `json:"status"`    //订单状态 0处理中，1已完成，2已取消
}

func (c *Contract) makeMaterialOrder(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 4 {
		return shim.Error("invalid arguments")
	}
	producer := args[0]
	materialType := args[1]
	count, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid count, got %s", args[2]))
	}
	price, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid price, got %s", args[3]))
	}
	return makeOrder(stub, producer, 0, materialType, count, price)
}

func (c *Contract) makeProductOrder(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 4 {
		return shim.Error("invalid arguments")
	}
	producer := args[0]
	productType := args[1]
	count, err := strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid count, got %s", args[2]))
	}
	price, err := strconv.ParseUint(args[3], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid price, got %s", args[3]))
	}
	return makeOrder(stub, producer, 1, productType, count, price)
}

func (c *Contract) confirmOrder(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("invalid arguments")
	}
	orderID := args[0]
	if orderID == "" {
		return shim.Error("orderID is empty")
	}
	okey := fmt.Sprintf("%s-%s", PrefixOrder, orderID)
	val, err := stub.GetState(okey)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get order from orderID %v", err))
	}
	if len(val) == 0 {
		return shim.Error(fmt.Sprintf("order(%s) does not exist", orderID))
	}
	var order Order
	if err := json.Unmarshal(val, &order); err != nil {
		return shim.Error(fmt.Sprintf("failed to unmarshal order %v", err))
	}
	if order.Status != 0 {
		return shim.Error(fmt.Sprintf("order status wrong(%d)", order.Status))
	}
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error("failed to get role")
	}
	if role != order.Payer {
		return shim.Error(fmt.Sprintf("only the payer can confirm order, payer is %s, you are %s", order.Payer, role))
	}
	if order.OrderType == 0 {
		if err := transferMaterial(stub, order.Producer, order.Payer, order.Type, order.Count); err != nil {
			return shim.Error(fmt.Sprintf("failed to transfer material %v", err))
		}
	} else {
		if err := transferProduct(stub, order.Producer, order.Payer, order.Type, order.Count); err != nil {
			return shim.Error(fmt.Sprintf("failed to transfer product %v", err))
		}
	}
	if err := addBalance(stub, order.Producer, order.Amount); err != nil {
		return shim.Error(fmt.Sprintf("failed to pay to %s, %v", order.Producer, err))
	}
	order.Status = 1
	val, err = json.Marshal(order)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal order %v", err))
	}
	if err = stub.PutState(okey, val); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state %v", err))
	}
	if err := stub.SetEvent("EvtConfirmOrder", val); err != nil {
		return shim.Error(fmt.Sprintf("failed to set event %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) cancelOrder(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("invalid arguments")
	}
	orderID := args[0]
	if orderID == "" {
		return shim.Error("orderID is empty")
	}
	key := fmt.Sprintf("%s-%s", PrefixOrder, orderID)
	val, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get state %v", err))
	}
	if len(val) == 0 {
		return shim.Error(fmt.Sprintf("order(%s) does not exist", orderID))
	}
	var order Order
	if err = json.Unmarshal(val, &order); err != nil {
		return shim.Error(fmt.Sprintf("failed to unmarshal order %v", err))
	}
	if order.Status != 0 {
		return shim.Error(fmt.Sprintf("order status wrong(%d)", order.Status))
	}
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error("failed to get role")
	}
	var compensate uint64
	var remain uint64
	if role == order.Payer {
		cancelCompensate, err := getCancelCompensate(stub)
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to get cancel compensate %v", err))
		}
		compensate = (order.Amount * cancelCompensate) / 100
		remain = order.Amount - compensate
		if err = addBalance(stub, order.Payer, remain); err != nil {
			return shim.Error(fmt.Sprintf("failed to add balance to payer %v", err))
		}
		if err = addBalance(stub, order.Producer, compensate); err != nil {
			return shim.Error(fmt.Sprintf("failed to add balance to producer %v", err))
		}
	} else if role == order.Producer {
		compensate = 0
		remain = order.Amount
		if err = addBalance(stub, order.Payer, remain); err != nil {
			return shim.Error(fmt.Sprintf("failed to add balance to payer %v", err))
		}
	} else {
		return shim.Error(fmt.Sprintf("only the payer(%s) and producer(%s) can cancel order, you are %s", order.Payer, order.Producer, role))
	}
	order.Status = 1
	val, err = json.Marshal(order)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal order %v", err))
	}
	if err = stub.PutState(key, val); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state %v", err))
	}
	evtData, err := json.Marshal(map[string]interface{}{
		"orderID":       orderID,
		"returnToPayer": remain,
		"payToProducer": compensate,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal event data %v", err))
	}
	if err = stub.SetEvent("EvtCancelOrder", evtData); err != nil {
		return shim.Error(fmt.Sprintf("failed to set event %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) getOrder(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("invalid arguments")
	}
	orderID := args[0]
	if orderID == "" {
		return shim.Error("orderID is empty")
	}
	key := fmt.Sprintf("%s-%s", PrefixOrder, orderID)
	val, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get state %v", err))
	}
	if len(val) == 0 {
		return shim.Error("order does not exist")
	}
	return shim.Success(val)
}

func (c *Contract) setCancelCompensate(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if role != "payment" {
		return shim.Error("only for payment")
	}
	if len(args) != 1 {
		return shim.Error("invalid arguments")
	}
	compensate, err := strconv.Atoi(args[0])
	if err != nil || compensate < 0 || compensate > 100 {
		return shim.Error(fmt.Sprintf("cancel compensate invlaid got %s", args[0]))
	}
	key := PrefixCancelCompensate
	val := []byte{byte(compensate)}
	if err := stub.PutState(key, val); err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(nil)
}

func makeOrder(stub shim.ChaincodeStubInterface, producer string, orderType int, pType string, count, price uint64) peer.Response {
	var remotePrice uint64
	var err error
	if orderType == 0 {
		remotePrice, err = getMaterialPrice(stub, producer, pType)
	} else {
		remotePrice, err = getProductPrice(stub, producer, pType)
	}
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get remote price, %v", err))
	}
	if remotePrice > price {
		return shim.Error(fmt.Sprintf("price missmatch, set %d, remote %d", price, remotePrice))
	}
	amount := remotePrice * count
	payer, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error("failed to get role")
	}
	orderID := stub.GetTxID()
	if err := reduceBalance(stub, payer, amount); err != nil {
		return shim.Error(fmt.Sprintf("failed to freeze balance %v", err))
	}
	t, err := stub.GetTxTimestamp()
	if err != nil {
		return shim.Error(err.Error())
	}
	order := Order{
		OrderID:   orderID,
		Payer:     payer,
		Producer:  producer,
		Amount:    amount,
		Count:     count,
		Type:      pType,
		OrderType: orderType,
		CreatedAt: time.Unix(t.GetSeconds(), 0),
		Status:    0,
	}
	okey := fmt.Sprintf("%s-%s", PrefixOrder, orderID)
	odata, err := json.Marshal(order)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal order %v", err))
	}
	if err := stub.PutState(okey, odata); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state %v", err))
	}
	if err := stub.SetEvent("EvtMakeOrder", odata); err != nil {
		return shim.Error(fmt.Sprintf("failed to set event %v", err))
	}
	return shim.Success(odata)
}

func getCancelCompensate(stub shim.ChaincodeStubInterface) (uint64, error) {
	key := PrefixCancelCompensate
	val, err := stub.GetState(key)
	if err != nil {
		return 0, err
	}
	if len(val) != 1 {
		return 0, fmt.Errorf("data in db with wrong format")
	}
	return uint64(val[0]), nil
}

//资金相关
func (c *Contract) mint(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if role != "payment" {
		return shim.Error("only for payment")
	}
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	amount, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error("invalid amount")
	}
	if err := addBalance(stub, args[0], amount); err != nil {
		return shim.Error(fmt.Sprintf("failed to add balance %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) burn(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if role != "payment" {
		return shim.Error("only for payment")
	}
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	amount, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error("invalid amount")
	}
	key := fmt.Sprintf("%s-%s", PrefixBalance, args[0])
	currentState, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get state, %v", err))
	}
	currentAmount := bytesToUint64(currentState)
	if currentAmount < amount {
		amount = currentAmount

	}
	newAmount := currentAmount - amount
	if newAmount == 0 {
		if err := stub.DelState(key); err != nil {
			return shim.Error(fmt.Sprintf("failed to del state, %v", err))
		}
		return shim.Success(nil)
	}
	val := uint64ToBytes(newAmount)
	if err := stub.PutState(key, val); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state, %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) balanceOf(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("invalid arguments")
	}
	key := fmt.Sprintf("%s-%s", PrefixBalance, args[0])
	val, err := stub.GetState(key)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get state, %v", err))
	}
	amount := bytesToUint64(val)
	m := map[string]interface{}{
		args[0]: amount,
	}
	resp, err := json.Marshal(m)
	if err != nil {
		return shim.Error("failed to marshal response")
	}
	return shim.Success(resp)
}

func reduceBalance(stub shim.ChaincodeStubInterface, role string, amount uint64) error {
	key := fmt.Sprintf("%s-%s", PrefixBalance, role)
	currentState, err := stub.GetState(key)
	if err != nil {
		return err
	}
	currentAmount := bytesToUint64(currentState)
	if amount > currentAmount {
		return fmt.Errorf("insufficient balance")
	}
	newAmount := currentAmount - amount
	val := uint64ToBytes(newAmount)
	if err := stub.PutState(key, val); err != nil {
		return err
	}
	return nil
}

func addBalance(stub shim.ChaincodeStubInterface, role string, amount uint64) error {
	key := fmt.Sprintf("%s-%s", PrefixBalance, role)
	currentState, err := stub.GetState(key)
	if err != nil {
		return err
	}
	currentAmount := bytesToUint64(currentState)
	newAmount := currentAmount + amount
	val := uint64ToBytes(newAmount)
	if err := stub.PutState(key, val); err != nil {
		return err
	}
	return nil
}
func transfer(stub shim.ChaincodeStubInterface, from, to string, amount uint64) error {
	if err := reduceBalance(stub, from, amount); err != nil {
		return err
	}
	return addBalance(stub, to, amount)
}
