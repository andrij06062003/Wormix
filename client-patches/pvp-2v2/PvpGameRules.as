package ru.pragmatix.wormix.game.rules
{
   import ru.pragmatix.wormix.game.BattleReplay;
   import ru.pragmatix.wormix.game.GameMaster;
   import ru.pragmatix.wormix.game.rules.gameresult.PvpResultDeterminator;
   import ru.pragmatix.wormix.game.settings.ModeSettings;
   import ru.pragmatix.wormix.game.settings.TimerSettings;
   import ru.pragmatix.wormix.game.squad.PlayerType;
   import ru.pragmatix.wormix.game.squad.Squad;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.social.Social;
   
   public class PvpGameRules extends AbstractDuelRules
   {
      
      protected var iStart:Boolean;
      
      protected var isWager:Boolean;
      
      private var replay:BattleReplay = new BattleReplay();
      
      private var info:BattleStartInfo;
      
      public function PvpGameRules(info:BattleStartInfo)
      {
         super(info);
         this.info = info;
         this.iStart = info.myTurn;
         this.isWager = info.battleType == BattleType.PVP_WAGER;
         this.replay.setMapId(info.mapId);
      }
      
      override public function createGM() : GameMaster
      {
         var sq1:Squad = null;
         var sq2:Squad = null;
         var playerSquad:uint = 0;
         var squads:Array = null;
         var i:int = 0;
         var profile:UserProfile = null;
         var sq:Squad = null;
         if(this.info.isTeamBattle())
         {
            // Patch (2v2): a squad per player in the server's turn order. The GameMaster gives turns to the
            // squads in this order and ends the battle when only one team (teamId) has worms left.
            squads = [];
            playerSquad = 0;
            for(i = 0; i < this.info.teamProfiles.length; i++)
            {
               profile = UserProfile(this.info.teamProfiles[i]);
               sq = new Squad("Squad " + (i + 1),i,profile == Application.userProfile ? PlayerType.LOCAL_PLAYER : PlayerType.REMOTE_PLAYER,uint(this.info.teamIds[i]));
               initSquad(sq,profile,posLists);
               if(profile == Application.userProfile)
               {
                  playerSquad = i;
               }
               squads.push(sq);
            }
            this.replay.setSquads.apply(this.replay,squads);
            return new GameMaster(squads,playerSquad,this.createModeSettings());
         }
         if(this.iStart)
         {
            sq1 = new Squad("Squad 1",0,PlayerType.LOCAL_PLAYER,1);
            initSquad(sq1,Application.userProfile,posLists);
            sq2 = new Squad("Squad 2",1,PlayerType.REMOTE_PLAYER,2);
            initSquad(sq2,opponentProfile,posLists);
            playerSquad = 0;
         }
         else
         {
            sq1 = new Squad("Squad 1",0,PlayerType.REMOTE_PLAYER,1);
            initSquad(sq1,opponentProfile,posLists);
            sq2 = new Squad("Squad 2",1,PlayerType.LOCAL_PLAYER,2);
            initSquad(sq2,Application.userProfile,posLists);
            playerSquad = 1;
         }
         this.replay.setSquads(sq1,sq2);
         return new GameMaster([sq1,sq2],playerSquad,this.createModeSettings());
      }
      
      private function createModeSettings() : ModeSettings
      {
         var ms:ModeSettings = new ModeSettings();
         ms.online = true;
         ms.spawnWeaponCrates = true;
         ms.spawnCraftCrateOnDamageGround = false;
         ms.spawnCraftCrateFallTop = true;
         ms.spawnMedkits = true;
         ms.spawnStars = false;
         ms.timer = TimerSettings.DEFAULT_TIMER_SETTINGS;
         ms.achievements = Social.settings.isAchievAvailable() && this.isWager;
         ms.resultDeterminator = PvpResultDeterminator.INSTANCE;
         return ms;
      }
   }
}

