package ru.pragmatix.wormix.pvp.messages.wager
{
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   
   public class WagerBattleRequestResult
   {
      
      public var host:String;
      
      public var port:int;
      
      public var mapId:uint;
      
      public var enemyProfile:UserProfileStructure;
      
      public var userProfile:UserProfileStructure;
      
      public var firstPlayerId:uint;
      
      public var seed:int;
      
      public var battleId:uint;
      
      public var reagentsForBattle:Vector.<int> = new Vector.<int>();
      
      // Patch (2v2): every player in turn order and their team (1 or 2). Empty in a 1v1 battle.
      public var teamProfiles:Array = [];
      
      public var teamIds:Array = [];
      
      public function WagerBattleRequestResult()
      {
         super();
      }
   }
}

