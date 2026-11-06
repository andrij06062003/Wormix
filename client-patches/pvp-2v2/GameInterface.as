package ru.pragmatix.wormix.gui
{
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.events.KeyboardEvent;
   import flash.events.MouseEvent;
   import flash.text.TextField;
   import ru.pragmatix.flox.core.Cookie;
   import ru.pragmatix.flox.gui.controls.UIContainer;
   import ru.pragmatix.flox.gui.controls.button.ButtonFactory;
   import ru.pragmatix.flox.gui.controls.button.IUIButton;
   import ru.pragmatix.flox.gui.controls.hint.HintMode;
   import ru.pragmatix.flox.gui.controls.message.PopUpMessage;
   import ru.pragmatix.flox.gui.controls.scroller.UIScrollBar;
   import ru.pragmatix.flox.gui.core.UISettings;
   import ru.pragmatix.flox.loader.ClassLoader;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.secure.ProtectedBoolean;
   import ru.pragmatix.wormix.Scene;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.events.OpenCloseBagEvent;
   import ru.pragmatix.wormix.gui.game.CharBattleHintControl;
   import ru.pragmatix.wormix.gui.game.Chat;
   import ru.pragmatix.wormix.gui.game.HatsPanel;
   import ru.pragmatix.wormix.gui.game.QualityControlPanel;
   import ru.pragmatix.wormix.gui.game.SoundControlPanel;
   import ru.pragmatix.wormix.gui.game.TeamBar;
   import ru.pragmatix.wormix.gui.game.TimerControl;
   import ru.pragmatix.wormix.gui.game.WeaponTipsControl;
   import ru.pragmatix.wormix.gui.game.Wind;
   import ru.pragmatix.wormix.gui.menu.BagMenu;
   import ru.pragmatix.wormix.gui.menu.HelpControl;
   import ru.pragmatix.wormix.gui.window.ScreenshotMaker;
   import ru.pragmatix.wormix.gui.window.VideoMaker;
   import ru.pragmatix.wormix.gui.window.gameover.SurrenderMessage;
   import ru.pragmatix.wormix.loader.Classes;
   import ru.pragmatix.wormix.model.Sounds;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.objects.Worm;
   import ru.pragmatix.wormix.scene.SceneCamera;
   import ru.pragmatix.wormix.secure.CheatDetectedEvent;
   import ru.pragmatix.wormix.secure.Jammer;
   
   public class GameInterface extends UIContainer
   {
      
      public static const MAX_SCALE:Number = 1.2;
      
      public static const MIN_SCALE:Number = 0.55;
      
      public static const NORMAL_SCALE:Number = 1;
      
      public static const ALL_INFO:uint = 0;
      
      public static const HP_ONLY:uint = 1;
      
      public static const DISABLE_INFO:uint = 2;
      
      private var scene:Scene;
      
      private var scale:Number = 0.55;
      
      private var lastScale:Number = 0.55;
      
      public var scrollBar:UIScrollBar;
      
      private var screenshot:ScreenshotMaker;
      
      public var videoMaker:VideoMaker;
      
      private var exitBtn:IUIButton;
      
      private var lastMouseX:Number = 0;
      
      private var lastMouseY:Number = 0;
      
      private var timer:TimerControl;
      
      public var wind:Wind;
      
      public var bagBtn:IUIButton;
      
      private var bagBtnDisabled:ProtectedBoolean;
      
      private var enemyTurnMsg:TextField;
      
      public var chat:Chat;
      
      public var help:IUIButton;
      
      public var helpMenu:HelpControl;
      
      public var soundControl:SoundControlPanel;
      
      public var qualityControl:QualityControlPanel;
      
      public var weaponHintBtn:WeaponTipsControl;
      
      private var surrenderMsg:SurrenderMessage;
      
      public var infoState:uint = 0;
      
      private var charBattleHintBtn:CharBattleHintControl;
      
      public var bagMenu:BagMenu;
      
      public var hatsPanel:HatsPanel;
      
      private var teams:Array;
      
      private var sceneClick:Boolean;
      
      private var log:Logger = new Logger(GameInterface);
      
      public var openBagByHotkey:Boolean = true;
      
      private var bagBtnActive:Boolean = true;
      
      public function GameInterface(scene:Scene)
      {
         super(ClassLoader.getNewInstance(Classes.BATTLE_INTERFACE));
         this.scene = scene;
         this.help = ButtonFactory.createButton(getChildByName("help"));
         this.help.setQtip(Messages.HELP_TITLE,HintMode.RIGHT);
         this.help.setSoundOnClick(null);
         this.help.setSoundOnMouseOver(Sounds.MENU_TURN);
         this.help.addHandler(this.onHelpClick);
         this.scrollBar = new UIScrollBar(getChildByName("slider") as MovieClip);
         this.scrollBar.setMaxValue(100);
         this.screenshot = new ScreenshotMaker(getChildByName("photo"),UISettings.scene);
         this.screenshot.setQtip(Messages.PHOTO_QTIP,HintMode.LEFT);
         var video:MovieClip = getChildByName("videoBlock") as MovieClip;
         if(Singletons.socialController.isVideoAvailable() || Singletons.socialController.isYoutubeAvailable())
         {
            this.videoMaker = new VideoMaker(video);
            this.videoMaker.setQtip(Messages.VIDEO_QTIP,HintMode.LEFT);
         }
         else
         {
            video.visible = false;
         }
         this.bagMenu = new BagMenu();
         this.bagBtn = ButtonFactory.createButton(getChildByName("bag"));
         this.bagBtn.setSoundOnClick(null);
         this.bagBtnDisabled = new ProtectedBoolean(false);
         this.setBagBtnDisabled(true);
         this.exitBtn = ButtonFactory.createButton(getChildByName("exit"));
         this.timer = new TimerControl(getChildByName("timer") as MovieClip);
         this.timer.setVisible(false);
         this.wind = new Wind(getChildByName("wind") as MovieClip);
         this.enemyTurnMsg = getChildByNameAsTextField("enemyTurnMsg");
         this.enemyTurnMsg.text = Messages.ENEMY_TURN_MSG;
         _removeChild(this.enemyTurnMsg);
         PopUpMessage.init(getChildByNameAsTextField("taunts"));
         this.charBattleHintBtn = new CharBattleHintControl(getChildByNameAsMovieClip("charHintBtn"),this);
         this.charBattleHintBtn.setQtip(Messages.CHAR_HINT_ON_OFF,HintMode.LEFT);
         this.weaponHintBtn = new WeaponTipsControl(getChildByNameAsMovieClip("tips"),this);
         this.weaponHintBtn.setQtip(Messages.TIPS_ON_OFF,HintMode.RIGHT);
         this.soundControl = new SoundControlPanel(getChildByName("sound") as MovieClip);
         this.qualityControl = new QualityControlPanel(getChildByName("qualityControl"),scene.getContainer());
         this.hatsPanel = new HatsPanel(getChildByNameAsMovieClip("hatPanel"));
         this.hatsPanel.setVisible(false);
         destroyOnRemove = true;
         this.initChat();
      }
      
      public function init(battleType:BattleType) : void
      {
         this.chat.setVisibleShowHideBtn(battleType != BattleType.MISSION && battleType != BattleType.HOT_SEAT && battleType != BattleType.SCENARIO);
      }
      
      private function onHelpClick(e:Event) : void
      {
         this.showHelp();
      }
      
      public function showHelp() : void
      {
         if(!this.helpMenu)
         {
            this.helpMenu = new HelpControl();
         }
         this.helpMenu.show();
      }
      
      public function showHats(hats:Array) : void
      {
         this.hatsPanel.setData(hats);
         this.hatsPanel.update();
         this.hatsPanel.setVisible(true);
      }
      
      public function hideHats() : void
      {
         this.hatsPanel.setVisible(false);
      }
      
      public function initTeams(squads:Array, battleType:BattleType) : void
      {
         var i:int = 0;
         var team:TeamBar = null;
         var t:TeamBar = null;
         this.teams = [];
         var minHp:int = 100000;
         for(var TOTAL_TEAM_BAR:uint = 4; i < TOTAL_TEAM_BAR; )
         {
            t = new TeamBar(getChildByName("team" + i) as DisplayObjectContainer);
            if(i <= squads.length - 1)
            {
               this.teams.push(t);
               t.setTeam(squads[i],battleType);
               minHp = Math.min(minHp,t.getTeam().profile.getTeamHp());
            }
            else
            {
               t.setVisible(false);
            }
            i++;
         }
         this.help.y = this.teams[this.teams.length - 1].y + this.teams[this.teams.length - 1].height;
         this.weaponHintBtn.y = this.help.y + this.help.height + 5;
         for each(team in this.teams)
         {
            team.setMinHp(minHp);
         }
      }
      
      public function activateWorm(worm:Worm) : void
      {
         var bar:TeamBar = null;
         for each(bar in this.teams)
         {
            bar.update();
            bar.setActive(bar.getTeam() == worm.squad);
         }
      }
      
      public function initChat() : void
      {
         this.chat = new Chat();
         this.chat.x = 0;
         this.chat.y = UISettings.stageHeight - this.chat.height;
         addChild(this.chat);
      }
      
      public function getChatPanel() : Chat
      {
         return this.chat;
      }
      
      public function setBagBtnDisabled(disabled:Boolean) : void
      {
         if(!disabled && !this.bagBtnActive)
         {
            return;
         }
         this.bagBtn.setDisabled(disabled);
         this.bagBtnDisabled.value = disabled;
         if(disabled && this.bagMenu.isShown())
         {
            this.bagMenu.hide();
         }
      }
      
      public function disableExitButton() : void
      {
         this.exitBtn.setDisabled(true);
      }
      
      public function showEnemyTurnMsg() : void
      {
         // Patch (2v2): during a teammate's turn say so instead of "the opponent's turn".
         this.enemyTurnMsg.text = this.isTeammateTurn() ? teammateTurnText() : Messages.ENEMY_TURN_MSG;
         _addChild(this.enemyTurnMsg);
      }
      
      private static function teammateTurnText() : String
      {
         var s:String = Messages.getMessage("TEAMMATE_TURN_MSG");
         return !s || s == "TEAMMATE_TURN_MSG" ? "очікування ходу напарника" : s;
      }
      
      // Uses the server's team list, so it still works after this player's own worms are all dead.
      private function isTeammateTurn() : Boolean
      {
         var info:BattleStartInfo = null;
         var activeId:uint = 0;
         var myTeam:int = -1;
         var activeTeam:int = -2;
         var i:int = 0;
         if(!Singletons.btc || !Singletons.btc.gameMaster || !Singletons.btc.gameMaster.activeSquad)
         {
            return false;
         }
         info = Singletons.btc.getBattleInfo();
         if(!info || !info.isTeamBattle())
         {
            return false;
         }
         activeId = Singletons.btc.gameMaster.activeSquad.profile.getId();
         for(i = 0; i < info.teamProfiles.length; i++)
         {
            if(info.teamProfiles[i].getId() == Application.userProfile.getId())
            {
               myTeam = int(info.teamIds[i]);
            }
            if(info.teamProfiles[i].getId() == activeId)
            {
               activeTeam = int(info.teamIds[i]);
            }
         }
         return myTeam == activeTeam;
      }
      
      public function hideEnemyTurnMsg() : void
      {
         if(_contains(this.enemyTurnMsg))
         {
            _removeChild(this.enemyTurnMsg);
         }
      }
      
      public function startTimer(delay:int, onComplete:Function) : void
      {
         this.timer.start(delay,onComplete);
      }
      
      public function stopTimer(hide:Boolean = true) : void
      {
         this.timer.stop();
         if(hide)
         {
            this.timer.hide();
         }
      }
      
      public function pauseTimer() : void
      {
         if(!Singletons.btc.gameMaster.getSettings().online)
         {
            this.timer.pause();
         }
      }
      
      public function resumeTimer() : void
      {
         if(!Singletons.btc.gameMaster.getSettings().online)
         {
            this.timer.resume();
         }
      }
      
      public function getTime() : uint
      {
         return this.timer.getTime();
      }
      
      public function deactivateBagButton() : void
      {
         this.setBagBtnDisabled(true);
         this.bagBtnActive = false;
      }
      
      override protected function initComponent() : void
      {
         super.initComponent();
         this.bagBtn.addHandler(this.onBagClick);
         this.bagBtn.setQtip(Messages.BAG_QTIP,HintMode.UNDER);
         getContainer().stage.addEventListener(KeyboardEvent.KEY_DOWN,this.onKeyPress);
         this.scene.addEventListener(MouseEvent.MOUSE_WHEEL,this.onSceneMouseScroll);
         this.scene.addEventListener(MouseEvent.MOUSE_MOVE,this.onSceneMouseMove);
         this.scene.addEventListener(MouseEvent.MOUSE_DOWN,this.onSceneMouseDown);
         this.scene.addEventListener(MouseEvent.MOUSE_UP,this.onSceneMouseUp);
         this.scrollBar.addEventListener(Event.CHANGE,this.onScrollChange);
         this.exitBtn.addHandler(this.onFlagClick);
         this.exitBtn.setQtip(Messages.FLAG_QTIP,HintMode.UNDER);
      }
      
      override public function destroy() : void
      {
         var bar:TeamBar = null;
         if(Boolean(this.videoMaker))
         {
            this.videoMaker.save();
            this.videoMaker.destroy();
         }
         if(this.hatsPanel.isShown())
         {
            this.hatsPanel.hide();
         }
         this.help.destroy();
         this.hatsPanel.destroy();
         this.scrollBar.destroy();
         this.bagBtn.destroy();
         this.bagMenu.destroy();
         this.timer.destroy();
         this.charBattleHintBtn.destroy();
         this.weaponHintBtn.destroy();
         this.soundControl.destroy();
         this.qualityControl.destroy();
         this.wind.destroy();
         this.chat.destroy();
         PopUpMessage.reset();
         getContainer().stage.removeEventListener(KeyboardEvent.KEY_DOWN,this.onKeyPress);
         this.scene.removeEventListener(MouseEvent.MOUSE_WHEEL,this.onSceneMouseScroll);
         this.scene.removeEventListener(MouseEvent.MOUSE_MOVE,this.onSceneMouseMove);
         this.scene.removeEventListener(MouseEvent.MOUSE_DOWN,this.onSceneMouseDown);
         this.scene.removeEventListener(MouseEvent.MOUSE_UP,this.onSceneMouseUp);
         this.scrollBar.removeEventListener(Event.CHANGE,this.onScrollChange);
         this.exitBtn.destroy();
         if(Boolean(this.screenshot))
         {
            this.screenshot.destroy();
         }
         SceneCamera.kill(this.scene);
         for each(bar in this.teams)
         {
            bar.destroy();
         }
         this.teams = [];
         this.teams.length = 0;
         this.scene = null;
         super.destroy();
      }
      
      private function onKeyPress(event:KeyboardEvent) : void
      {
         switch(event.keyCode)
         {
            case HotKeys.BAG_OPEN:
            case HotKeys.BAG_OPEN_OLD:
               if(this.openBagByHotkey)
               {
                  this.toggleShowBag();
               }
               break;
            case HotKeys.BAG_OPEN_1:
            case HotKeys.BAG_OPEN_2:
            case HotKeys.BAG_OPEN_3:
            case HotKeys.BAG_OPEN_4:
            case HotKeys.BAG_OPEN_5:
            case HotKeys.BAG_OPEN_6:
            case HotKeys.BAG_OPEN_7:
            case HotKeys.BAG_OPEN_8:
            case HotKeys.BAG_OPEN_9:
               if(event.shiftKey || event.ctrlKey)
               {
                  this.toggleShowBag(event.keyCode);
               }
               break;
            case HotKeys.CHAT_OPEN:
               if(event.ctrlKey)
               {
                  if(this.chat != null)
                  {
                     this.chat.setVisible(!this.chat.isVisible());
                  }
               }
               break;
            case HotKeys.ZOOM_IN:
            case HotKeys.ZOOM_IN_NUMPAD:
               this.scrollBar.setValue(this.scrollBar.getValue() - this.scrollBar.getStep());
               break;
            case HotKeys.ZOOM_OUT:
            case HotKeys.ZOOM_OUT_NUMPAD:
               this.scrollBar.setValue(this.scrollBar.getValue() + this.scrollBar.getStep());
               break;
            case HotKeys.REC:
               if(event.ctrlKey && Singletons.socialController.isVideoAvailable() && !this.videoMaker.isDialogOpened() && (this.screenshot == null || !this.screenshot.isDialogOpened()))
               {
                  this.videoMaker.onVideoClick();
               }
               break;
            case HotKeys.PHOTO:
               if(event.ctrlKey && Singletons.socialController.isAlbumsAvailable() && (this.videoMaker == null || !this.videoMaker.isDialogOpened()))
               {
                  this.screenshot.onScreenshotClick();
               }
               break;
            case HotKeys.UPDATE_HINT:
               if(event.ctrlKey)
               {
                  this.charBattleHintBtn.update(null);
               }
         }
      }
      
      private function onFlagClick(event:Event) : void
      {
         if(!this.surrenderMsg)
         {
            this.surrenderMsg = new SurrenderMessage();
            this.surrenderMsg.show();
         }
         else
         {
            this.surrenderMsg.show();
         }
      }
      
      private function onBagClick(event:MouseEvent) : void
      {
         this.toggleShowBag();
         if(Boolean(Singletons.learningMissionController))
         {
            this.bagBtn.dispatchEvent(new OpenCloseBagEvent(true));
         }
      }
      
      private function toggleShowBag(hotKey:uint = 0) : void
      {
         if(this.bagBtn.isDisabled() != this.bagBtnDisabled.value)
         {
            Application.curScene.dispatchEvent(new CheatDetectedEvent(Jammer.CHEAT_ENGINE_BAN_TYPE,"bag button hack 1"));
            this.bagBtn.setDisabled(this.bagBtnDisabled.value);
         }
         if(this.bagMenu.isShown())
         {
            this.bagMenu.hide();
         }
         else if(!this.bagBtnDisabled.value)
         {
            if(Singletons.btc.gameMaster.playersTurn())
            {
               this.bagMenu.initFromProfile(Singletons.btc.gameMaster.getCurrentProfile());
               this.bagMenu.showWitHotKeys(true,hotKey);
            }
            else
            {
               Application.curScene.dispatchEvent(new CheatDetectedEvent(Jammer.CHEAT_ENGINE_BAN_TYPE,"bag button hack 2"));
            }
         }
      }
      
      public function onScrollChange(e:Event) : void
      {
         this.scale = (MAX_SCALE - MIN_SCALE) * (100 - this.scrollBar.getValue()) / 100 + MIN_SCALE;
         if(this.lastScale != this.scale)
         {
            this.lastScale = this.scale;
            SceneCamera.slowCameraScale(this.scene,this.scale);
            Cookie.setValue(Scene.SCALE_COOKIE_NAME,this.scale,false);
         }
      }
      
      public function onSceneMouseScroll(event:MouseEvent) : void
      {
         this.scrollBar.setValue(this.scrollBar.getValue() - 2 * event.delta);
      }
      
      public function setSliderPositionByScale(scale:Number) : void
      {
         this.scale = scale;
         var newPercent:int = 100 - (scale - MIN_SCALE) * 100 / (MAX_SCALE - MIN_SCALE);
         this.scrollBar.setValue(newPercent,false);
      }
      
      private function onSceneMouseDown(event:MouseEvent) : void
      {
         this.lastMouseX = event.stageX;
         this.lastMouseY = event.stageY;
         this.sceneClick = true;
      }
      
      private function onSceneMouseUp(event:MouseEvent) : void
      {
         this.sceneClick = false;
      }
      
      public function onSceneMouseMove(event:MouseEvent) : void
      {
         var deltaX:int = 0;
         var deltaY:int = 0;
         if(event.buttonDown && this.sceneClick)
         {
            deltaX = this.lastMouseX - event.stageX;
            deltaY = this.lastMouseY - event.stageY;
            this.lastMouseX = event.stageX;
            this.lastMouseY = event.stageY;
            SceneCamera.cameraMove(this.scene,deltaX,deltaY);
         }
      }
   }
}

