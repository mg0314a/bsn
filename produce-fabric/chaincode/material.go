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

// Material 物料
type Material struct {
	Producer     string    `json:"producer"`
	CreatedAt    time.Time `json:"createdAt"`
	BatchID      string    `json:"batchID"`
	MaterialType string    `json:"materialType"`
	TotalNum     uint64    `json:"totalNum"`
}

func (c *Contract) getMyMaterials(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	preserve, err := getMyMaterials(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	data, err := json.Marshal(preserve)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success(data)
}

func (c *Contract) registerMaterial(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(err.Error())
	}
	if !strings.HasPrefix(role, "material.") {
		return shim.Error("only for material producer")
	}
	if len(args) != 3 {
		return shim.Error("invalid arguments")
	}
	materialType := args[0]
	totalNum, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error("invalid argument(1 totalNum)")
	}
	batchID := args[2]
	t, err := stub.GetTxTimestamp()
	if err != nil {
		return shim.Error(err.Error())
	}
	material := Material{
		Producer:     role,
		CreatedAt:    time.Unix(t.GetSeconds(), 0),
		BatchID:      batchID,
		MaterialType: materialType,
		TotalNum:     totalNum,
	}
	mpkey, err := stub.CreateCompositeKey(PrefixMaterialPreserve, []string{role, materialType, batchID})
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to create key, %v", err))
	}
	if err := stub.PutState(mpkey, uint64ToBytes(totalNum)); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state, %v", err))
	}
	mbkey := fmt.Sprintf("%s-%s", PrefixMaterialBatchInfo, batchID)
	mbval, err := json.Marshal(material)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal value, %v", err))
	}
	if err := stub.PutState(mbkey, mbval); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state, %v", err))
	}
	if err := stub.SetEvent("EvtMaterialCreated", mbval); err != nil {
		return shim.Error(fmt.Sprintf("failed to set event, %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) consumeMaterial(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	materialType := args[0]
	num, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid num, got %s", args[1]))
	}
	total := num
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get role %v", err))
	}
	iter, err := stub.GetStateByPartialCompositeKey(PrefixMaterialPreserve, []string{role, materialType})
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to create key %v", err))
	}
	for iter.HasNext() && num > 0 {
		kv, err := iter.Next()
		if err != nil {
			return shim.Error(fmt.Sprintf("failed to get iter next %v", err))
		}
		keptNum := bytesToUint64(kv.Value)
		if keptNum > num {
			keptNum -= num
			if err := stub.PutState(kv.Key, uint64ToBytes(keptNum)); err != nil {
				return shim.Error(fmt.Sprintf("failed to put state"))
			}
			num = 0
		} else {
			if err := stub.DelState(kv.Key); err != nil {
				return shim.Error(fmt.Sprintf("failed to del state"))
			}
			num -= keptNum
		}
	}
	if num != 0 {
		return shim.Error(fmt.Sprintf("insufficient materials, %d less", num))
	}
	evtData, err := json.Marshal(map[string]interface{}{
		"who":          role,
		"materialType": materialType,
		"num":          total,
	})
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal event data %v", err))
	}
	if err := stub.SetEvent("EvtMaterialConsumed", evtData); err != nil {
		return shim.Error(fmt.Sprintf("failed to set event %v", err))
	}
	return shim.Success(nil)
}

func (c *Contract) getMaterialPrice(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	producer := args[0]
	materialType := args[1]
	price, err := getMaterialPrice(stub, producer, materialType)
	if err != nil {
		return shim.Error(err.Error())
	}
	return shim.Success([]byte(strconv.FormatUint(price, 10)))
}

func (c *Contract) setMaterialPrice(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 2 {
		return shim.Error("invalid arguments")
	}
	materialType := args[0]
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get role %v", err))
	}
	if !strings.HasPrefix(role, "material.") {
		return shim.Error(fmt.Sprintf("only for material producer, got %s", role))
	}
	price, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		return shim.Error(fmt.Sprintf("invalid price, got %v", args[1]))
	}
	key := fmt.Sprintf("%s-%s-%s", PrefixMaterialPrice, role, materialType)
	if err := stub.PutState(key, uint64ToBytes(price)); err != nil {
		return shim.Error(fmt.Sprintf("failed to put state, %v", err))
	}
	return shim.Success(nil)
}

func getMaterialPrice(stub shim.ChaincodeStubInterface, role string, materialType string) (uint64, error) {
	key := fmt.Sprintf("%s-%s-%s", PrefixMaterialPrice, role, materialType)
	val, err := stub.GetState(key)
	if err != nil {
		return 0, fmt.Errorf("failed to get state, %v", err)
	}
	if len(val) != 8 {
		return 0, fmt.Errorf("price for materialType(%s) not found", materialType)
	}
	return bytesToUint64(val), nil
}

func getMyMaterials(stub shim.ChaincodeStubInterface) (map[string]uint64, error) {
	role, err := cid.GetMSPID(stub)
	if err != nil {
		return nil, err
	}
	iter, err := stub.GetStateByPartialCompositeKey(PrefixMaterialPreserve, []string{role})
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
		m[attr[1]] += bytesToUint64(kv.Value)
	}
	return m, nil
}

func transferMaterial(stub shim.ChaincodeStubInterface, from, to string, materialType string, num uint64) error {
	if from == to {
		return fmt.Errorf("transfer to a same guy is forbidden")
	}
	iter, err := stub.GetStateByPartialCompositeKey(PrefixMaterialPreserve, []string{from, materialType})
	if err != nil {
		return err
	}
	total := num
	for iter.HasNext() && num > 0 {
		kv, err := iter.Next()
		if err != nil {
			return err
		}
		ot, attr, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return err
		}
		attr[0] = to
		toKey, err := stub.CreateCompositeKey(ot, attr)
		if err != nil {
			return err
		}
		keptNum := bytesToUint64(kv.Value)
		if keptNum > num {
			keptNum -= num
			if err := stub.PutState(kv.Key, uint64ToBytes(keptNum)); err != nil {
				return err
			}
			if err := stub.PutState(toKey, uint64ToBytes(num)); err != nil {
				return err
			}
			num = 0
		} else {
			if err := stub.DelState(kv.Key); err != nil {
				return err
			}
			if err := stub.PutState(toKey, uint64ToBytes(keptNum)); err != nil {
				return err
			}
			num -= keptNum
		}
	}
	if num != 0 {
		return fmt.Errorf("insufficient materials, %d less", num)
	}
	evtData, err := json.Marshal(map[string]interface{}{
		"from":         from,
		"to":           to,
		"materialType": materialType,
		"num":          total,
	})
	if err != nil {
		return err
	}
	if err := stub.SetEvent("EvtMaterialTransferred", evtData); err != nil {
		return err
	}
	return nil
}
