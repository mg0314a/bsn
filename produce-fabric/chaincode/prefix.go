package main

const (
	// PrefixMaterialPreserve 物料库存 (组合: prefix + role + materialType + batchID) => uint64个数
	PrefixMaterialPreserve = "\x01"
	// PrefixBalance 余额 ('%s-%s', prefix, role) => uint64余额
	PrefixBalance = "\x02"
	// PrefixMaterialBatchInfo 物料批次信息 ('%s-%s', prefix, batchID) => Material
	PrefixMaterialBatchInfo = "\x03"
	// PrefixMaterialPrice 物料价格 ('%s-%s-%s', prefix, role, materialType) => uint64价格
	PrefixMaterialPrice = "\x04"
	// PrefixProductPrice 产品价格 ('%s-%s-%s', prefix, role, productType) => uint64价格
	PrefixProductPrice = "\x05"
	// PrefixProduct 产品列表 ('%s-%s', prefix, productID) => Product
	PrefixProduct = "\x06"
	// PrefixMaterialProduct 物料批号到产品ID的映射，用于溯源 (组合 prefix + batchID + productID) => 1
	PrefixMaterialProduct = "\x07"
	// PrefixProductPreserve 产品库存 (组合 prefix + role + productType + productID) => 1
	PrefixProductPreserve = "\x08"
	// PrefixOrder 订单 ('%s-%s', prefix, orderID) => Order
	PrefixOrder = "\x09"
	// PrefixCancelCompensate //下单者取消订单时，补偿给供货商的比例，百分比，直接为key
	PrefixCancelCompensate = "\x10"
	// PrefixOwner 拥有者, 直接为key
	PrefixOwner = "\x11"
)
