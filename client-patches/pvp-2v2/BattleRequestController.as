package ru.pragmatix.wormix.pvp.controllers
{
   import flash.events.Event;
   import ru.pragmatix.flox.core.FN;
   import ru.pragmatix.flox.core.SocketEvents;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.gui.events.UIEvent;
   import ru.pragmatix.flox.net.IncommingMessageEvent;
   import ru.pragmatix.flox.utils.MessageUtils;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.events.InterfaceEvent;
   import ru.pragmatix.wormix.game.mission.ScenarioData;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.gui.menu.SelectMapMenu;
   import ru.pragmatix.wormix.gui.window.Message;
   import ru.pragmatix.wormix.gui.window.WaitForBattleDialog;
   import ru.pragmatix.wormix.loader.BattleStartDataLoader;
   import ru.pragmatix.wormix.messages.client.StartBattle;
   import ru.pragmatix.wormix.messages.server.StartBattleResult;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.net.MainConnection;
   import ru.pragmatix.wormix.net.Server;
   import ru.pragmatix.wormix.pvp.messages.client.BattleRequest;
   import ru.pragmatix.wormix.pvp.messages.client.CancelBattle;
   import ru.pragmatix.wormix.pvp.messages.server.BattleRequestResult;
   import ru.pragmatix.wormix.pvp.messages.wager.WagerBattleRequest;
   import ru.pragmatix.wormix.pvp.messages.wager.WagerBattleRequestError;
   import ru.pragmatix.wormix.pvp.messages.wager.WagerBattleRequestResult;
   import ru.pragmatix.wormix.pvp.messages.wager.WagerCancelBattle;
   import ru.pragmatix.wormix.settings.PvpWager;
   import ru.pragmatix.wormix.settings.ScenarioRecord;
   import ru.pragmatix.wormix.social.Social;
   
   public class BattleRequestController
   {
      
      private var selectMapMenuRemote:SelectMapMenu;
      
      private var selectMapMenuLocal:SelectMapMenu;
      
      private var dialog:WaitForBattleDialog;
      
      private var info:BattleStartInfo;
      
      private var callById:Boolean = false;
      
      private var dataLoader:BattleStartDataLoader;
      
      private var requestParams:Object;
      
      // Patch (2v2): the wager request is for a 2v2 battle.
      private var teamMode:Boolean = false;
      
      public function BattleRequestController()
      {
         super();
      }
      
      public function missionRequest(profile:UserProfile) : void
      {
         if(Application.isUserBusy())
         {
            return;
         }
         Application.setUserBusy(true);
         Message.showBattleLoadingMsg(Messages.BATTLE_MISSION_LOAD);
         this.info = new BattleStartInfo(BattleType.MISSION);
         this.info.enemyProfile = profile;
         this.info.myTurn = true;
         this.info.missionId = 0;
         this.request(this.loadMissionData,false);
      }
      
      private function loadMissionData() : void
      {
         this.dataLoader = new BattleStartDataLoader();
         this.dataLoader.load(this.info,this.onDataForMissionLoad,this.onDataForMissionFail);
      }
      
      private function onDataForMissionFail() : void
      {
         this.dataLoader = null;
         this.reset(true);
         Message.hideBattleLoadingMsg();
      }
      
      private function onDataForMissionLoad() : void
      {
         this.dataLoader = null;
         SocketEvents.single(MainConnection.INSTANCE,StartBattleResult,this.onStartMissionResult);
         var startBattle:StartBattle = new StartBattle();
         startBattle.missionId = this.info.missionId;
         MainConnection.sendMessage(startBattle);
      }
      
      private function onStartMissionResult(e:IncommingMessageEvent) : void
      {
         var msg:StartBattleResult = StartBattleResult(e.message);
         this.info.battleId = msg.battleId;
         this.populateReagents(msg.reagentsForBattle);
         Singletons.btc.startBattle(this.info);
         this.reset(false);
      }
      
      public function scenarioRequest(rec:ScenarioRecord) : void
      {
         if(Application.isUserBusy())
         {
            return;
         }
         Application.setUserBusy(true);
         Message.showBattleLoadingMsg(Messages.BATTLE_MISSION_LOAD);
         this.info = new BattleStartInfo(BattleType.SCENARIO);
         var dataClass:Class = rec.getClass();
         var data:ScenarioData = new dataClass();
         this.info.myTurn = true;
         this.info.scenario = data;
         this.info.mapId = data.map.id;
         this.info.noMapGround = data.noMapGround;
         this.info.missionId = rec.getId();
         this.request(this.loadScenarioData,false);
      }
      
      private function loadScenarioData() : void
      {
         this.dataLoader = new BattleStartDataLoader();
         this.dataLoader.load(this.info,this.onScenarioDataLoad,this.onScenarioDataFail);
      }
      
      private function onScenarioDataLoad() : void
      {
         this.dataLoader = null;
         SocketEvents.single(MainConnection.INSTANCE,StartBattleResult,this.onStartScenarioResult);
         var startBattle:StartBattle = new StartBattle();
         startBattle.missionId = this.info.missionId;
         MainConnection.sendMessage(startBattle);
      }
      
      private function onStartScenarioResult(e:IncommingMessageEvent) : void
      {
         var msg:StartBattleResult = StartBattleResult(e.message);
         this.info.battleId = msg.battleId;
         this.populateReagents(msg.reagentsForBattle);
         Singletons.btc.startBattle(this.info);
         this.reset(false);
         Message.hideBattleLoadingMsg();
      }
      
      private function onScenarioDataFail() : void
      {
         this.dataLoader = null;
         this.reset(true);
         Message.hideBattleLoadingMsg();
      }
      
      public function wagerRequest(wager:PvpWager, team:Boolean = false) : void
      {
         this.teamMode = team;
         Application.setUserBusy(true);
         this.info = new BattleStartInfo(BattleType.PVP_WAGER);
         this.info.wager = wager;
         this.request(this.sendWagerBattleRequest,true);
      }
      
      private function sendWagerBattleRequest() : void
      {
         MainConnection.addMessageListener(WagerBattleRequestResult,this.onWagerBattleRequestResult);
         MainConnection.addMessageListener(WagerBattleRequestError,this.onWagerBattleRequestResult);
         var req:WagerBattleRequest = new WagerBattleRequest(this.info.pvpServer.getHost(),this.info.pvpServer.getAvailablePorts()[0],this.info.mapId,this.info.wager.ingamePrice);
         req.mode = this.teamMode ? 1 : 0;
         MainConnection.sendMessage(req);
      }
      
      private function clearWagerEvents() : void
      {
         MainConnection.removeMessageListener(WagerBattleRequestResult,this.onWagerBattleRequestResult);
         MainConnection.removeMessageListener(WagerBattleRequestError,this.onWagerBattleRequestResult);
      }
      
      private function onWagerBattleRequestResult(e:IncommingMessageEvent) : void
      {
         var result:WagerBattleRequestResult = null;
         var error:WagerBattleRequestError = null;
         var msgList:Object = null;
         this.clearWagerEvents();
         this.hideDialog();
         if(e.message is WagerBattleRequestResult)
         {
            result = WagerBattleRequestResult(e.message);
            this.info.enemyProfile = new UserProfile();
            this.info.myTurn = result.firstPlayerId == Application.userProfile.getId();
            this.info.pvpServer = new Server(result.host,[result.port]);
            this.info.mapId = result.mapId;
            this.info.seed = result.seed;
            this.info.battleId = result.battleId;
            this.populateReagents(result.reagentsForBattle);
            Application.userProfile.initFromStructure(result.userProfile);
            Application.userProfile.addIngameMoney(this.info.wager.ingamePrice);
            if(result.teamProfiles.length > 2)
            {
               this.initTeams(result);
            }
            this.loadDataAndStart(result.enemyProfile);
         }
         else
         {
            error = WagerBattleRequestError(e.message);
            msgList = {};
            msgList[WagerBattleRequestError.NOT_ENOUGH_MONEY] = Messages.ERROR_HACKING_ATTEMPT;
            msgList[WagerBattleRequestError.ERROR] = Messages.ERROR_SERVER;
            MessageUtils.showErrorMsgWithSound(Messages.BATTLE_REQUEST_TITLE,error.code,msgList);
            this.reset(true);
         }
      }
      
      // Patch (2v2): every player in the server's turn order. This player is Application.userProfile and the enemy
      // the server named is info.enemyProfile (filled in by loadDataAndStart), so the rest of the game sees the
      // same objects.
      private function initTeams(result:WagerBattleRequestResult) : void
      {
         var s:UserProfileStructure = null;
         var p:UserProfile = null;
         this.info.teamProfiles = [];
         this.info.teamIds = result.teamIds;
         for each(s in result.teamProfiles)
         {
            if(s.id == Application.userProfile.getId())
            {
               p = Application.userProfile;
            }
            else if(s.id == result.enemyProfile.id)
            {
               p = this.info.enemyProfile;
            }
            else
            {
               p = new UserProfile();
               p.initFromStructure(s);
            }
            this.info.teamProfiles.push(p);
         }
      }
      
      public function friendBattleRequest(profile:UserProfile) : void
      {
         Application.setUserBusy(true);
         this.info = new BattleStartInfo(BattleType.PVP_WITH_FRIEND);
         this.info.enemyProfile = profile;
         this.info.myTurn = true;
         this.callById = false;
         this.selectMapAndRequest(this.sendFriendBattleRequest,true);
      }
      
      public function friendLocalBattleRequest(profile:UserProfile) : void
      {
         Application.setUserBusy(true);
         this.info = new BattleStartInfo(BattleType.HOT_SEAT);
         this.info.enemyProfile = profile;
         this.info.myTurn = true;
         this.callById = false;
         this.showSelectMapMenuLocal();
      }
      
      public function friendBattleRequestByID(profile:UserProfile) : void
      {
         Application.setUserBusy(true);
         this.info = new BattleStartInfo(BattleType.PVP_WITH_FRIEND);
         this.info.enemyProfile = profile;
         this.info.myTurn = true;
         this.callById = true;
         this.selectMapAndRequest(this.sendFriendBattleRequest,true);
      }
      
      private function sendFriendBattleRequest() : void
      {
         SocketEvents.single(MainConnection.INSTANCE,BattleRequestResult,this.onFriendBattleRequestResult);
         MainConnection.sendMessage(new BattleRequest(this.info.pvpServer.getHost(),this.info.pvpServer.getAvailablePorts()[0],this.info.mapId,this.info.enemyProfile.getSocialId(),this.info.seed));
      }
      
      private function onFriendBattleRequestResult(e:IncommingMessageEvent) : void
      {
         var postAvailable:Boolean = false;
         var msg:String = null;
         var buttons:uint = 0;
         this.hideDialog();
         var result:BattleRequestResult = BattleRequestResult(e.message);
         switch(result.result)
         {
            case BattleRequestResult.AGREE:
               this.info.battleId = result.battleId;
               Application.userProfile.initFromStructure(result.myProfile);
               this.loadDataAndStart(result.enemyProfile);
               break;
            default:
               if(result.result == BattleRequestResult.ENEMY_ID_NOT_FOUND)
               {
                  MessageBox.showMsg(Messages.BATTLE_REQUEST_TITLE,Messages.ERROR_REQUEST_FRIEND_TO_BATTLE,MessageBox.OK_BTN);
               }
               else
               {
                  postAvailable = Social.postController.isFriendNotificationAvailable();
                  switch(result.result)
                  {
                     case BattleRequestResult.DISAGREE:
                        msg = Messages.BATTLE_REQUEST_DECLINE;
                        break;
                     case BattleRequestResult.BUSY:
                        msg = Messages.BATTLE_REQUEST_BUSY;
                        break;
                     case BattleRequestResult.NOT_ONLINE:
                        msg = Messages.BATTLE_REQUEST_NOT_ONLINE;
                  }
                  if(postAvailable)
                  {
                     msg += "<br>" + Messages.VIRAL_QUESTION_CHALLENGE_POST;
                  }
                  buttons = postAvailable ? uint(MessageBox.OK_CANCEL_BTN) : uint(MessageBox.OK_BTN);
                  MessageBox.showMsg(Messages.BATTLE_REQUEST_TITLE,msg,buttons,this.callToBattlePostClick,this.onCancelClick);
               }
         }
      }
      
      private function onCancelClick() : void
      {
         this.reset(true);
      }
      
      private function callToBattlePostClick() : void
      {
         Social.postController.callToPvp(Application.userProfile,this.info.enemyProfile);
         this.reset(true);
      }
      
      private function selectMapAndRequest(sendMsgFn:Function, showWaitDialog:Boolean) : void
      {
         this.requestParams = {
            "sendMsgFn":sendMsgFn,
            "showWaitDialog":showWaitDialog
         };
         this.showSelectMapMenuRemote();
      }
      
      private function populateReagents(reagentsForBattle:Vector.<int>) : void
      {
         var id:int = 0;
         if(Boolean(reagentsForBattle))
         {
            for each(id in reagentsForBattle)
            {
               if(id >= 0)
               {
                  this.info.craftItemIdInCrate.push(id);
               }
            }
         }
      }
      
      private function request(sendMsgFn:Function, showWaitDialog:Boolean) : void
      {
         Application.setUserBusy(true);
         if(!this.info.mapId)
         {
            this.info.mapId = Singletons.mapController.getRandomMapId(this.info.battleType);
         }
         if(this.info.battleType.isPvp())
         {
            this.info.pvpServer = Server.PVP;
         }
         this.info.seed = Math.random() * 1000;
         sendMsgFn();
         if(showWaitDialog)
         {
            this.dialog = new WaitForBattleDialog(this.info.battleType,this.callById);
            this.dialog.addEventListener(InterfaceEvent.CANCEL_BATTLE_REQUEST,this.cancelRequest);
            if(Boolean(this.info.enemyProfile))
            {
               this.dialog.setProfile(this.info.enemyProfile);
            }
            this.dialog.show();
         }
      }
      
      private function cancelRequest(e:Event) : void
      {
         switch(this.info.battleType)
         {
            case BattleType.PVP_WITH_FRIEND:
               MainConnection.sendMessage(new CancelBattle(Application.userProfile.getId(),this.info.enemyProfile.getId()));
               break;
            case BattleType.PVP_WAGER:
               this.clearWagerEvents();
               MainConnection.sendMessage(new WagerCancelBattle());
         }
         Message.WAITING_DATA_MSG.show();
         FN.delay(function():void
         {
            reset(true);
            Message.WAITING_DATA_MSG.hide();
         },this,500);
      }
      
      private function hideDialog() : void
      {
         if(Boolean(this.dialog))
         {
            this.dialog.hide();
            this.dialog.killDelayTimer();
         }
      }
      
      private function loadDataAndStart(updatedEnemyProfile:UserProfileStructure) : void
      {
         this.info.enemyProfile.initFromStructure(updatedEnemyProfile);
         Singletons.btc.loadDataAndStart(this.info);
         Application.mainMenu.mainMenu.hideRatingPanel();
      }
      
      private function startBattle() : void
      {
         Singletons.battleController.loadDataAndStart(this.info);
         this.reset(false);
      }
      
      private function reset(cancel:Boolean) : void
      {
         this.info = null;
         if(Boolean(this.dialog))
         {
            this.dialog.removeEventListener(InterfaceEvent.CANCEL_BATTLE_REQUEST,this.cancelRequest);
         }
         this.dialog = null;
         if(cancel)
         {
            Application.setUserBusy(false);
         }
      }
      
      private function showSelectMapMenuRemote() : void
      {
         if(!this.selectMapMenuRemote)
         {
            this.selectMapMenuRemote = new SelectMapMenu();
            this.selectMapMenuRemote.addEventListener(UIEvent.MENU_ITEM_SELECT,this.onMapSelectRemote);
            this.selectMapMenuRemote.addEventListener(UIEvent.WINDOW_CANCEL,this.onMapSelectClose);
         }
         this.selectMapMenuRemote.show();
      }
      
      private function showSelectMapMenuLocal() : void
      {
         if(!this.selectMapMenuLocal)
         {
            this.selectMapMenuLocal = new SelectMapMenu();
            this.selectMapMenuLocal.addEventListener(UIEvent.MENU_ITEM_SELECT,this.onMapSelectLocal);
            this.selectMapMenuLocal.addEventListener(UIEvent.WINDOW_CANCEL,this.onMapSelectClose);
         }
         this.selectMapMenuLocal.show();
      }
      
      private function onMapSelectClose(e:Event) : void
      {
         if(!SelectMapMenu(e.target).getSelected())
         {
            Application.setUserBusy(false);
            this.info = null;
         }
      }
      
      private function onMapSelectRemote(event:Event) : void
      {
         this.info.mapId = Boolean(this.selectMapMenuRemote.selectedMap) ? this.selectMapMenuRemote.selectedMap.id : 0;
         this.request(this.requestParams.sendMsgFn,this.requestParams.showWaitDialog);
      }
      
      private function onMapSelectLocal(event:Event) : void
      {
         this.info.mapId = Boolean(this.selectMapMenuLocal.selectedMap) ? this.selectMapMenuLocal.selectedMap.id : 0;
         this.requestForLocalBattle();
      }
      
      private function requestForLocalBattle() : void
      {
         Application.setUserBusy(true);
         if(!this.info.mapId)
         {
            this.info.mapId = Singletons.mapController.getRandomMapId(this.info.battleType);
         }
         if(this.info.battleType.isPvp())
         {
            this.info.pvpServer = Server.PVP;
         }
         this.info.seed = Math.random() * 1000;
         this.startBattle();
      }
   }
}

