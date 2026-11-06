package ru.pragmatix.wormix.controller
{
   import config.LevelUpRewardConfig;
   import config.ScenarioConfig;
   import flash.events.Event;
   import flash.events.EventDispatcher;
   import flash.utils.ByteArray;
   import ru.pragmatix.flox.core.Cookie;
   import ru.pragmatix.flox.core.Events;
   import ru.pragmatix.flox.core.FN;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.gui.controls.menu.UIMenu;
   import ru.pragmatix.flox.gui.controls.message.PopUpMessage;
   import ru.pragmatix.flox.gui.controls.window.UIWindow;
   import ru.pragmatix.flox.gui.core.UIMessages;
   import ru.pragmatix.flox.gui.events.UIEvent;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.model.MoneyType;
   import ru.pragmatix.flox.net.IncommingMessageEvent;
   import ru.pragmatix.flox.secure.LowFpsEvent;
   import ru.pragmatix.flox.secure.SpeedhackDetectedEvent;
   import ru.pragmatix.flox.social.post.IPostToStatus;
   import ru.pragmatix.flox.sound.SoundController;
   import ru.pragmatix.wormix.Scene;
   import ru.pragmatix.wormix.controls.ControlSets;
   import ru.pragmatix.wormix.events.GameEvent;
   import ru.pragmatix.wormix.events.NextTurnEvent;
   import ru.pragmatix.wormix.game.GameMaster;
   import ru.pragmatix.wormix.game.rules.HotSeatGameRules;
   import ru.pragmatix.wormix.game.rules.IGameRules;
   import ru.pragmatix.wormix.game.rules.MissionGameRules;
   import ru.pragmatix.wormix.game.rules.PvpGameRules;
   import ru.pragmatix.wormix.game.rules.ScenarioGameRules;
   import ru.pragmatix.wormix.game.squad.Squad;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.gui.game.Taunts;
   import ru.pragmatix.wormix.gui.house.achiev.Stats;
   import ru.pragmatix.wormix.gui.window.Message;
   import ru.pragmatix.wormix.gui.window.gameover.AbstractGameOverDialog;
   import ru.pragmatix.wormix.gui.window.gameover.FriendGameOverMessage;
   import ru.pragmatix.wormix.gui.window.gameover.GameOverMessage;
   import ru.pragmatix.wormix.gui.window.gameover.PvpGameOverMessage;
   import ru.pragmatix.wormix.gui.window.gameover.boss.BossGameOverDrawMessage;
   import ru.pragmatix.wormix.gui.window.gameover.boss.BossGameOverLoseMessage;
   import ru.pragmatix.wormix.gui.window.gameover.boss.BossGameOverWinMessage;
   import ru.pragmatix.wormix.gui.window.levelup.LevelUpDialog;
   import ru.pragmatix.wormix.loader.BattleStartDataLoader;
   import ru.pragmatix.wormix.messages.client.EndBattle;
   import ru.pragmatix.wormix.messages.client.PvpChatMessage;
   import ru.pragmatix.wormix.model.Sounds;
   import ru.pragmatix.wormix.model.battle.BattleResult;
   import ru.pragmatix.wormix.model.battle.BattleResultInfo;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.battle.EndBattleStatus;
   import ru.pragmatix.wormix.model.battle.MissionType;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.items.ItemsStore;
   import ru.pragmatix.wormix.model.items.backpack.BackpackItem;
   import ru.pragmatix.wormix.model.items.backpack.WeaponBackpack;
   import ru.pragmatix.wormix.model.rewards.RewardPack;
   import ru.pragmatix.wormix.model.rewards.type.RewardHat;
   import ru.pragmatix.wormix.model.rewards.type.RewardReagent;
   import ru.pragmatix.wormix.model.rewards.type.RewardWeapon;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   import ru.pragmatix.wormix.net.MainConnection;
   import ru.pragmatix.wormix.objects.HatDropped;
   import ru.pragmatix.wormix.objects.Worm;
   import ru.pragmatix.wormix.pvp.action.Action;
   import ru.pragmatix.wormix.pvp.action.ActionEvent;
   import ru.pragmatix.wormix.pvp.action.Actions;
   import ru.pragmatix.wormix.settings.BattleAwardSettings;
   import ru.pragmatix.wormix.settings.LevelUpRewardPack;
   import ru.pragmatix.wormix.settings.ScenarioRecord;
   import ru.pragmatix.wormix.social.Social;
   import ru.pragmatix.wormix.utils.CraftUtil;
   import ru.pragmatix.wormix.utils.EventMaster;
   import ru.pragmatix.wormix.weapons.Arsenal;
   
   public class BattleController extends EventDispatcher
   {
      
      private static const BLOOD_COOKIE_NAME:String = "blood";
      
      private static const BRAINS_COOKIE_NAME:String = "brains";
      
      private const MAX_FINITE_WEAPONS_ALLOWED:uint = 10;
      
      private var log:Logger = new Logger(BattleController);
      
      public var scene:Scene;
      
      public var gameMaster:GameMaster;
      
      public var battleStats:BattleRecords;
      
      private var startInfo:BattleStartInfo;
      
      private var resultInfo:BattleResultInfo;
      
      private var dataLoader:BattleStartDataLoader;
      
      private var inBattle:Boolean;
      
      private var awardHat:HatRecord = null;
      
      public var restrictedAwardWeapons:Array;
      
      private var showBlood:Boolean;
      
      private var showBrains:Boolean;
      
      public function BattleController()
      {
         super();
         this.showBlood = Cookie.getBooleanValue(BLOOD_COOKIE_NAME,false);
         this.showBrains = Cookie.getBooleanValue(BRAINS_COOKIE_NAME,false);
      }
      
      public function loadDataAndStart(info:BattleStartInfo) : void
      {
         Application.setUserBusy(true);
         this.startInfo = info;
         Message.showBattleLoadingMsg(Messages.BATTLE_INIT);
         this.dataLoader = new BattleStartDataLoader();
         this.dataLoader.load(info,FN.createDelegate(this.startBattle,this,[info]),this.onDataLoadFail);
      }
      
      public function startBattle(info:BattleStartInfo) : void
      {
         var gameRules:IGameRules = null;
         var item:BackpackItem = null;
         this.startInfo = info;
         this.resultInfo = null;
         if(info.battleType.isPvp())
         {
            gameRules = new PvpGameRules(info);
            Singletons.pvpController.enter(info,FN.createDelegate(this.enterBattle,this,[gameRules]),this.onPvpLoginFail);
         }
         else
         {
            switch(info.battleType)
            {
               case BattleType.HOT_SEAT:
                  this.enterBattle(new HotSeatGameRules(info));
                  break;
               case BattleType.MISSION:
                  Singletons.battleAvailabilityController.decBattlesCount();
                  this.enterBattle(new MissionGameRules(info));
                  break;
               case BattleType.SCENARIO:
                  Singletons.battleAvailabilityController.decBattlesCount();
                  this.restrictedAwardWeapons = [];
                  for each(item in Application.userProfile.getWeaponBackpack().getDepositSortedArray())
                  {
                     if(!item.isInfinite() && item.getCount() > this.MAX_FINITE_WEAPONS_ALLOWED)
                     {
                        this.restrictedAwardWeapons.push(item.getId());
                     }
                  }
                  this.enterBattle(new ScenarioGameRules(info));
            }
         }
      }
      
      private function onPvpLoginFail(error:String) : void
      {
         this.onInitFail();
         this.dataLoader = null;
         MessageBox.showError(UIMessages.TITLE_ERROR,error);
      }
      
      private function onDataLoadFail() : void
      {
         this.onInitFail();
      }
      
      private function onInitFail() : void
      {
         MainConnection.sendMessage(this.createEndBattleMsgByInfo(BattleResult.DRAW_GAME));
         this.reset();
         Application.setUserBusy(false);
      }
      
      private function enterBattle(gameRules:IGameRules) : void
      {
         this.inBattle = true;
         this.scene = this.startInfo.scene;
         this.startInfo.scene = null;
         Application.instance.openScene(this.scene);
         if(!Singletons.context.getAppMode().isLocal())
         {
            Singletons.context.getPreloaderParams().preloaderMainClass.hideBanner();
         }
         this.scene.init(this.startInfo.battleType);
         this.scene.gameInterface.bagMenu.addEventListener(UIEvent.MENU_ITEM_SELECT,this.onWeaponSelect);
         this.scene.gameInterface.bagMenu.addEventListener(UIEvent.WINDOW_SHOW,this.onBagOpen);
         this.scene.gameInterface.bagMenu.addEventListener(UIEvent.WINDOW_CLOSE,this.onBagCancel);
         this.scene.gameInterface.hatsPanel.addEventListener(UIEvent.MENU_ITEM_SELECT,this.onHatSelected);
         if(this.startInfo.battleType.isPvp())
         {
            Singletons.pvpController.addChatMessageListener(this.onChatMessage);
         }
         this.battleStats = new BattleRecords();
         this.gameMaster = gameRules.createGM();
         this.scene.gameInterface.initTeams(this.gameMaster.getSquads(),this.startInfo.battleType);
         EventMaster.once(this.scene,GameEvent.SCENE_STABLE,function():void
         {
            scene.dispatchEvent(new NextTurnEvent(0));
         });
         Singletons.speedhackGuard.addEventListener(SpeedhackDetectedEvent.SPEEDHACK_DETECTED,this.onSpeedhackDetected);
         Singletons.speedhackGuard.run();
         Singletons.pvpVictory = false;
         SoundController.play(Sounds.BATTLE_START);
      }
      
      public function endBattle(result:BattleResult, status:EndBattleStatus, signature:ByteArray = null) : void
      {
         if(!this.inBattle)
         {
            return;
         }
         if(!signature)
         {
            signature = new ByteArray();
         }
         this.inBattle = false;
         Singletons.speedhackGuard.stop();
         Singletons.speedhackGuard.removeEventListener(SpeedhackDetectedEvent.SPEEDHACK_DETECTED,this.onSpeedhackDetected);
         if(Boolean(this.startInfo.scenario))
         {
            this.startInfo.scenario.clear();
            this.startInfo.scenario = null;
         }
         dispatchEvent(new Event(GameEvent.EXIT_BATTLE));
         this.resultInfo = new BattleResultInfo();
         this.resultInfo.result = result;
         this.resultInfo.signature = signature;
         this.resultInfo.status = status;
         this.resultInfo.missionType = MissionType.getMissionType(this.startInfo.enemyProfile);
         this.resultInfo.bonusExp = this.gameMaster.getBonusExperience();
         if(this.startInfo.battleType == BattleType.SCENARIO)
         {
            if(this.resultInfo.result == BattleResult.WINNER)
            {
               this.resultInfo.craftItemIdInCrate = this.startInfo.craftItemIdInCrate;
            }
         }
         else
         {
            this.resultInfo.craftItemIdInCrate = this.gameMaster.getCraftSpawnerController().getCalculatedReagents();
         }
         if(status == EndBattleStatus.PVP_CONNECTION_LOSED)
         {
            MessageBox.showError(UIMessages.TITLE_ERROR,Messages.PVP_SERVER_CONNECTION_CLOSED);
         }
         if(status != EndBattleStatus.MAIN_CONNECTION_LOSED)
         {
            this.sendEndBattleMsg(this.resultInfo);
            this.calculateAwards(this.startInfo,this.resultInfo);
         }
         if(this.startInfo.battleType.isPvp())
         {
            Singletons.pvpController.removeChatMessageListener(this.onChatMessage);
            if(status != EndBattleStatus.PVP_CONNECTION_LOSED && status != EndBattleStatus.MAIN_CONNECTION_LOSED)
            {
               FN.delay(Singletons.pvpController.closeConnection,Singletons.pvpController,1000);
            }
            else if(status == EndBattleStatus.MAIN_CONNECTION_LOSED)
            {
               Singletons.pvpController.closeConnection();
            }
         }
         this.battleStats.incrementAfterBattle(this.startInfo,this.resultInfo);
         Singletons.achievController.increase(this.battleStats.records.ids,this.battleStats.records.values);
         this.reset();
      }
      
      public function cancelBattle(status:EndBattleStatus) : void
      {
         this.endBattle(BattleResult.DRAW_GAME,status);
      }
      
      private function sendEndBattleMsg(resultInfo:BattleResultInfo) : void
      {
         var endBattleMsg:EndBattle = this.createEndBattleMsgByInfo(resultInfo.result);
         endBattleMsg.signature = resultInfo.signature;
         endBattleMsg.banType = this.gameMaster.getBan();
         if(this.startInfo.battleType.isPvp())
         {
            Singletons.pvpController.sendEndBattleCommand(resultInfo.status);
         }
         endBattleMsg.expBonus = resultInfo.bonusExp;
         if(this.startInfo.battleType != BattleType.HOT_SEAT)
         {
            endBattleMsg.items = this.gameMaster.getShots();
         }
         MainConnection.sendMessage(endBattleMsg);
      }
      
      private function createEndBattleMsgByInfo(result:BattleResult) : EndBattle
      {
         var endBattleMsg:EndBattle = new EndBattle(result.value + this.startInfo.battleId);
         endBattleMsg.battleId = this.startInfo.battleId;
         endBattleMsg.type = this.startInfo.battleType.isPvp() ? EndBattle.TYPE_PVP_GAME : MissionType.getMissionType(this.startInfo.enemyProfile).value;
         if(Boolean(this.startInfo.missionId))
         {
            endBattleMsg.missionId = this.startInfo.missionId;
         }
         if(this.resultInfo != null)
         {
            endBattleMsg.collectedReagents = this.resultInfo.craftItemIdInCrate;
         }
         return endBattleMsg;
      }
      
      private function chekHashFunction(battleId:int) : int
      {
         var c2:int = 606399277;
         var key:int = battleId;
         key = key ^ 0x3D ^ key >>> 16;
         key += key << 3;
         key ^= key >>> 4;
         key *= c2;
         return key ^ key >>> 15;
      }
      
      private function reset() : void
      {
         if(Boolean(this.scene) && Boolean(this.scene.gameInterface) && Boolean(this.scene.gameInterface.bagMenu))
         {
            this.scene.gameInterface.bagMenu.removeEventListener(UIEvent.MENU_ITEM_SELECT,this.onWeaponSelect);
            this.scene.gameInterface.bagMenu.removeEventListener(UIEvent.WINDOW_SHOW,this.onBagOpen);
            this.scene.gameInterface.bagMenu.removeEventListener(UIEvent.WINDOW_CLOSE,this.onBagCancel);
            this.scene.gameInterface.hatsPanel.removeEventListener(UIEvent.MENU_ITEM_SELECT,this.onHatSelected);
         }
         if(Boolean(this.gameMaster))
         {
            this.gameMaster.deactivate();
            this.gameMaster = null;
         }
         if(Boolean(this.battleStats))
         {
            this.battleStats = null;
         }
         PopUpMessage.reset();
         Arsenal.resetWeapons();
         Taunts.deactivate();
         Application.userProfile.getWeaponBackpack().reset();
      }
      
      private function onGameOverDialogClose(e:Event) : void
      {
         if(this.awardHat != null)
         {
            Application.userProfile.setHat(this.awardHat);
            this.awardHat = null;
         }
         Application.mainMenu.updateProfileInfo(Application.userProfile,true);
         Application.instance.openMainMenu();
         var checkLevel:Boolean = true;
         var postToStatus:Boolean = IPostToStatus(e.target).getStatusPostSelected();
         if(this.startInfo.battleType != BattleType.HOT_SEAT && this.startInfo.battleType != BattleType.SCENARIO)
         {
            if(postToStatus)
            {
               Social.postController.postToStatus(this.startInfo,this.resultInfo,this.postNewLevel);
               checkLevel = false;
            }
            else if(e.target is AbstractGameOverDialog && (e.target as AbstractGameOverDialog).createNoteSelected)
            {
               Social.postController.postBattleDescription(this.startInfo,this.resultInfo,this.postNewLevel);
               checkLevel = false;
            }
            else if(this.startInfo.battleType == BattleType.PVP_WITH_FRIEND && this.resultInfo.result == BattleResult.WINNER)
            {
               Social.postController.winInBattleWithFriend(this.startInfo.enemyProfile,this.postNewLevel);
               checkLevel = false;
            }
         }
         else if(this.startInfo.battleType == BattleType.SCENARIO)
         {
            if(ScenarioConfig.isBoss(this.startInfo.missionId))
            {
               Social.postController.postGameOverBoss(postToStatus,ScenarioConfig.getRecord(this.startInfo.missionId),this.resultInfo.result);
            }
            else if(postToStatus && !ScenarioConfig.isBoss(this.startInfo.missionId))
            {
               Social.postController.postTutorialComplete(ScenarioConfig.getRecord(this.startInfo.missionId).getName());
            }
         }
         if(checkLevel)
         {
            this.postNewLevel();
         }
      }
      
      private function postNewLevel() : void
      {
         var levelUpDialog:LevelUpDialog = null;
         var rewardPack:LevelUpRewardPack = null;
         if(this.resultInfo.newLevel != 0)
         {
            levelUpDialog = new LevelUpDialog(this.resultInfo);
            Events.single(levelUpDialog,UIEvent.WINDOW_CLOSE,this.onLevelUpDialogClose);
            levelUpDialog.show();
            rewardPack = LevelUpRewardConfig.getRewardByLevel(Application.userProfile.getLevel());
            if(rewardPack != null)
            {
               this.addItems(rewardPack.getWeaponRewards(),rewardPack.getHatRewards());
            }
         }
         else
         {
            this.clearInitInfo();
         }
      }
      
      private function addItems(weapons:Array, hats:Array) : void
      {
         var weapon:RewardWeapon = null;
         var hat:RewardHat = null;
         var hatRec:HatRecord = null;
         var bag:WeaponBackpack = Application.userProfile.getWeaponBackpack();
         for each(weapon in weapons)
         {
            if(Boolean(this.restrictedAwardWeapons) && this.restrictedAwardWeapons.indexOf(weapon.getId()) == -1)
            {
               bag.putItem(ItemsStore.getWeaponById(weapon.getId()),false,weapon.getCount() + bag.getCountItem(weapon.getId()));
            }
         }
         this.awardHat = null;
         for each(hat in hats)
         {
            hatRec = ItemsStore.getHatById(hat.getId(),Application.userProfile.getCharType());
            Application.userProfile.getWardrobe().putItem(hatRec);
            this.awardHat = hatRec;
         }
      }
      
      private function onLevelUpDialogClose(e:Event) : void
      {
         var levelUpDialog:LevelUpDialog = e.target as LevelUpDialog;
         if(levelUpDialog.statusPostSelected)
         {
            Social.postController.newLevel(levelUpDialog.resultInfo.newLevel);
         }
         else if(levelUpDialog.shopSelected)
         {
            Singletons.shopController.showShopMenu();
         }
         this.clearInitInfo();
      }
      
      private function leaveTheGame() : void
      {
         Application.instance.openMainMenu();
         this.clearInitInfo();
      }
      
      private function clearInitInfo() : void
      {
         Social.postController.postBattleResultActivity(this.startInfo,this.resultInfo);
         this.startInfo.scene = null;
         this.startInfo = null;
         this.resultInfo = null;
         this.scene = null;
         Application.setUserBusy(false);
      }
      
      private function calculateAwards(startInfo:BattleStartInfo, resultInfo:BattleResultInfo) : void
      {
         var msg:UIWindow = null;
         var rewardPack:RewardPack = null;
         var record:ScenarioRecord = null;
         var idReg:int = 0;
         var id:int = 0;
         if(resultInfo.status == EndBattleStatus.STUCK || resultInfo.status == EndBattleStatus.DESYNC)
         {
            new MessageBox(Messages.BATTLE_CANCEL_TITLE,resultInfo.status.endBattleDescription,MessageBox.OK_BTN,this.leaveTheGame).show();
            return;
         }
         switch(startInfo.battleType)
         {
            case BattleType.PVP_WAGER:
               resultInfo.rating = BattleAwardSettings.getRating(resultInfo.result,startInfo.enemyProfile,startInfo.wager);
               resultInfo.money = BattleAwardSettings.getMoney(resultInfo.result,startInfo.wager);
               msg = new PvpGameOverMessage(startInfo,resultInfo);
               Application.userProfile.setRating(Application.userProfile.getRating() + resultInfo.rating);
               Application.userProfile.addIngameMoney(resultInfo.money);
               this.battleStats.increment(Stats.WAGERS_EARNED,resultInfo.money);
               break;
            case BattleType.PVP_WITH_FRIEND:
               msg = new FriendGameOverMessage(startInfo,resultInfo);
               break;
            case BattleType.MISSION:
               resultInfo.money = BattleAwardSettings.getMoneyAward(resultInfo);
               resultInfo.exp = BattleAwardSettings.getExpAward(resultInfo);
               Application.userProfile.addIngameMoney(resultInfo.money);
               this.checkLevelUp(resultInfo.exp + resultInfo.bonusExp);
               msg = GameOverMessage.createGameOverMessage(startInfo,resultInfo);
               this.battleStats.increment(Stats.MISSIONS_EARNED,resultInfo.money);
               break;
            case BattleType.SCENARIO:
               record = ScenarioConfig.getRecord(startInfo.missionId);
               if(resultInfo.result == BattleResult.WINNER)
               {
                  if(!ScenarioConfig.isBoss(record.getId()) || startInfo.missionId != ScenarioConfig.lastCompletedBoss() + 1)
                  {
                     rewardPack = record.getAward();
                  }
                  else
                  {
                     rewardPack = record.getAwardFirstWin() != null ? record.getAwardFirstWin() : record.getAward();
                  }
                  if(ScenarioConfig.isBoss(record.getId()) && Boolean(rewardPack))
                  {
                     if(Boolean(resultInfo.craftItemIdInCrate) && resultInfo.craftItemIdInCrate.length > 0)
                     {
                        for each(idReg in resultInfo.craftItemIdInCrate)
                        {
                           rewardPack.putReward(new RewardReagent(idReg,1));
                        }
                     }
                  }
               }
               if(rewardPack != null)
               {
                  Application.userProfile.addIngameMoney(rewardPack.getRewardMoney(MoneyType.INGAME));
                  Application.userProfile.addRealMoney(rewardPack.getRewardMoney(MoneyType.REAL));
                  this.checkLevelUp(rewardPack.getRewardExp() + resultInfo.bonusExp);
               }
               if(ScenarioConfig.isBoss(startInfo.missionId))
               {
                  if(resultInfo.result == BattleResult.WINNER)
                  {
                     msg = BossGameOverWinMessage.createBossGameOverMessage(rewardPack);
                     this.addItems(rewardPack.getWeaponRewards(),rewardPack.getHatRewards());
                  }
                  else if(resultInfo.result == BattleResult.LOSER)
                  {
                     msg = BossGameOverLoseMessage.createBossGameOverLoseMessage();
                  }
                  else if(resultInfo.result == BattleResult.DRAW_GAME)
                  {
                     msg = BossGameOverDrawMessage.createBossGameOverDrawMessage();
                  }
               }
               else
               {
                  msg = GameOverMessage.createGameOverMessage(startInfo,resultInfo);
               }
               break;
            case BattleType.HOT_SEAT:
               resultInfo.result = BattleResult.WINNER;
               msg = new FriendGameOverMessage(startInfo,resultInfo);
         }
         if(Boolean(resultInfo.craftItemIdInCrate))
         {
            for each(id in resultInfo.craftItemIdInCrate)
            {
               CraftUtil.addReagentInBackpack(id,1);
            }
         }
         Application.mainMenu.updateUserInfo();
         Events.single(msg,UIEvent.WINDOW_CLOSE,this.onGameOverDialogClose);
         if(Boolean(resultInfo.status.endBattleDescription))
         {
            MessageBox.showMsg(Messages.BATTLE_CANCEL_TITLE,resultInfo.status.endBattleDescription,MessageBox.OK_BTN,function():void
            {
               msg.show();
            });
         }
         else
         {
            msg.show();
         }
      }
      
      private function checkLevelUp(exp:int) : void
      {
         if(UserProfileHelper.addExperienceAndLevelUp(Application.userProfile,exp))
         {
            Application.userProfile.addIngameMoney(BattleAwardSettings.moneyForNewLevel);
            this.resultInfo.newLevel = Application.userProfile.getLevel();
            this.resultInfo.moneyForNewLevel = BattleAwardSettings.moneyForNewLevel;
            Singletons.battleAvailabilityController.resetBattleCount();
         }
      }
      
      private function onWeaponSelect(e:Event) : void
      {
         Application.curScene.dispatchEvent(new ActionEvent([new Action(Actions.UI_WEAPON_SELECT,UIMenu(e.target).getSelected().getId())]));
         ControlSets.WORM_CONTROL_SET.activate();
      }
      
      private function onHatSelected(e:Event) : void
      {
         var hatAction:Action = null;
         if(Boolean(this.gameMaster && this.gameMaster.activeWorm) && Boolean(!this.gameMaster.activeWorm.disposed) && this.gameMaster.activeWorm.stable)
         {
            hatAction = new Action(Actions.WORM_USE,HatDropped(UIMenu(e.target).getSelected()).getId());
            Application.curScene.dispatchEvent(new ActionEvent([hatAction]));
         }
      }
      
      private function onBagOpen(e:Event) : void
      {
         var aw:Worm = this.gameMaster.activeWorm;
         if(aw != null)
         {
            ControlSets.WORM_CONTROL_SET.deactivate();
            Application.curScene.dispatchEvent(new ActionEvent([new Action(Actions.UI_WEAPON_DEACTIVATE)]));
         }
      }
      
      private function onBagCancel(e:Event) : void
      {
         var aw:Worm = this.gameMaster.activeWorm;
         if(aw != null)
         {
            ControlSets.WORM_CONTROL_SET.activate();
            Application.curScene.dispatchEvent(new ActionEvent([new Action(Actions.UI_WEAPON_ACTIVATE)]));
         }
      }
      
      public function getSquad(profile:UserProfile) : Squad
      {
         return Boolean(this.gameMaster) ? this.gameMaster.getSquadByProfile(profile) : null;
      }
      
      private function onChatMessage(e:IncommingMessageEvent) : void
      {
         var chatMsg:PvpChatMessage = PvpChatMessage(e.message);
         var from:UserProfile = this.startInfo.enemyProfile;
         var p:UserProfile = null;
         if(this.startInfo.isTeamBattle())
         {
            // Patch (2v2): show who wrote it.
            for each(p in this.startInfo.teamProfiles)
            {
               if(p.getId() == chatMsg.id)
               {
                  from = p;
               }
            }
         }
         this.addChatMessage(chatMsg.message,from);
      }
      
      public function addChatMessage(msg:String, profile:UserProfile) : void
      {
         if(Boolean(this.scene) && Boolean(this.scene.gameInterface))
         {
            this.scene.gameInterface.chat.addMessage(msg,profile);
         }
      }
      
      private function onSpeedhackDetected(e:SpeedhackDetectedEvent) : void
      {
         this.endBattle(BattleResult.LOSER,EndBattleStatus.SPEED_HACK_CHEATER);
      }
      
      private function onLowFps(e:LowFpsEvent) : void
      {
         this.endBattle(BattleResult.LOSER,EndBattleStatus.FPS_CHEATER);
      }
      
      public function isInBattle() : Boolean
      {
         return this.inBattle;
      }
      
      public function getShowBlood() : Boolean
      {
         return this.showBlood;
      }
      
      public function getShowBrains() : Boolean
      {
         return this.showBrains;
      }
      
      public function toggleShowBlood() : Boolean
      {
         this.showBlood = !this.showBlood;
         Cookie.setValue(BLOOD_COOKIE_NAME,this.showBlood);
         return this.showBlood;
      }
      
      public function toggleShowBrains() : Boolean
      {
         this.showBrains = !this.showBrains;
         Cookie.setValue(BRAINS_COOKIE_NAME,this.showBrains);
         return this.showBrains;
      }
      
      public function getOpponent() : UserProfile
      {
         return this.startInfo.enemyProfile;
      }
      
      public function getBattleInfo() : BattleStartInfo
      {
         return this.startInfo;
      }
   }
}

