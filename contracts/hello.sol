pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

contract Hello {
   string message;

   struct Ret {
      uint256 name;
      uint256 gender;
   }

   Ret owner;

   constructor() public {
      message = "hello.";
      owner.name = 123;
      owner.gender = 346;
   }

   event EvtSet(address from, string msg);

   function set(string memory name) public {
      message = name;
      emit EvtSet(msg.sender, message);
   }

   function get() public view returns(string memory, bool) {
      return (message, true);
   }

   function setName(uint256 name, uint256 gender) public returns(uint256, uint256) {
      owner.name = name;
      owner.gender = gender;
      return (owner.name, owner.gender);
   }

   function getOwner() public view returns(Ret memory ret) {
      ret = owner;
      return (ret);
   }
}