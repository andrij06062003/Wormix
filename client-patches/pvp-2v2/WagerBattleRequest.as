package ru.pragmatix.wormix.pvp.messages.wager
{
   public class WagerBattleRequest
   {
      
      public var host:String;
      
      public var port:int;
      
      public var wager:int;
      
      public var mapId:uint;
      
      // Patch: 0 = 1v1, 1 = 2v2. Sent as a byte after mapId only when not 0.
      public var mode:int = 0;
      
      public function WagerBattleRequest(host:String = null, port:int = 0, mapId:uint = 0, wager:int = 0)
      {
         super();
         this.host = host;
         this.port = port;
         this.wager = wager;
         this.mapId = mapId;
      }
   }
}

