#!/bin/bash

function clearContainers() {
  CONTAINER_IDS=$(docker ps -a | awk '($2 ~ /dev-.*producecc-1\.0.*/) {print $1}')
  if [ -z "$CONTAINER_IDS" -o "$CONTAINER_IDS" == " " ]; then
    echo "---- No containers available for deletion ----"
  else
    docker rm -f $CONTAINER_IDS
  fi
}

function removeUnwantedImages() {
  DOCKER_IMAGE_IDS=$(docker images | awk '($1 ~ /dev.*producecc-1\.0.*/) {print $3}')
  if [ -z "$DOCKER_IMAGE_IDS" -o "$DOCKER_IMAGE_IDS" == " " ]; then
    echo "---- No images available for deletion ----"
  else
    docker rmi -f $DOCKER_IMAGE_IDS
  fi
}

function cryptogen() {
    rm -rf crypto-config
	./bin/cryptogen generate --config=./crypto-config.yaml
}

function configtxgen() {
    rm -rf channel-artifacts
	mkdir -p channel-artifacts
	FABRIC_CFG_PATH=${PWD} ./bin/configtxgen -profile ProduceGenesis -channelID sys-channel -outputBlock ./channel-artifacts/genesis.block
	FABRIC_CFG_PATH=${PWD} ./bin/configtxgen -profile ProduceChannel -outputCreateChannelTx ./channel-artifacts/channel.tx -channelID produce-channel
}

function networkUp() {
	IMAGE_TAG=1.4.3 docker-compose -f docker-compose.yaml up -d orderer lcd audio cpu tv pc payment store cli
  if [ $? -ne 0 ]; then
    echo "ERROR !!!! Unable to start network"
    exit 1
  fi
}

function networkDown() {
    IMAGE_TAG=1.4.3 docker-compose -f docker-compose.yaml down --volumes --remove-orphans
    clearContainers
    removeUnwantedImages
}

function run() {
  echo "starting network..."
  networkUp >/dev/null 2>&1
  echo "network started"

  echo "run test script..."
  docker exec cli scripts/script.sh

  echo "stopping network..."
  networkDown >/dev/null 2>&1
  echo "done"
}

MODE=$1

if [ "${MODE}" == "up" ]; then
  networkUp
elif [ "${MODE}" == "down" ]; then ## Clear the network
  networkDown
elif [ "${MODE}" == "generate" ]; then ## Generate Artifacts
  cryptogen
  configtxgen
elif [ "${MODE}" == "run" ]; then
  run
else
  echo "wrong command, run | up | down | generate"
  exit 1
fi