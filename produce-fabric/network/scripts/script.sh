#!/bin/bash

DELAY="3"
TIMEOUT="10"
VERBOSE="false"
COUNTER=1
MAX_RETRY=5
ORDER_ID=""
TV_ORDER_ID=""

CC_SRC_PATH="produce/"

createChannel() {
    CORE_PEER_LOCALMSPID=material.lcd
	CORE_PEER_ADDRESS=material-lcd:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/lcd.example.com/users/Admin@lcd.example.com/msp
	echo
	echo "===================== Creating channel ===================== "
	echo peer channel create -o produce-orderer:7050 -c produce-channel -f ./channel-artifacts/channel.tx
	peer channel create -o produce-orderer:7050 -c produce-channel -f ./channel-artifacts/channel.tx
	echo "===================== Channel created ===================== "
}

joinChannel () {
	for org in material.lcd material.audio material.cpu product.tv product.pc payment store
	do
		CORE_PEER_LOCALMSPID=$org
		CORE_PEER_ADDRESS=${org/./-}:7051
		CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${org#*.}.example.com/users/Admin@${org#*.}.example.com/msp
		echo 
		echo "===================== Org $org joining channel ===================== "
		echo peer channel join -b produce-channel.block -o produce-orderer:7050
		peer channel join -b produce-channel.block -o produce-orderer:7050
		echo "===================== Channel joined ===================== "
	done
}

installChaincode() {
	for org in material.lcd material.audio material.cpu product.tv product.pc payment store
	do
		CORE_PEER_LOCALMSPID=$org
		CORE_PEER_ADDRESS=${org/./-}:7051
		CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${org#*.}.example.com/users/Admin@${org#*.}.example.com/msp
		echo 
		echo "===================== Org $org installing chaincode ===================== "
		echo peer chaincode install -n producecc -v 1.0 -l golang -p ${CC_SRC_PATH}
		peer chaincode install -n producecc -v 1.0 -l golang -p ${CC_SRC_PATH}
		echo "===================== Chaincode isntalled ===================== "
	done
}

instantiateChaincode() {
	CORE_PEER_LOCALMSPID=material.lcd
	CORE_PEER_ADDRESS=material-lcd:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/lcd.example.com/users/Admin@lcd.example.com/msp
	echo 
	echo "===================== Instantiating chaincode ===================== "
	echo peer chaincode instantiate -o produce-orderer:7050 -C produce-channel -n producecc -l golang -v 1.0 -c '{"Args":["init","123","abc"]}' -P "OR('material.lcd.peer','material.audio.peer','material.cpu.peer','product.tv.peer','product.pc.peer','payment.peer','store.peer')"
	peer chaincode instantiate -o produce-orderer:7050 -C produce-channel -n producecc -l golang -v 1.0 -c '{"Args":["init","123","abc"]}' -P "OR('material.lcd.peer','material.audio.peer','material.cpu.peer','product.tv.peer','product.pc.peer','payment.peer','store.peer')"
	echo "===================== Chaincode instantiated ===================== "
}

setPrice() {
	CORE_PEER_LOCALMSPID=$1.$2
	CORE_PEER_ADDRESS=$1-$2:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/$2.example.com/users/User1@$2.example.com/msp
	if [ $1 = "material" ]; then
		FUNC=setMaterialPrice
	else
		FUNC=setProductPrice
	fi
	echo 
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["'$FUNC'","'$3'","'$4'"]}' \
        --peerAddresses $1-$2:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["'$FUNC'","'$3'","'$4'"]}' \
        --peerAddresses $1-$2:7051 
	echo "===================== Chaincode invoked ===================== "
}

mint() {
	CORE_PEER_LOCALMSPID=payment
	CORE_PEER_ADDRESS=payment:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/payment.example.com/users/User1@payment.example.com/msp
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["mint","'$1'","'$2'"]}' \
        --peerAddresses payment:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["mint","'$1'","'$2'"]}' \
        --peerAddresses payment:7051 
	echo "===================== Chaincode invoked ===================== "
}

setCancelCompensate() {
	CORE_PEER_LOCALMSPID=payment
	CORE_PEER_ADDRESS=payment:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/payment.example.com/users/User1@payment.example.com/msp
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["setCancelCompensate","'$1'"]}' \
        --peerAddresses payment:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["setCancelCompensate","'$1'"]}' \
        --peerAddresses payment:7051 
	echo "===================== Chaincode invoked ===================== "
}

makeOrder() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	if [[ $2 == product* ]]; then
		FUNC=makeProductOrder
	else
		FUNC=makeMaterialOrder
	fi
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["'$FUNC'","'$2'","'$3'","'$4'","'$5'"]}' --peerAddresses ${1/./-}:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["'$FUNC'","'$2'","'$3'","'$4'","'$5'"]}' --peerAddresses ${1/./-}:7051 > order.txt 2>&1
	ORDER_JSON=$(grep -Eo "\{.*\}" order.txt | sed 's/\\//g')
	ORDER_ID=$(echo $ORDER_JSON | jq '.orderID')
	if [ $1 = "store" ]; then
		TV_ORDER_ID=$ORDER_ID
	fi
	echo "===================== Chaincode invoked ===================== "
	echo "Make order successful:"
	echo $ORDER_JSON | jq '.'
}

registerGoods() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	if [[ $1 == product* ]]; then
		FUNC=registerProduct
	else
		FUNC=registerMaterial
	fi
	shift 1
	callopt="{\"Args\":[\"${FUNC}\""
    for arg in $*
    do
        callopt="${callopt},\"${arg}\""
    done
    callopt="${callopt}]}"
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c $callopt \
        --peerAddresses $CORE_PEER_ADDRESS
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c $callopt \
        --peerAddresses $CORE_PEER_ADDRESS
	echo "===================== Chaincode invoked ===================== "
}

confirmOrder() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	if [ $1 = "store" ]; then 
		ORDER_ID=$TV_ORDER_ID
	fi
	echo "Confirm order: ${ORDER_ID}"
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["confirmOrder",'${ORDER_ID}']}' \
        --peerAddresses ${1/./-}:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["confirmOrder",'${ORDER_ID}']}' \
        --peerAddresses ${1/./-}:7051 
	echo "===================== Chaincode invoked ===================== "
}

cancelOrder() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	echo "Cancel order: ${ORDER_ID}"
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["cancelOrder",'${ORDER_ID}']}' \
        --peerAddresses ${1/./-}:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["cancelOrder",'${ORDER_ID}']}' \
        --peerAddresses ${1/./-}:7051 
	echo "===================== Chaincode invoked ===================== "
}

getBalance() {
	CORE_PEER_LOCALMSPID=payment
	CORE_PEER_ADDRESS=payment:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/payment.example.com/users/User1@payment.example.com/msp
	echo "====================== Balance ======================"
	for org in material.lcd material.audio material.cpu product.tv product.pc store
	do
		peer chaincode query -C produce-channel -n producecc -c '{"Args":["balanceOf","'$org'"]}'
	done
	echo "======================   End   ======================"
}

getMyProducts() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	peer chaincode query -C produce-channel -n producecc -c '{"Args":["getMyProducts"]}'
}

getMyMaterials() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	peer chaincode query -C produce-channel -n producecc -c '{"Args":["getMyMaterials"]}'
}

consumeMaterial() {
	CORE_PEER_LOCALMSPID=$1
	CORE_PEER_ADDRESS=${1/./-}:7051
	CORE_PEER_MSPCONFIGPATH=/opt/gopath/src/github.com/hyperledger/fabric/peer/crypto/peerOrganizations/${1#*.}.example.com/users/User1@${1#*.}.example.com/msp
	echo "===================== Invoking chaincode ===================== "
	echo peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["consumeMaterial","'$2'","'$3'"]}' \
        --peerAddresses ${1/./-}:7051 
	peer chaincode invoke -o produce-orderer:7050 -C produce-channel --waitForEvent -n producecc -c '{"Args":["consumeMaterial","'$2'","'$3'"]}' \
        --peerAddresses ${1/./-}:7051 
	echo "===================== Chaincode invoked ===================== "
}

print() {
	echo
	echo 
	echo "********************************************************"
	echo $1
	echo "********************************************************"
}

## Create channel
sleep 1
print "Creating channel..."
createChannel

## Join all the peers to the channel
print "Having all peers join the channel..."
joinChannel

## Install chaincode on all peers
print "Installing chaincode..."
installChaincode

# Instantiate chaincode
print "Instantiating chaincode..."
instantiateChaincode

sleep 3

print "set cancel compensate"
setCancelCompensate 50

print "set LCD price"
setPrice material lcd LCD 100

print "set Audio price"
setPrice material audio Audio 50

print "set CPU price"
setPrice material cpu CPU 200

print "set TV price"
setPrice product tv TV 3000

print "set PC price"
setPrice product pc PC 5000

for org in material.lcd material.audio material.cpu product.tv product.pc store
do
	print "为${org}充值"
	mint $org 100000
done

print "查看balance"
getBalance

print "store 向 product.tv 下单5个TV"
makeOrder store product.tv TV 5 3000

print "product.tv 向 material.lcd 下单100个LCD"
makeOrder product.tv material.lcd LCD 100 100

print "material.lcd 工厂生产中..."
sleep 3

print "material.lcd 生产完成，将LCD按照生产批号注册上链"
registerGoods material.lcd LCD 300 LCD_1

print "material.lcd 链下交货，product.tv验货后，确认收货"
confirmOrder product.tv

print "查看product.tv的物料库存"
getMyMaterials product.tv

print "查看balance"
getBalance

print "product.tv 向 material.audio 下单100个Audio"
makeOrder product.tv material.audio Audio 100 50

print "material.audio 工厂生产中..."
sleep 3

print "material.audio 生产完成，将Audio按照生产批号注册上链"
registerGoods material.audio Audio 300 Audio_1

print "material.audio 链下交货，product.tv验货后，确认收货"
confirmOrder product.tv

print "查看product.tv的物料库存"
getMyMaterials product.tv

print "查看balance"
getBalance

print "product.tv 向 material.cpu 下单100个CPU"
makeOrder product.tv material.cpu CPU 100 200

print "material.cpu 工厂生产中..."
sleep 3

print "material.cpu 生产完成，将CPU按照生产批号注册上链"
registerGoods material.cpu CPU 300 CPU_1

print "material.audio 链下交货，product.tv验货后，确认收货"
confirmOrder product.tv

print "查看product.tv的物料库存"
getMyMaterials product.tv

print "查看balance"
getBalance

print "product.tv 消耗物料"
consumeMaterial product.tv LCD 20
consumeMaterial product.tv Audio 20
consumeMaterial product.tv CPU 20

print "product.tv 工厂生产中..."
sleep 3

print "product.tv 生产完成，将TV注册上链"
registerGoods product.tv TV TV_1 2020-05 LCD_1 Audio_1 CPU_1
registerGoods product.tv TV TV_2 2020-05 LCD_1 Audio_1 CPU_1
registerGoods product.tv TV TV_3 2020-05 LCD_1 Audio_1 CPU_1
registerGoods product.tv TV TV_4 2020-05 LCD_1 Audio_1 CPU_1
registerGoods product.tv TV TV_5 2020-05 LCD_1 Audio_1 CPU_1

print "product.tv 链下交货，store收货后，在链上确认收货"
confirmOrder store

print "查看product.tv的物料库存"
getMyMaterials product.tv

print "查看store的产品库存"
getMyProducts store

print "查看balance"
getBalance

print "store 向 product.pc 下单2个PC"
makeOrder store product.pc PC 2 5000

print "查看balance"
getBalance

print "store 取消订单"
cancelOrder store

print "查看balance"
getBalance

print "store 再向 product.pc 下单2个PC"
makeOrder store product.pc PC 2 5000

print "查看balance"
getBalance

print "product.pc 取消订单"
cancelOrder product.pc

print "查看balance"
getBalance

echo
echo "========= produce network sample setup completed =========== "
echo

exit 0