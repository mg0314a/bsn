package main

import (
	"fmt"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Contract 合约
type Contract struct{}

// Init Init
func (c *Contract) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

// Invoke Invoke
func (c *Contract) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	fn, args := stub.GetFunctionAndParameters()
	switch fn {
	//material
	case "getMyMaterials":
		return c.getMyMaterials(stub, args)
	case "registerMaterial":
		return c.registerMaterial(stub, args)
	case "consumeMaterial":
		return c.consumeMaterial(stub, args)
	case "setMaterialPrice":
		return c.setMaterialPrice(stub, args)
	case "getMaterialPrice":
		return c.getMaterialPrice(stub, args)
	//product
	case "getMyProducts":
		return c.getMyProducts(stub, args)
	case "setProductPrice":
		return c.setProductPrice(stub, args)
	case "getProductPrice":
		return c.getProductPrice(stub, args)
	case "registerProduct":
		return c.registerProduct(stub, args)
	//payment
	case "makeMaterialOrder":
		return c.makeMaterialOrder(stub, args)
	case "makeProductOrder":
		return c.makeProductOrder(stub, args)
	case "confirmOrder":
		return c.confirmOrder(stub, args)
	case "cancelOrder":
		return c.cancelOrder(stub, args)
	case "getOrder":
		return c.getOrder(stub, args)
	case "balanceOf":
		return c.balanceOf(stub, args)
	//only for payment
	case "setCancelCompensate":
		return c.setCancelCompensate(stub, args)
	case "mint":
		return c.mint(stub, args)
	case "burn":
		return c.burn(stub, args)
	}
	return shim.Error("unsupported method")
}

func main() {
	err := shim.Start(new(Contract))
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
