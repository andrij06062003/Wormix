package ru.pragmatix.wormix.pvp.controllers
{
   import flash.events.Event;
   import ru.pragmatix.flox.core.Strings;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.gui.events.UIEvent;
   import ru.pragmatix.flox.model.MoneyType;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   import ru.pragmatix.wormix.settings.PvpWager;
   import ru.pragmatix.flox.gui.controls.UIMessage;
   import ru.pragmatix.flox.loader.ClassLoader;
   import ru.pragmatix.flox.model.Language;
   import ru.pragmatix.flox.net.IncommingMessageEvent;
   import ru.pragmatix.flox.sound.SoundController;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.events.InterfaceEvent;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.gui.window.BattleOfferDialog;
   import ru.pragmatix.wormix.gui.window.Message;
   import ru.pragmatix.wormix.loader.UserSocialDataLoader;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.model.Sounds;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.net.MainConnection;
   import ru.pragmatix.wormix.net.Server;
   import ru.pragmatix.wormix.pvp.messages.client.BattleOfferResult;
   import ru.pragmatix.wormix.pvp.messages.client.CancelBattle;
   import ru.pragmatix.wormix.pvp.messages.server.BattleOffer;
   import ru.pragmatix.wormix.social.Social;
   
   public class BattleOfferController
   {
      
      private var battleStartInfo:BattleStartInfo;
      
      private var dialog:BattleOfferDialog;
      
      private var socialDataLoader:UserSocialDataLoader;
      
      private var message:UIMessage;
      
      // Patch (2v2 teams): the open team invite / call and its window.
      private var teamOffer:BattleOffer;
      
      private var teamBox:MessageBox;
      
      public function BattleOfferController()
      {
         super();
         MainConnection.addMessageListener(BattleOffer,this.onBattleOffer);
         if(Social.settings.getGUILanguage().equals(Language.EN))
         {
            this.message = new UIMessage(ClassLoader.getNewInstance("LoadingMessageWithIcon"),{
               "text":{
                  "text":Messages.ENEMY_WAITING_MSG,
                  "fontSize":18
               },
               "descr":Messages.BATTLE_ENEMY_DATA_LOAD
            });
         }
         else
         {
            this.message = new UIMessage(ClassLoader.getNewInstance("LoadingMessageWithIcon"),{
               "text":{"text":Messages.ENEMY_WAITING_MSG},
               "descr":Messages.BATTLE_ENEMY_DATA_LOAD
            });
         }
      }
      
      private function onBattleCancel(e:IncommingMessageEvent) : void
      {
         if(Boolean(this.dialog))
         {
            this.dialog.hide();
         }
         this.reset(true);
      }
      
      private function onBattleOffer(e:IncommingMessageEvent) : void
      {
         SoundController.play(Sounds.CHAT_MESSAGE);
         var battleOffer:BattleOffer = BattleOffer(e.message);
         var enemyProfile:UserProfileStructure = battleOffer.enemyProfile;
         if(battleOffer.kind != 0)
         {
            this.onTeamOffer(battleOffer);
            return;
         }
         if(Application.isUserBusy())
         {
            MainConnection.sendMessage(new BattleOfferResult(BattleOfferResult.BUSY,battleOffer.battleId,enemyProfile.id));
            return;
         }
         this.message.show();
         Application.userProfile.initFromStructure(battleOffer.myProfile);
         MainConnection.addMessageListener(CancelBattle,this.onBattleCancel);
         Application.setUserBusy(true);
         this.battleStartInfo = new BattleStartInfo(BattleType.PVP_WITH_FRIEND);
         this.battleStartInfo.mapId = battleOffer.mapId;
         this.battleStartInfo.seed = battleOffer.seed;
         this.battleStartInfo.pvpServer = new Server(battleOffer.host,[battleOffer.port]);
         this.battleStartInfo.battleId = battleOffer.battleId;
         this.socialDataLoader = new UserSocialDataLoader();
         this.socialDataLoader.loadSocialDataByStrucures([enemyProfile],this.onLoadEnemyProfile);
      }
      
      // ---- Patch (2v2 teams) ----
      
      private static function text(key:String, fallback:String) : String
      {
         var s:String = Messages.getMessage(key);
         return !s || s == key ? fallback : s;
      }
      
      private function onTeamOffer(offer:BattleOffer) : void
      {
         var brc:BattleRequestController = Singletons.battleRequestController;
         var name:String = null;
         if(offer.kind == 3)
         {
            name = Boolean(brc.teammate) && brc.teammate.getId() == offer.enemyProfile.id ? UserProfileHelper.getFullNameTextByProfile(brc.teammate) : String(offer.enemyProfile.id);
            brc.teammate = null;
            MessageBox.showMsg(text("TEAM_TITLE","Команда 2 на 2"),Strings.format(text("TEAM_LEFT","{name} більше не у вашій команді."),{"name":name}));
            return;
         }
         if(Application.isUserBusy() || Boolean(this.teamOffer))
         {
            MainConnection.sendMessage(new BattleOfferResult(BattleOfferResult.BUSY,offer.battleId,offer.enemyProfile.id));
            return;
         }
         this.teamOffer = offer;
         Application.setUserBusy(true);
         MainConnection.addMessageListener(CancelBattle,this.onTeamOfferCancel);
         new UserSocialDataLoader().loadSocialDataByStrucures([offer.enemyProfile],this.onTeamOfferProfile);
      }
      
      private function onTeamOfferProfile(profiles:Array, success:Boolean) : void
      {
         var offer:BattleOffer = this.teamOffer;
         if(!offer)
         {
            return; // cancelled meanwhile
         }
         var profile:UserProfile = UserProfile(profiles[0]);
         var name:String = UserProfileHelper.getFullNameTextByProfile(profile);
         var msg:String = null;
         if(offer.kind == 1)
         {
            msg = Strings.format(text("TEAM_INVITE","{name} запрошує вас до своєї команди для боїв 2 на 2. У бою ви будете на одному боці."),{"name":name});
         }
         else
         {
            msg = Strings.format(text("TEAM_CALL","Ваш напарник {name} шукає бій 2 на 2 зі ставкою {wager}. Приєднатися?"),{
               "name":name,
               "wager":String(offer.seed)
            });
         }
         var self:BattleOfferController = this;
         this.teamBox = MessageBox.createMsg(text("TEAM_TITLE","Команда 2 на 2"),msg,MessageBox.OK_CANCEL_BTN,function():void
         {
            self.answerTeamOffer(true,profile);
         });
         this.teamBox.setOkBtnText(offer.kind == 1 ? text("TEAM_ACCEPT","Прийняти") : text("TEAM_JOIN","Прийняти"));
         this.teamBox.setCancelBtnText(text("TEAM_DECLINE","Відмова"));
         this.teamBox.addEventListener(UIEvent.WINDOW_CANCEL,function(e:Event):void
         {
            self.answerTeamOffer(false,profile);
         });
         this.teamBox.show();
      }
      
      private function answerTeamOffer(agree:Boolean, profile:UserProfile) : void
      {
         var offer:BattleOffer = this.teamOffer;
         var wager:PvpWager = null;
         if(!offer)
         {
            return;
         }
         this.endTeamOffer();
         if(agree && offer.kind == 2)
         {
            wager = BattleRequestController.findWager(offer.seed);
            if(!wager || !UserProfileHelper.isEnoughMoney(MoneyType.INGAME,wager) || Application.userProfile.getLevel() < wager.minLevel)
            {
               agree = false;
               MessageBox.showMsg(text("TEAM_TITLE","Команда 2 на 2"),text("TEAM_CANT_JOIN","Ви не можете зіграти з такою ставкою."));
            }
         }
         MainConnection.sendMessage(new BattleOfferResult(agree ? BattleOfferResult.AGREE : BattleOfferResult.DISAGREE,offer.battleId,offer.enemyProfile.id));
         if(!agree)
         {
            return;
         }
         Singletons.battleRequestController.teammate = profile;
         if(offer.kind == 2)
         {
            Singletons.battleRequestController.wagerRequest(wager,true);
         }
      }
      
      private function onTeamOfferCancel(e:IncommingMessageEvent) : void
      {
         if(Boolean(this.teamBox))
         {
            this.teamBox.hide();
         }
         this.endTeamOffer();
      }
      
      private function endTeamOffer() : void
      {
         MainConnection.removeMessageListener(CancelBattle,this.onTeamOfferCancel);
         this.teamOffer = null;
         this.teamBox = null;
         Application.setUserBusy(false);
      }
      
      private function onLoadEnemyProfile(profiles:Array, success:Boolean) : void
      {
         this.battleStartInfo.enemyProfile = profiles[0];
         this.battleOffer(this.battleStartInfo.enemyProfile);
      }
      
      private function battleOffer(profile:UserProfile) : void
      {
         if(!this.dialog)
         {
            this.dialog = new BattleOfferDialog();
            this.dialog.addEventListener(InterfaceEvent.ACCEPT_BATTLE_OFFER,this.onAcceptBattleOffer);
            this.dialog.addEventListener(InterfaceEvent.DECLINE_BATTLE_OFFER,this.onDeclineBattleOffer);
         }
         this.dialog.setProfile(profile);
         this.message.hide();
         this.dialog.show();
      }
      
      private function onAcceptBattleOffer(e:Event) : void
      {
         Message.showBattleLoadingMsg(Messages.BATTLE_INIT);
         MainConnection.sendMessage(new BattleOfferResult(BattleOfferResult.AGREE,this.battleStartInfo.battleId,this.battleStartInfo.enemyProfile.getId()));
         Singletons.battleController.loadDataAndStart(this.battleStartInfo);
         this.reset(false);
      }
      
      private function onDeclineBattleOffer(e:Event) : void
      {
         MainConnection.sendMessage(new BattleOfferResult(BattleOfferResult.DISAGREE,this.battleStartInfo.battleId,this.battleStartInfo.enemyProfile.getId()));
         this.reset(true);
      }
      
      private function reset(cancel:Boolean) : void
      {
         this.battleStartInfo = null;
         MainConnection.removeMessageListener(CancelBattle,this.onBattleCancel);
         this.socialDataLoader = null;
         if(cancel)
         {
            Application.setUserBusy(false);
         }
      }
   }
}

