pragma solidity ^0.4.18;

contract MiningPoolManagement {
    //Define manager.
     address public Manager;
    //Only manager can modify. 
     modifier onlyManager {
         require(msg.sender ==Manager);
         _;
      }
//MiningPool Registry Information.
   struct mpool {
         bool status;
         string AppAddr;
}
   mapping (address=>mpool) public MPools;   
   uint public RegistryPoolNum;
//Construction function, initially define the creator as the manager.
    function MiningPoolManagement() public {
            Manager=msg.sender;
     }
//Management power thansfer.
  function transferManagement(address newManager) onlyManager public {
               Manager=newManager;
       }
   //MiningPool regstiry, only manager can modify. 
   function MinerPoolSetting(address MinerPool,bool status,string AppAddr) onlyManager public {
        if (MPools[MinerPool].status!=true){
            RegistryPoolNum+=1;
        } 
        MPools[MinerPool]=mpool(status,AppAddr);
        if (status==false&&RegistryPoolNum>1) {
             RegistryPoolNum-=1;      
        }    
    }
}