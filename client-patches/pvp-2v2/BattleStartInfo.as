package ru.pragmatix.wormix.model.battle
{
   import ru.pragmatix.wormix.Scene;
   import ru.pragmatix.wormix.game.mission.ScenarioData;
   import ru.pragmatix.wormix.loader.level.MapConfig;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.net.Server;
   import ru.pragmatix.wormix.settings.PvpWager;
   
   public class BattleStartInfo
   {
      
      public var battleType:BattleType;
      
      private var _enemyProfile:UserProfile;
      
      public var mapId:uint;
      
      public var scene:Scene;
      
      public var mapConfig:MapConfig;
      
      public var noMapGround:Boolean = false;
      
      public var seed:int;
      
      public var pvpServer:Server;
      
      public var wager:PvpWager;
      
      public var myTurn:Boolean;
      
      public var scenario:ScenarioData;
      
      public var battleId:uint;
      
      public var missionId:int;
      
      public var craftItemIdInCrate:Vector.<int> = new Vector.<int>();
      
      // Patch (2v2): every player in turn order (Application.userProfile for this one) and their teams.
      public var teamProfiles:Array;
      
      public var teamIds:Array;
      
      public function isTeamBattle() : Boolean
      {
         return this.teamProfiles != null && this.teamProfiles.length > 2;
      }
      
      public function BattleStartInfo(battleType:BattleType)
      {
         super();
         this.battleType = battleType;
      }
      
      public function toString() : String
      {
         return "BattleStartInfo{enemyProfile=" + String(this._enemyProfile) + ",mapId=" + String(this.mapId) + ",scene=" + String(this.scene) + ",pvpServer=" + String(this.pvpServer) + "," + "battleId=" + this.battleId + "}";
      }
      
      public function get enemyProfile() : UserProfile
      {
         return this._enemyProfile;
      }
      
      public function set enemyProfile(value:UserProfile) : void
      {
         this._enemyProfile = value;
      }
   }
}

