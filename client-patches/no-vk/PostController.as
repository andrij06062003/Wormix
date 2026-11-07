package ru.pragmatix.wormix.social.common.post
{
   import flash.display.DisplayObject;
   import flash.display.MovieClip;
   import flash.display.Sprite;
   import flash.events.Event;
   import ru.pragmatix.flox.core.ApplicationContext;
   import ru.pragmatix.flox.core.Arrays;
   import ru.pragmatix.flox.core.Strings;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.gui.events.UIEvent;
   import ru.pragmatix.flox.gui.utils.InterfaceUtils;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.social.controller.ISocialController;
   import ru.pragmatix.flox.social.controller.SocialType;
   import ru.pragmatix.flox.social.post.AbstractPostController;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.gui.house.achiev.AchievRecord;
   import ru.pragmatix.wormix.gui.window.PostStatusDialog;
   import ru.pragmatix.wormix.model.battle.BattleResult;
   import ru.pragmatix.wormix.model.battle.BattleResultInfo;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.items.ItemRecord;
   import ru.pragmatix.wormix.model.items.ItemsHelper;
   import ru.pragmatix.wormix.model.items.WeaponRecord;
   import ru.pragmatix.wormix.model.race.Race;
   import ru.pragmatix.wormix.model.race.RaceRecord;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   import ru.pragmatix.wormix.settings.DailyBonusPrize;
   import ru.pragmatix.wormix.settings.DailyBonusType;
   import ru.pragmatix.wormix.settings.MinRequirements;
   import ru.pragmatix.wormix.settings.ScenarioRecord;
   import ru.pragmatix.wormix.social.common.post.schemes.icons.IPostIconsScheme;
   import ru.pragmatix.wormix.social.common.post.schemes.texts.ActivityScheme;
   import ru.pragmatix.wormix.social.common.post.schemes.texts.CommonScheme;
   import ru.pragmatix.wormix.social.common.post.schemes.texts.NotificationScheme;
   import ru.pragmatix.wormix.social.common.post.schemes.texts.WallPostScheme;
   import ru.pragmatix.wormix.social.interfaces.IPostController;
   import ru.pragmatix.wormix.utils.MenuUtils;
   
   public class PostController extends AbstractPostController implements IPostController
   {
      
      public static const ICON_SIZE:int = 100;
      
      protected static const log:Logger = new Logger(PostController);
      
      protected var activityScheme:ActivityScheme;
      
      protected var wallScheme:WallPostScheme;
      
      protected var commonScheme:CommonScheme;
      
      protected var notificationScheme:NotificationScheme;
      
      protected var iconsScheme:IPostIconsScheme;
      
      private var completeMissionsCount:uint = 0;
      
      public function PostController(context:ApplicationContext, socialController:ISocialController, iconsScheme:IPostIconsScheme, iconsFolder:String, activityScheme:ActivityScheme = null, notificationScheme:NotificationScheme = null, wallPostScheme:WallPostScheme = null)
      {
         super(context,socialController,iconsFolder || "data/icons");
         // No VK here: posting to the wall, the activity feed or friends' notifications only ever waited for
         // an answer that never came, so turn all of it off (every poster below is checked for null).
         activityPoster = null;
         wallPoster = null;
         notifier = null;
         this.iconsScheme = iconsScheme;
         if(!activityScheme)
         {
            this.activityScheme = new ActivityScheme();
         }
         if(!wallPostScheme)
         {
            this.wallScheme = new WallPostScheme();
         }
         if(!notificationScheme)
         {
            this.notificationScheme = new NotificationScheme();
         }
         this.commonScheme = new CommonScheme();
         this.commonScheme.checkIfAllTranslationsExists();
         if(Boolean(activityPoster))
         {
            this.activityScheme.checkIfAllTranslationsExists();
         }
         if(Boolean(wallPoster))
         {
            this.wallScheme.checkIfAllTranslationsExists();
         }
         if(Boolean(notifier))
         {
            this.notificationScheme.checkIfAllTranslationsExists();
         }
      }
      
      public function raceChanged(race:RaceRecord) : void
      {
         var icon:String = null;
         if(!Race.WORM.getRaceRecord().equals(race))
         {
            switch(race.configName)
            {
               case "boxer":
                  icon = this.iconsScheme.boxerIcon;
                  break;
               case "demon":
                  icon = this.iconsScheme.demonIcon;
                  break;
               case "rabbit":
                  icon = this.iconsScheme.rabbitIcon;
                  break;
               case "zombie":
                  icon = this.iconsScheme.zombieIcon;
                  break;
               case "cat":
                  icon = this.iconsScheme.catIcon;
            }
            this.askAndPostOnWall(this.wallScheme.raceChangedTitle,this.wallScheme.shareQuestion,this.wallScheme.raceChanged,{"race":race.getName()},icon);
            postActivity(this.activityScheme.raceChanged,{"raceName":race.getName()},icon);
         }
      }
      
      private function winInPvp(profile:UserProfile, result:BattleResultInfo, activityOnly:Boolean = false) : void
      {
         var params:Object = null;
         var enemyName:String = UserProfileHelper.getFullName(profile);
         if(Boolean(result) && Boolean(profile.getRating()))
         {
            params = {
               "enemyName":enemyName,
               "rating":String(profile.getRating())
            };
            postActivity(this.activityScheme.winInPvp,params,this.iconsScheme.winInPvp);
         }
         else
         {
            postActivity(this.activityScheme.winInPvpNoRating,{"enemyName":enemyName},this.iconsScheme.winInPvp);
         }
      }
      
      public function postBattleDescription(start:BattleStartInfo, result:BattleResultInfo, callback:Function) : void
      {
      }
      
      public function rubyFoundInHouse(friendProfile:UserProfile) : void
      {
         var params:Object = {"name":UserProfileHelper.getFullNameTextByProfile(friendProfile)};
         postActivity(this.activityScheme.rubyFoundInHouse,params,this.iconsScheme.rubyFoundInHouse);
         this.askAndPostOnWall(this.wallScheme.rubyFoundTitle,this.wallScheme.shareQuestion,this.wallScheme.rubyFoundInHouse,params,this.iconsScheme.rubyFoundInHouse);
      }
      
      public function kickTeamMate(profile:UserProfile) : void
      {
         postActivity(this.activityScheme.kickTeamMate,{"name":UserProfileHelper.getFullNameTextByProfile(profile)},this.iconsScheme.kickTeamMate);
         this.askAndNotify(profile,this.notificationScheme.kickTeamMateTitle,this.notificationScheme.kickTeamMateQuestion,this.notificationScheme.kickTeamMate,null,this.iconsScheme.kickTeamMate);
      }
      
      private function missionCompleteActivity() : void
      {
         postActivity(this.activityScheme.missionComplete,null,this.iconsScheme.missionComplete);
      }
      
      public function enter4DaySequence() : void
      {
         postActivity(this.activityScheme.enter4DaySequence,null,this.iconsScheme.enter4DaySequence);
      }
      
      public function share4DayBonus() : void
      {
         postOnWall(this.wallScheme.enter4DaySequence,null,this.iconsScheme.enter4DaySequence);
      }
      
      protected function getPrizeIcon(prize:DailyBonusPrize) : String
      {
         var icon:String = null;
         if(prize.type == DailyBonusType.FIGHT)
         {
            icon = this.iconsScheme.enter5DaySequence_2mission;
         }
         else if(prize.type == DailyBonusType.REACTION)
         {
            icon = this.iconsScheme.enter5DaySequence_reaction;
         }
         else if(prize.type == DailyBonusType.FUZY)
         {
            icon = this.iconsScheme.enter5DaySequence_fuzzes;
         }
         else if(prize.type == DailyBonusType.RUBY)
         {
            icon = this.iconsScheme.enter5DaySequence_rubies;
         }
         return icon;
      }
      
      public function enter5DaySequence(prize:DailyBonusPrize) : void
      {
         var msg:String = null;
         var icon:String = null;
         if(Boolean(activityPoster))
         {
            if(prize.type == DailyBonusType.FIGHT)
            {
               msg = this.activityScheme.enter5DaySequence_2mission;
            }
            else if(prize.type == DailyBonusType.REACTION)
            {
               msg = this.activityScheme.enter5DaySequence_reaction;
            }
            else if(prize.type == DailyBonusType.FUZY)
            {
               msg = this.activityScheme.enter5DaySequence_fuzzes;
            }
            else if(prize.type == DailyBonusType.RUBY)
            {
               msg = this.activityScheme.enter5DaySequence_rubies;
            }
            icon = this.getPrizeIcon(prize);
            postActivity(this.activityScheme.enter5DaySequence,{"reward":Strings.format(msg,{"count":prize.count})},icon);
         }
      }
      
      public function share5DaySequence(prize:DailyBonusPrize) : void
      {
         var msg:String = null;
         var icon:String = null;
         if(Boolean(wallPoster))
         {
            if(prize.type == DailyBonusType.FIGHT)
            {
               msg = this.wallScheme.enter5DaySequence_2mission;
            }
            else if(prize.type == DailyBonusType.REACTION)
            {
               msg = this.wallScheme.enter5DaySequence_reaction;
            }
            else if(prize.type == DailyBonusType.FUZY)
            {
               msg = this.wallScheme.enter5DaySequence_fuzzes;
            }
            else if(prize.type == DailyBonusType.RUBY)
            {
               msg = this.wallScheme.enter5DaySequence_rubies;
            }
            icon = this.getPrizeIcon(prize);
            postOnWall(this.wallScheme.enter5DaySequence,{"reward":Strings.format(msg,{"count":prize.count})},icon);
         }
      }
      
      public function newLevel(level:uint) : void
      {
         var icon:String = this.iconsScheme.getNewLevelByRace(Application.userProfile.getCharType());
         postOnWall(this.wallScheme.newLevel,{"level":level},icon);
         postActivity(this.activityScheme.newLevel,{"level":level},icon);
      }
      
      protected function newHat(hatName:String, item:HatRecord) : void
      {
         this.askAndPostOnWall(this.wallScheme.newHatTitle,this.wallScheme.shareQuestion,this.wallScheme.newHat,{"hat":hatName},this.iconsScheme.getHatIcon(item));
      }
      
      protected function newWeapon(weaponName:String, item:WeaponRecord) : void
      {
      }
      
      protected function newExpensiveWeapon(weaponName:String, item:WeaponRecord) : void
      {
         this.askAndPostOnWall(this.wallScheme.newWeaponTitle,this.wallScheme.shareQuestion,this.wallScheme.newWeapon,{"weapon":weaponName},this.iconsScheme.getWeaponIcon(item));
      }
      
      protected function newItems(itemNames:Array, items:Array) : void
      {
         postActivity(this.activityScheme.newItems,{"items":itemNames.join(", ")},this.iconsScheme.newItems);
      }
      
      public function newPhotoAdded(photoUrl:String) : void
      {
         postActivity(this.activityScheme.newPhotoAdded,null,this.iconsScheme.postStatus,photoUrl);
      }
      
      public function newTeamMate(profile:UserProfile) : void
      {
         postActivity(this.activityScheme.newTeamMate,{"name":UserProfileHelper.getFullNameTextByProfile(profile)},this.iconsScheme.newTeamMate);
         this.askAndNotify(profile,"",this.notificationScheme.newTeamMateQuestion,this.notificationScheme.newTeamMate,null,this.iconsScheme.newTeamMate);
      }
      
      public function callToPvp(profile:UserProfile, enemyProfile:UserProfile) : void
      {
         var icon:String = this.iconsScheme.callToPvp[Singletons.randomSeed.getIntInRange(0,this.iconsScheme.callToPvp.length - 1)];
         var msg:String = this.getCallToPvpText(profile,enemyProfile);
         if(Boolean(notifier))
         {
            notifier.notify(enemyProfile,msg,"",iconsDir + icon);
         }
         else if(Boolean(activityPoster))
         {
            activityPoster.activity(msg,iconsDir + icon,Messages.PLAY_CAPTION);
         }
      }
      
      protected function getCallToPvpText(profile:UserProfile, enemyProfile:UserProfile) : String
      {
         var params:Object = {
            "rating":profile.getRating(),
            "name":UserProfileHelper.getFullNameTextByProfile(enemyProfile)
         };
         return Strings.format(Boolean(profile.getRating()) ? this.commonScheme.callToPvp : this.commonScheme.callToPvpNoRating,params);
      }
      
      public function activityPumpFriendsReaction(profile:UserProfile, count:uint) : void
      {
         postActivity(Application.userProfile.getSocialData().isFemale() ? this.activityScheme.activityPumpFriendsReactionW : this.activityScheme.activityPumpFriendsReaction,{
            "name":UserProfileHelper.getFullNameTextByProfile(profile),
            "count":count
         },this.iconsScheme.activityPumpFriendsReaction);
      }
      
      public function requestPumpReaction(profile:UserProfile, needToAsk:Boolean) : void
      {
         if(Boolean(profile))
         {
            if(needToAsk)
            {
               this.askAndNotify(profile,"",this.notificationScheme.requestPumpReactionQuestion,this.notificationScheme.requestPumpReaction,null,this.iconsScheme.requestPumpReaction);
            }
            else
            {
               notify(profile,this.notificationScheme.requestPumpReaction,null,this.iconsScheme.requestPumpReaction);
            }
         }
         else
         {
            MessageBox.showMsg(Messages.TITLE_ATTENTION,this.notificationScheme.requestPumpReactionEmptySelection);
         }
      }
      
      public function winInBattleWithFriend(profile:UserProfile, callback:Function) : void
      {
         if(Boolean(this.notificationScheme.remindAboutLose) || Singletons.socialController.getType().equals(SocialType.VK))
         {
            this.askAndNotify(profile,this.notificationScheme.remindAboutLoseTitle,this.notificationScheme.remindAboutLoseQuestion,this.notificationScheme.remindAboutLose,null,this.iconsScheme.remindAboutLose,callback);
         }
      }
      
      public function shareAchiev(achievRecord:AchievRecord, level:int, icon:MovieClip) : void
      {
         var totalMsg:uint = 0;
         var text:String = null;
         if(Boolean(wallPoster))
         {
            totalMsg = this.wallScheme.achievTexts.length;
            text = Strings.format(this.wallScheme.achievTexts[int(Math.random() * totalMsg)],{
               "achievement_name":achievRecord.levels[level].name,
               "user_name":UserProfileHelper.getFullName(Application.userProfile)
            });
            wallPoster.wallPostLoadedImg(text,Messages.GAME_NAME,icon,ICON_SIZE,ICON_SIZE);
         }
      }
      
      public function shareAward(item:ItemRecord) : void
      {
         var textMsg:String = this.wallScheme.achievAwardTexts[int(Math.random() * this.wallScheme.achievAwardTexts.length)];
         this.askAndPostOnWall(this.wallScheme.achieveAwardQuestionTitle,this.wallScheme.shareQuestion,textMsg,{});
      }
      
      public function postGameOverBoss(isPost:Boolean, bossInfo:ScenarioRecord, result:BattleResult) : void
      {
         var wallSchemeText:String = null;
         if(result == BattleResult.WINNER)
         {
            wallSchemeText = this.activityScheme.winOverBossText;
         }
         else if(result == BattleResult.LOSER)
         {
            wallSchemeText = this.wallScheme.loseOverBossText;
         }
         else if(result == BattleResult.DRAW_GAME)
         {
            wallSchemeText = this.wallScheme.drawOverBossText;
         }
         var text:String = Strings.format(wallSchemeText,{
            "user_name":UserProfileHelper.getFullName(Application.userProfile),
            "boss_name":bossInfo.getName()
         });
         if(isPost)
         {
            if(result == BattleResult.WINNER)
            {
               postOnWall(this.wallScheme.winBoss[bossInfo.getId() - 1],null,this.iconsScheme.winOnBossIcon(bossInfo.getId() - 1));
            }
            else if(result == BattleResult.LOSER)
            {
               postOnWall(text,null,this.iconsScheme.bossIcon);
            }
            else if(result == BattleResult.DRAW_GAME)
            {
               postOnWall(text,null,this.iconsScheme.bossIcon);
            }
         }
         postActivity(text,null,this.iconsScheme.bossIcon);
      }
      
      public function postTutorialComplete(nameMission:String) : void
      {
         postOnWall(this.wallScheme.tutorialComplete,{"name_mission":nameMission},this.iconsScheme.tutorialIcon);
      }
      
      public function postBattleResultActivity(startInfo:BattleStartInfo, resultInfo:BattleResultInfo) : void
      {
         if(resultInfo.result.equals(BattleResult.WINNER))
         {
            if(startInfo.battleType == BattleType.MISSION)
            {
               if(!this.completeMissionsCount)
               {
                  this.missionCompleteActivity();
                  ++this.completeMissionsCount;
               }
            }
            if(startInfo.battleType == BattleType.PVP_WAGER || startInfo.battleType == BattleType.PVP_WITH_FRIEND)
            {
               this.winInPvp(startInfo.enemyProfile,resultInfo);
            }
         }
      }
      
      public function newItem(item:ItemRecord) : void
      {
         var name:String = null;
         if(Boolean(item))
         {
            name = ItemsHelper.getItemName(item);
            if(item is HatRecord)
            {
               this.newHat(name,item as HatRecord);
            }
            else if(item is WeaponRecord)
            {
               if(item.getIngamePrice() >= MinRequirements.EXPENSIVE_PRICE_THRESHOLD || item.getRealPrice() >= MinRequirements.EXPENSIVE_REALPRICE_THRESHOLD)
               {
                  this.newExpensiveWeapon(name,item as WeaponRecord);
               }
               else
               {
                  this.newWeapon(name,item as WeaponRecord);
               }
            }
         }
      }
      
      public function newItemList(items:Array) : void
      {
         var item:ItemRecord = null;
         var itemNames:Array = [];
         for each(item in items)
         {
            itemNames.push(ItemsHelper.getItemName(item));
         }
         this.newItems(itemNames,items);
      }
      
      public function postToStatus(start:BattleStartInfo, result:BattleResultInfo, callback:Function) : void
      {
         var text:Array = this.getStatusText(start,result);
         var d:PostStatusDialog = new PostStatusDialog();
         d.setText(text,this.postStatus);
         d.addEventListener(UIEvent.WINDOW_CLOSE,function(e:Event):void
         {
            callback();
         });
         d.show();
      }
      
      private function postStatus(text:String) : void
      {
         if(Boolean(wallPoster))
         {
            wallPoster.wallPost(text,"Test Wormix",iconsDir + this.iconsScheme.postStatus);
         }
         else if(Boolean(activityPoster))
         {
            activityPoster.activity(text,iconsDir + this.iconsScheme.postStatus,Messages.PLAY_CAPTION);
         }
      }
      
      private function getStatusText(startInfo:BattleStartInfo, resultInfo:BattleResultInfo) : Array
      {
         var status:String = null;
         var text:Array = [];
         var p:UserProfile = Application.userProfile;
         var params:Object = {
            "level":p.getLevel(),
            "rating":p.getRating(),
            "opponent":UserProfileHelper.getFullNameTextByProfile(startInfo.enemyProfile)
         };
         if(resultInfo.result.equals(BattleResult.WINNER))
         {
            text.push(Strings.format(this.commonScheme.status_winner,params));
         }
         if(startInfo.battleType.isPvp())
         {
            text.push(this.getCallToPvpText(p,startInfo.enemyProfile));
         }
         var statuses:Array = [this.commonScheme.status0,this.commonScheme.status1,this.commonScheme.status2,this.commonScheme.status3];
         var need:uint = 3 - text.length;
         for(var i:uint = 0; i < need; i++)
         {
            status = statuses[Singletons.randomSeed.getIntInRange(0,statuses.length - 1)];
            text.push(status);
            Arrays.remove(statuses,status);
         }
         return text;
      }
      
      public function isPostBattleDescriptionAvailable() : Boolean
      {
         return false;
      }
      
      public function isFriendNotificationAvailable() : Boolean
      {
         return notifier != null;
      }
      
      protected function askAndPostOnWall(questionTitle:String, question:String, msg:String, params:Object = null, icon:String = null) : void
      {
         var msgBox:MessageBox = null;
         if(Boolean(wallPoster))
         {
            msgBox = MessageBox.createMsg(questionTitle,question,MessageBox.OK_CANCEL_BTN,function():void
            {
               postOnWall(msg,params,icon);
            });
            InterfaceUtils.centerYBy(msgBox.getChildByName("msg"),msgBox.getChildByName("heightLine"));
            msgBox.show();
         }
      }
      
      protected function askAndPostOnWallLoadedIcon(questionTitle:String, question:String, msg:String, icon:DisplayObject, params:Object = null, height:uint = 0, width:uint = 0) : void
      {
         var msgBox:MessageBox = null;
         if(Boolean(wallPoster))
         {
            msgBox = MessageBox.createMsg(questionTitle,question,MessageBox.OK_CANCEL_BTN,function():void
            {
               wallPoster.wallPostLoadedImg(Strings.format(msg,params),Messages.GAME_NAME,icon,height,width);
            });
            InterfaceUtils.centerYBy(msgBox.getChildByName("msg"),msgBox.getChildByName("heightLine"));
            msgBox.show();
         }
      }
      
      protected function askAndNotify(profile:UserProfile, questionTitle:String, question:String, msg:String, params:Object = null, icon:String = null, callback:Function = null) : void
      {
         var msgBox:MessageBox = null;
         if(Boolean(notifier))
         {
            msgBox = MessageBox.createMsg(questionTitle,question,MessageBox.OK_CANCEL_BTN,function():void
            {
               notify(profile,msg,params,icon);
            });
            InterfaceUtils.centerYBy(msgBox.getChildByName("msg"),msgBox.getChildByName("heightLine"));
            msgBox.addEventListener(UIEvent.WINDOW_CLOSE,function(e:Event):void
            {
               if(callback != null)
               {
                  callback();
               }
            });
            msgBox.show();
         }
         else if(callback != null)
         {
            // Nothing to ask: carry on (e.g. the level-up dialog after a won friend battle).
            callback();
         }
      }
      
      protected function getItemIcon(item:ItemRecord) : DisplayObject
      {
         var icon:DisplayObject = MenuUtils.getMenuIcon(item);
         InterfaceUtils.scaleByMaxSize(icon,ICON_SIZE * 0.9,ICON_SIZE * 0.9);
         var iconCnt:Sprite = new Sprite();
         iconCnt.addChild(icon);
         InterfaceUtils.alignByParentCenter(icon);
         return iconCnt;
      }
   }
}

