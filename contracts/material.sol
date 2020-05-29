pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "./access.sol";
import "./openzeppelin/math/SafeMath.sol";

contract material {
    struct RawMaterial {
        address producer; // 生产厂家
        uint256 createdAt;
        uint256 batchID;
        uint256 materialType; // 类型标签, 例如说明类型
        uint256 totalNum; // 生产总量
        uint256 keptNum; // owner的持有数量, 等于0即用完了
    }

    // TODO 增加类型
    mapping(address => mapping(uint256 => uint256)) public usedBatchIdx; // 用户用尽的批次序号统计, 结合keptMaterials使用
    mapping(address => mapping(uint256 => RawMaterial[])) public keptMaterials; // 用户持有的材料
    mapping(address => mapping(uint256 => uint256)) prices;
    mapping(uint256 => RawMaterial) public batchInfos; // 根据batchID查询批次信息
    mapping(uint256 => address) producers;

    event EvtMaterialCreated(address from, uint256 materialType, uint256 num);
    event EvtMaterialTransferred(address from, address to, uint256 materialType, uint256 num);
    event EvtMaterialConsumed(address from, uint256 materialType, uint256 num);
    event EvtPriceUpdated(address from, uint256 materialType, uint256 price);

    Access access; // 权限管理合约实例

    constructor(address accessContractAddress) public {
        access = Access(accessContractAddress);
    }

    //only owner
    function updateProducer(uint256 materialType, address producer) public {
        producers[materialType] = producer;
    }

    function newMaterial(uint256 materialType, uint256 totalNum, uint256 batchID) public returns(RawMaterial memory m) {
        require(access.isMaterialProducer(msg.sender), "only for material producer");
        m = RawMaterial({
            producer: msg.sender,
            createdAt: now,
            batchID: batchID,
            materialType: materialType,
            totalNum: totalNum,
            keptNum: totalNum
        });
        batchInfos[batchID] = m; // 放入批次信息
        keptMaterials[msg.sender][materialType].push(m);
        emit EvtMaterialCreated(msg.sender, materialType, totalNum);
        return m;
    }

    function getMaterialProducer(uint256 materialType) public view returns(address) {
        return producers[materialType];
    }

    // only Payment Call
    function transferMaterial(address _from, address _to, uint256 materialType, uint256 num) public {
        require(access.isPayment(msg.sender), "only for payment");
        require(_from != _to, "transfer to a same guy is forbidden.");
        require(usedBatchIdx[_from][materialType] < keptMaterials[_from][materialType].length, "insufficient materials.");

        uint256 idxFrom = usedBatchIdx[_from][materialType]; // 获取供货方当前批次尚未使用完的最新idx
        RawMaterial storage mFrom = keptMaterials[_from][materialType][idxFrom];
        if (keptMaterials[_to][materialType].length == 0) {
            RawMaterial memory m;
            m.producer = _from;
            m.createdAt = mFrom.createdAt;
            m.batchID = mFrom.batchID;
            m.materialType = mFrom.materialType;
            m.totalNum = mFrom.totalNum;
            m.keptNum = 0;
            keptMaterials[_to][materialType].push(m); // 没有数据先进入一个
        }
        uint256 _idx = keptMaterials[_to][materialType].length - 1;
        RawMaterial storage mTo = keptMaterials[_to][materialType][_idx];
        if (mTo.batchID != mFrom.batchID) {
            RawMaterial memory m;
            m.producer = _from;
            m.createdAt = mFrom.createdAt;
            m.batchID = mFrom.batchID;
            m.materialType = mFrom.materialType;
            m.totalNum = mFrom.totalNum;
            m.keptNum = 0;
            keptMaterials[_to][materialType].push(m); // 没有数据先进入一个
            mTo = keptMaterials[_to][materialType][_idx+1];
        }
        if (mFrom.keptNum >= num) {
            mFrom.keptNum = SafeMath.sub(mFrom.keptNum, num);
            mTo.keptNum = SafeMath.add(mTo.keptNum, num);
            if (mFrom.keptNum == 0) {
                usedBatchIdx[_from][materialType]++; // 消耗完当前批次的材料, idx+1.
            }
        } else {
            uint256 nextBatchNum = num - mFrom.keptNum;
            mTo.keptNum = SafeMath.add(mTo.keptNum, mFrom.keptNum);
            mFrom.keptNum = 0;
            usedBatchIdx[_from][materialType]++; // 消耗完当前批次的材料, idx+1.
            transferMaterial(_from, _to, materialType, nextBatchNum); // 由下一个批次进行提供, 收货方此时并未进行消耗, 故idxTo不改变
        }
        emit EvtMaterialTransferred(_from, _to, materialType, num);
    }

    function consumeMaterial(uint256 materialType, uint256 num) public {
        require(usedBatchIdx[msg.sender][materialType] < keptMaterials[msg.sender][materialType].length, "insufficient materials.");
        require(keptMaterials[msg.sender][materialType][usedBatchIdx[msg.sender][materialType]].producer != msg.sender, "producer cannot be consumer");
        uint256 idx = usedBatchIdx[msg.sender][materialType]; // 获取当前批次尚未使用完的最新idx
        RawMaterial storage m = keptMaterials[msg.sender][materialType][idx];
        if (m.keptNum >= num) {
            m.keptNum = SafeMath.sub(m.keptNum, num);
            if (m.keptNum == 0) {
                usedBatchIdx[msg.sender][materialType]++; // 消耗完当前批次的材料, idx+1.
            }
        } else {
            uint256 nextBatchNum = num - m.keptNum;
            m.keptNum = 0;
            usedBatchIdx[msg.sender][materialType]++; // 消耗完当前批次的材料, idx+1.
            consumeMaterial(materialType, nextBatchNum); // 由下一个批次进行提供, 收货方此时并未进行消耗, 故idxTo不改变
        }
        emit EvtMaterialConsumed(msg.sender, materialType, num);
    }

    function getMyMaterial(uint256 materialType) public view returns (uint256 num) {
        RawMaterial[] memory myMaterials = keptMaterials[msg.sender][materialType];
        uint256 idx = usedBatchIdx[msg.sender][materialType];
        for (;idx < myMaterials.length; idx++) {
            num = SafeMath.add(num, myMaterials[idx].keptNum);
        }
        return num;
    }

    // 通过批次ID查看批次信息
    function showBatchInfo(uint256 batchID) public view
    returns(address producer, uint256 createdAt, uint256 batchId, uint256 materialType, uint256 totalNum, uint256 keptNum) {
        RawMaterial memory m = batchInfos[batchID];
        return (m.producer, m.createdAt, m.batchID, m.materialType, m.totalNum, 0);
    }

    function getPrice(address to, uint256 materialType) public view returns(uint256) {
        return prices[to][materialType];
    }

    // TODO 限定只有生产商才有资格调用这个函数
    function setPrice(uint256 materialType, uint256 price) public {
        require(access.isMaterialProducer(msg.sender), "only for material producer");
        prices[msg.sender][materialType] = price;
        producers[materialType] = msg.sender;
        emit EvtPriceUpdated(msg.sender, materialType, price);
    }

}
