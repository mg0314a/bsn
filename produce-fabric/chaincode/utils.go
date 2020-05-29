package main

import (
	"encoding/binary"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

func uint64ToBytes(a uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, a)
	return b
}

func bytesToUint64(b []byte) uint64 {
	if len(b) < 8 {
		return 0
	}
	return binary.BigEndian.Uint64(b)
}

func updateState(stub shim.ChaincodeStubInterface, key string, fn func([]byte) error) error {
	val, err := stub.GetState(key)
	if err != nil {
		return err
	}
	if err = fn(val); err != nil {
		return err
	}
	if err = stub.PutState(key, val); err != nil {
		return err
	}
	return nil
}
