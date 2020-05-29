pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import './openzeppelin/math/SafeMath.sol';
import './openzeppelin/access/Ownable.sol';
import './material.sol';
import './produce.sol';

struct Order {
    address payer; //下单者
    address producer; //供货商
    uint256 amount; //订单总金额
    uint256 count; //商品(元件或者产品)数量
    uint256 orderType; //元件种类，如果是产品订单则这个字段无效
    uint256 createdAt; //订单创建时间
    uint8   status; //订单状态 0处理中，1已完成，2已取消
}

contract Payment is Ownable {
    using SafeMath for uint256;

    event EvtMint(address account, uint256 amount);
    event EvtMakeOrder(uint256 orderType, uint256 indexed id, address indexed payer, address indexed producer, uint256 cnt, uint256 price);
    event EvtConfirmOrder(address from, uint256 orderType, uint256 indexed id);
    event EvtCancelOrder(uint256 indexed id, uint256 amount2Payer, uint256 amount2Producer);

    mapping(address => uint256) balances; //各商户的可用余额
    mapping(uint256 => Order) orders; //订单，订单ID=>订单实例，订单完成或者取消则删除
    uint256 materialOrderID = 0;  // 元件订单ID，在偶数空间递增
    uint256 productOrderID = 1; // 产品订单ID，在奇数空间递增
    material materialProducer; //元件供货商
    Produce productProducer; //产品代工厂
    uint256 cancelCompensate; //代工厂取消订单时，补偿给供货商的比例，百分比

    constructor(uint256 _cancelCompensate) public {
        cancelCompensate = _cancelCompensate;
    }

    function setMaterialProducer(address producer) public onlyOwner {
        materialProducer = material(producer);
    }

    function setProductProducer(address producer) public onlyOwner {
        productProducer = Produce(producer);
    }

    function makeOrder (bool isMaterial, address to, uint256 orderType, uint256 _count, uint256 _price) public returns(uint256 id) {
        require(to != address(0), "can not make order to nobody.");
        uint256 remotePrice = 0;
        id = 0;
        if (isMaterial) {
            require(address(materialProducer) != address(0), "material contract address not set");
            remotePrice = materialProducer.getPrice(to, orderType);
            id = materialOrderID;
            materialOrderID += 2;
        } else {
            require(address(productProducer) != address(0), "produce contract address not set");
            remotePrice = productProducer.getProductPrice(to, orderType);
            id = productOrderID;
            productOrderID += 2;
        }
        require(_price >= remotePrice, "price mismatch");
        uint256 _amount = remotePrice.mul(_count);
        require(balances[msg.sender] >= _amount, "Insufficient balance");
        
        balances[msg.sender] = balances[msg.sender].sub(_amount);

        Order memory order = Order({
            payer: msg.sender,
            producer: to,
            amount: _amount,
            count: _count,
            orderType: orderType,
            createdAt: now,
            status: 0
        });
        orders[id] = order;

        emit EvtMakeOrder(orderType, id, order.payer, order.producer, _count, _price);

        return id;
    }

    // 用于交付产品或材料时，进行资金划拨和所有权变更
    function confirmOrder(uint256 id) public {
        require(orders[id].payer != address(0), "order does not exis");
        require(orders[id].payer == msg.sender, "only the payer can confirm order");
        require(orders[id].status == 0, "order status wrong");
        orders[id].status = 1;
        if (id & 1 == 0) { //元件订单
            materialProducer.transferMaterial(orders[id].producer, orders[id].payer, orders[id].orderType, orders[id].count);
        } else { //产品订单
            productProducer.transferProducts(orders[id].producer, orders[id].payer, orders[id].orderType, orders[id].count);
        }
        //将资金支付给供货商
        balances[orders[id].producer] = balances[orders[id].producer].add(orders[id].amount);
        emit EvtConfirmOrder(msg.sender, orders[id].orderType, id);

        delete orders[id];
    }

    function cancelOrder(uint256 id) public {
        require(orders[id].payer != address(0), "order does not exis");
        require(orders[id].producer == msg.sender || orders[id].payer == msg.sender, "only the payer and producer can cancel order");
        require(orders[id].status == 0, "order status wrong");
        orders[id].status = 2;
        //TODO 这里需要通知下单者吗
        if (msg.sender == orders[id].producer) {
            //生产商取消订单则将资金全部退回下单者
            balances[orders[id].payer] = balances[orders[id].payer].add(orders[id].amount);

            emit EvtCancelOrder(id, orders[id].amount, 0);
        } else {
            //下单者取消订单，则按照一定赔付比例赔付给生产商
            uint256 compensate = (orders[id].amount * cancelCompensate) / 100;
            uint256 remain = orders[id].amount.sub(compensate);
            balances[orders[id].producer] = balances[orders[id].producer].add(compensate);
            balances[orders[id].payer] = balances[orders[id].payer].add(remain);

            emit EvtCancelOrder(id, remain, compensate);
        }
        delete orders[id];
    }

    function getOrder(uint256 id) public view returns(Order memory order) {
        require(orders[id].payer != address(0), "order does not exis");
        return orders[id];
    }

    function mint(address account, uint256 amount) public onlyOwner {
        require(account != address(0), "mint to the zero address");
        balances[account] = balances[account].add(amount);
        emit EvtMint(account, amount);
    }

    function burn(address account, uint256 amount) public onlyOwner {
        require(account != address(0), "burn from the zero address");
        if (amount > balances[account]) {
            amount = balances[account];
        }
        balances[account] = balances[account].sub(amount);
    }

    function balanceOf(address account) public view returns(uint256 amount) {
        return balances[account];
    }
}