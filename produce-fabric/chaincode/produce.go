package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/cid"
	"github.com/hyperledger/fabric/protos/peer"
)

// Product 产品
type Product struct {
	Owner     string    `json:"owner"`
	CreatedAt time.Time `json:"createdAt"`
	// BatchID 产品批号
	BatchID string `json:"batchID"`
	// MaterialBatches 这个产品包含的物料的批次号
	MaterialBatches []string `json:"materialBatches"`
	ProductType     string   `json:"productType"`
}

func (c *Contract) getMyProducts(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	preverse, err := getMyProducts(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	data, err := json.Marshal(preverse)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

func (c *Contract) setProductPrice(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get role %v", err))
	}
	if !strings.HasPrefix(role, "product.") {
		return shim.Error(fmt.Sprintf("only for product producer, got %s", role))
	}
	productType := args[0]
	price, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid price, got %s", args[1]))
	}
	key := fmt.Sprintf("%s-%s-%s", PrefixProductPrice, role, productType)
	if err := stub.PutState(key, uint64ToBytes(price)); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state, %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) getProductPrice(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	producer := args[0]
	productType := args[1]
	price, err := getProductPrice(stub, producer, productType)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(strconv.FormatUint(price, 10)))
}

func (c *Contract) registerProduct(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if !strings.HasPrefix(role, "product.") {
		return shim.Error("only for product producer")
	}
	if len(args) < 4 {
		return shim.Error("invalid arguments")
	}
	productType := args[0]
	productID := args[1]
	batchID := args[2]
	materialBatches := args[3:]
	t, err := stub.GetTxTimestamp()
	if err != nil {
		return shim.Error(err.Error())
	}
	product := Product{
		Owner:           role,
		CreatedAt:       time.Unix(t.GetSeconds(), 0),
		BatchID:         batchID,
		MaterialBatches: materialBatches,
		ProductType:     productType,
	}
	pData, err := json.Marshal(product)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal product %v", err))
	}
	pKey := fmt.Sprintf("%s-%s", PrefixProduct, productID)
	existing, err := stub.GetState(pKey)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get state, %v", err))
	}
	if len(existing) > 0 {
		return shim.Error(fmt.Sprintf("product(%s) already exist", productID))
	}
	if err := stub.PutState(pKey, pData); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state %v", err))
	}
	for _, mbatch := range materialBatches {
		mpKey, err := stub.CreateCompositeKey(PrefixMaterialProduct, []string{mbatch, productID})
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to create mpkey, %v", err))
		}
		if err := stub.PutState(mpKey, []byte{1}); err != nil {
			return shim.Error(fmt.Sprintf("failed to put state %v", err))
		}
	}
	ppKey, err := stub.CreateCompositeKey(PrefixProductPreserve, []string{role, productType, productID})
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to create ppkey, %v", err))
	}
	if err := stub.PutState(ppKey, []byte{1}); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state %v", err))
	}
	if err := stub.SetEvent("EvtProductCreated", pData); err != nil {
		return shim.Error(fmt.Sprintf("failed to set event %v", err))
	}
	return shim.Success(nil)
}

func transferProduct(stub shim.ChaincodeStubInterface, from, to, productType string, count uint64) error {
	if from == to {
		return fmt.Errorf("transfer to a same guy is forbidden")
	}
	var i uint64
	iter, err := stub.GetStateByPartialCompositeKey(PrefixProductPreserve, []string{from, productType})
	if err != nil {
		return err
	}
	for iter.HasNext() && i < count {
		kv, err := iter.Next()
		if err != nil {
			return err
		}
		_, attr, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return err
		}
		if len(attr) != 3 {
			return fmt.Errorf("internal key split error")
		}
		productID := attr[2]
		if err := changeProductOwner(stub, productID, to); err != nil {
			return err
		}
		i++
	}
	if i != count {
		return fmt.Errorf("insufficient materials, %d less", count-i)
	}
	return nil
}

func changeProductOwner(stub shim.ChaincodeStubInterface, id, to string) error {
	var product Product
	pkey := fmt.Sprintf("%s-%s", PrefixProduct, id)
	val, err := stub.GetState(pkey)
	if err != nil {
		return fmt.Errorf("failed to get state %w", err)
	}
	if len(val) == 0 {
		return fmt.Errorf("product(%s) does not exist", id)
	}
	if err := json.Unmarshal(val, &product); err != nil {
		return fmt.Errorf("failed to unmarshal product %w", err)
	}
	old := product.Owner
	product.Owner = to
	newData, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("failed to marshal product %w", err)
	}
	if err := stub.PutState(pkey, newData); err != nil {
		return fmt.Errorf("failed to put state %w", err)
	}
	oppkey, err := stub.CreateCompositeKey(PrefixProductPreserve, []string{old, product.ProductType, id})
	if err != nil {
		return fmt.Errorf("failed to create oppkey %w", err)
	}
	if err := stub.DelState(oppkey); err != nil {
		return fmt.Errorf("failed to del state %w", err)
	}
	nppkey, err := stub.CreateCompositeKey(PrefixProductPreserve, []string{to, product.ProductType, id})
	if err != nil {
		return fmt.Errorf("failed to create nppkey %w", err)
	}
	if err := stub.PutState(nppkey, []byte{1}); err != nil {
		return fmt.Errorf("failed to put state %w", err)
	}
	evtData, err := json.Marshal(map[string]interface{}{
		"id":  id,
		"old": old,
		"new": to,
	})
	if err != nil {
		return err
	}
	if err := stub.SetEvent("EvtProductOwnerChanged", evtData); err != nil {
		return err
	}
	return nil
}

func getProductPrice(stub shim.ChaincodeStubInterface, producer, productType string) (uint64, error) {
	key := fmt.Sprintf("%s-%s-%s", PrefixProductPrice, producer, productType)
	val, err := stub.GetState(key)
	if err != nil {
		return 0, fmt.Errorf("failed to get state %w", err)
	}
	if len(val) != 8 {
		return 0, fmt.Errorf("product price not found, set price first")
	}
	return bytesToUint64(val), nil
}

func getMyProducts(stub shim.ChaincodeStubInterface) (map[string]uint64, error) {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return nil, err
	}
	iter, err := stub.GetStateByPartialCompositeKey(PrefixProductPreserve, []string{role})
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	m := make(map[string]uint64)
	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, err
		}
		_, attr, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return nil, err
		}
		if len(attr) != 3 {
			return nil, fmt.Errorf("internal key format wrong")
		}
		m[attr[1]]++
	}
	return m, nil
}
