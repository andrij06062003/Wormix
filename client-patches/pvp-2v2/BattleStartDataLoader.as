package ru.pragmatix.wormix.loader
{
   import ru.pragmatix.flox.core.FN;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.wormix.Scene;
   import ru.pragmatix.wormix.controller.*;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.gui.window.Message;
   import ru.pragmatix.wormix.loader.level.MapConfig;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   
   public class BattleStartDataLoader
   {
      
      private var log:Logger = new Logger(BattleStartDataLoader);
      
      private var info:BattleStartInfo;
      
      private var socialDataLoader:UserSocialDataLoader;
      
      public function BattleStartDataLoader()
      {
         super();
      }
      
      public function load(info:BattleStartInfo, onSuccess:Function, onFail:Function) : void
      {
         this.info = info;
         try
         {
            Singletons.mapController.loadMapById(info.mapId,FN.createDelegate(this.onMapLoaded,this,[onSuccess,onFail],true),info.noMapGround);
         }
         catch(e:Object)
         {
            log.error("Error in loading Map id = " + info.mapId + ":" + e.toString());
         }
      }
      
      private function onMapLoaded(scene:Scene, mapConfig:MapConfig, success:Boolean, onSuccess:Function, onFail:Function) : void
      {
         var profiles:Array = null;
         var other:Object = null;
         this.info.scene = scene;
         this.info.mapConfig = mapConfig;
         if(success)
         {
            if(this.info.battleType != BattleType.SCENARIO)
            {
               profiles = [];
               profiles = profiles.concat(this.info.enemyProfile.getTeam());
               if(this.info.battleType == BattleType.PVP_WAGER || this.info.battleType == BattleType.PVP_WITH_FRIEND)
               {
                  profiles = profiles.concat(Application.userProfile.getTeam());
               }
               if(this.info.isTeamBattle())
               {
                  // Patch (2v2): the teammate and the second enemy too.
                  for each(other in this.info.teamProfiles)
                  {
                     if(other != Application.userProfile && other != this.info.enemyProfile)
                     {
                        profiles = profiles.concat(other.getTeam());
                     }
                  }
               }
               this.socialDataLoader = new UserSocialDataLoader();
               Message.showBattleLoadingMsg(Messages.BATTLE_ENEMY_DATA_LOAD);
               this.socialDataLoader.loadSocialData(profiles,FN.createDelegate(onSuccess,this,[]));
            }
            else
            {
               onSuccess();
            }
         }
         else
         {
            onFail();
         }
      }
   }
}

