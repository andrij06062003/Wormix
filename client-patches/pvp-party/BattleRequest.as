package ru.pragmatix.wormix.pvp.messages.client
{
   public class BattleRequest
   {
      
      public var host:String;
      
      public var port:int;
      
      public var mapId:uint;
      
      public var seed:int;
      
      public var enemyId:String;
      
      // Patch (2v2 teams): 1 = invite into my team instead of a battle. Sent as a byte at the end when not 0.
      public var mode:int = 0;
      
      public function BattleRequest(host:String = null, port:int = 0, mapId:uint = 0, enemyId:String = "", seed:int = 0)
      {
         super();
         this.host = host;
         this.port = port;
         this.mapId = mapId;
         this.enemyId = enemyId;
         this.seed = seed;
      }
   }
}

