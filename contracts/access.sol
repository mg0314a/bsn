pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import './openzeppelin/access/Ownable.sol';

contract Access is Ownable {
    mapping(address => uint8) roles;
    /*
    Payment          00000001 0x01
    ProductProducer  00000010 0x02
    MaterialProducer 00000100 0x04
    */
    uint8 paymentMask = 0x01;
    uint8 productProducerMask = 0x02;
    uint8 materialProducerMask = 0x04;

    function isPayment(address account) public view returns(bool) {
        uint8 role = roles[account];
        return (role & paymentMask) != 0;
    }

    function isProductProducer(address account) public view returns(bool) {
        uint8 role = roles[account];
        return (role & productProducerMask) != 0;
    }
    function isMaterialProducer(address account) public view returns(bool) {
        uint8 role = roles[account];
        return (role & materialProducerMask) != 0;
    }

    function grantPayment(address account) public onlyOwner {
        uint8 role = roles[account];
        role ^= paymentMask;
        roles[account] = role;
    }

    function grantProductProducer(address account) public onlyOwner {
        uint8 role = roles[account];
        role ^= productProducerMask;
        roles[account] = role;
    }

    function grantMaterialProducer(address account) public onlyOwner {
        uint8 role = roles[account];
        role ^= materialProducerMask;
        roles[account] = role;
    }
}