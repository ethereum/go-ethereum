pragma solidity ^0.4.18;

contract MobileMine {
    //Define the Manager
     address public Manager;
    //Only manager can modify. 
     modifier onlyManager {
         require(msg.sender ==Manager);
         _;
      }
//Define miner mining informaiton.
      struct Miner {
           bool Registry;
           uint TotalPay;   //Having mined reward.
           uint PayTime;   //Reward pay time.
        }
//Active users' information.
    struct ActiveInfo {
          uint LastTime;     //Last calculate time.
          uint ActiveNum;  //Active users number.
          uint RegistryUsers; //the number of already registrtyUsers.
        }
  /*Miner and active users defining  */
   mapping (address => Miner) public Miners;
   ActiveInfo public ActiveUsers;
   uint public ReceiveFoundation;    //Having received reward foundation. 
   uint MineAmount;
//Constuct function£¬initially define the creator as the manager.
   function MobileMine() public {
            Manager=msg.sender;
     }
  //Define the contract can receive mining reward foundation.
   function () payable public {
         ReceiveFoundation+=msg.value;
   }
//Management power transfer.
  function transferManagement(address newManager) onlyManager public {
               Manager=newManager;
       }
  //Miner registry setting, only manager can modify.
  function MinerSetting(address MobileMiner) onlyManager public {
        if (Miners[MobileMiner].Registry!=true){   
             Miners[MobileMiner].Registry=true;
             ActiveUsers.RegistryUsers+=1;
        }
    }
/* Miner mine function, modify miner's status*/
  function Mine()  public returns (bool success){
        //If not registry or has been payed in one day, return false.
       if (Miners[msg.sender].Registry!=true||Miners[msg.sender].PayTime+86400>now){
             return false;
        }  
        //Pay the reward and change the miner's status.
        MineAmount=this.balance/(ActiveUsers.RegistryUsers+1);
        msg.sender.transfer(MineAmount);
        Miners[msg.sender].TotalPay+= MineAmount;
        Miners[msg.sender].PayTime=now;
       //Check if the calculating time of active user is lasting one day or not. 
        if(ActiveUsers.LastTime+86400<now){
                      ActiveUsers.LastTime=now;
                      ActiveUsers.ActiveNum=1;
                      Manager.transfer(this.balance/100);                      
         }else{
                     ActiveUsers.ActiveNum+=1;
             }
        return true;
  }
} 
