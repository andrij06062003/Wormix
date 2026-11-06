package ru.pragmatix.wormix.pvp.messages.server
{
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   
   public class BattleOffer
   {
      
      public var host:String;
      
      public var port:int;
      
      public var mapId:uint;
      
      public var seed:int;
      
      public var enemyProfile:UserProfileStructure;
      
      public var myProfile:UserProfileStructure;
      
      public var battleId:uint;
      
      // Patch (2v2 teams): 0 battle challenge, 1 team invite, 2 teammate calls you into a 2v2 wager battle
      // (the wager in seed), 3 your teammate left the team.
      public var kind:int = 0;
      
      public function BattleOffer()
      {
         super();
      }
      
      public function toString() : String
      {
         return "BattleOffer{mapId=" + String(this.mapId) + ",enemyProfile=" + String(this.enemyProfile) + ",myProfile=" + String(this.myProfile) + "}";
      }
   }
}

