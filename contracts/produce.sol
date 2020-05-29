pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import './material.sol';
import './payment.sol';
import './queue.sol';
import './access.sol';
import './openzeppelin/access/Ownable.sol';

// 生产合约
contract Produce is Ownable {
    struct Product {
        address owner;
        address producer;
        uint256 createdAt;
        uint256 batch; //产品批次
        uint256[] materialBatches; //所用到的零件批次
        bool sold; //是否已经售卖
    }

    using Queue for imap;

    event EvtProductOwnerChanged(uint256 indexed id, address oldOwner, address newOwner);
    event EvtProductCreated(uint256 productType, uint256 indexed productID, address creator);

    material materialContract; //物料合约实例
    Payment paymentContract; //结算合约实例
    Access access; //权限管理合约实例

    mapping(address => mapping(uint256 => imap)) keptProducts; //持有产品ID（产品库存)
    mapping(uint256 => Product) public products; //产品，产品ID=>产品实例
    mapping(uint256 => bool) isOrderFlying; //是否有订单还未确认，每种元件只允许同时有一个订单未确认
    mapping(address => mapping(uint256 => uint256)) productPrice; //产品价格, 暂定只有一种产品，如果有多种产品，将结构改为 mapping(address => mapping(uint256 => uint256))
    uint256 materialTypeCount; //产品包含的元件种类个数
    mapping(uint256 => imap) materialTrace; //溯源 物料批次号=>产品ID的队列

    modifier mcMustBeSet() {
        require(
            address(materialContract) != address(0),
            "material contract not be set, call 'setMaterialContract' first"
        );
        _;
    }

    // pc = paymentContract
    modifier pcMustBeSet() {
        require(
            address(paymentContract) != address(0),
            "payment contract not be set, call 'setPaymentContract' first"
        );
        _;
    }

    constructor(address accessContractAddress, uint256 _materialTypeCount) public {
        access = Access(accessContractAddress);
        materialTypeCount = _materialTypeCount;
    }

    function updateProductPrice(uint256 productType, uint256 newPrice) public {
        require(access.isProductProducer(msg.sender), "only for product producer");
        productPrice[msg.sender][productType] = newPrice;
    }

    function getProductPrice(address _to, uint256 productType) public view returns(uint256 price) {
        return productPrice[_to][productType];
    }

    function setMaterialContract(address _materialContract) public onlyOwner {
        materialContract = material(_materialContract);
    }

    function setPaymentContract(address _paymentContract) public onlyOwner {
        paymentContract = Payment(_paymentContract);
    }

    function registerProduct(uint256 productType, uint256 id, uint256 batchNumber, uint256[] memory materialBatches) public {
        require(access.isProductProducer(msg.sender), "only for product producer");
        require(products[id].owner == address(0), "product already exists");

        products[id] = Product({
            owner: msg.sender,
            producer: msg.sender,
            createdAt: now,
            batch: batchNumber,
            materialBatches: materialBatches,
            sold: false
        });
        keptProducts[msg.sender][productType].enqueue(id);

        for (uint i = 0; i < materialBatches.length; i++) {
            materialTrace[materialBatches[i]].enqueue(id);
        }

        emit EvtProductCreated(productType, id, msg.sender);
    }

    function transferProducts(address from, address to, uint256 productType, uint256 count) public {
        require(access.isPayment(msg.sender), "only for payment");
        require(count <= keptProducts[from][productType].len(), "insufficient product");
        for (uint i = 0; i < count; i++) {
            uint256 id = keptProducts[from][productType].dequeue();
            keptProducts[to][productType].enqueue(id);
            changeProductOwner(id, to);
        }
    }

    function details(uint256 id) public view returns(Product memory) {
        require(products[id].owner != address(0), "product does not exist");
        Product memory p = products[id];
        return p;
    }

    function changeProductOwner(uint256 id, address newOwner) private {
        require(products[id].owner != address(0), "product does not exist");
        require(products[id].owner != newOwner, "self-transfer is disallowed");

        address oldOwner = products[id].owner;
        products[id].owner = newOwner;

        emit EvtProductOwnerChanged(id, oldOwner, newOwner);
    }

    function getMyProducts(uint256 productType) public view returns(uint256[] memory myProductIDs) {
//        require(keptProducts[msg.sender][productType].len() > 0, "You have no keptProducts.");
        myProductIDs = new uint256[](keptProducts[msg.sender][productType].len());
        uint256 i = 0;
        for (uint256 j = keptProducts[msg.sender][productType].head; j < keptProducts[msg.sender][productType].tail; j++) {
            myProductIDs[i++] = keptProducts[msg.sender][productType].map[j];
        }
        return myProductIDs;
    }

    function trace(uint256 materialBatchNum) public view returns(uint256[] memory ids) {
        ids = new uint256[](materialTrace[materialBatchNum].len());
        for (uint i = materialTrace[materialBatchNum].head; i < materialTrace[materialBatchNum].tail; i++) {
            ids[i-materialTrace[materialBatchNum].head] = materialTrace[materialBatchNum].map[i];
        }
        return ids;
    }
}