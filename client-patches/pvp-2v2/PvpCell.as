package ru.pragmatix.wormix.gui.menu.arena.pvp
{
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.text.TextField;
   import ru.pragmatix.flox.core.Strings;
   import ru.pragmatix.flox.gui.controls.MessageBox;
   import ru.pragmatix.flox.gui.events.UIEvent;
   import ru.pragmatix.flox.gui.controls.UIContainer;
   import ru.pragmatix.flox.gui.controls.button.ButtonFactory;
   import ru.pragmatix.flox.gui.controls.button.IUIButton;
   import ru.pragmatix.flox.model.MoneyType;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   import ru.pragmatix.wormix.settings.PvpWager;
   import ru.pragmatix.wormix.social.Social;
   
   public class PvpCell extends UIContainer
   {
      
      private var btn:IUIButton;
      
      private var wager:PvpWager;
      
      private var icon:UIContainer;
      
      private var availableField:TextField;
      
      public function PvpCell(container:DisplayObjectContainer)
      {
         super(container);
         this.btn = ButtonFactory.createButton(getChildByName("btn"));
         this.btn.addHandler(this.onBtnClick);
         this.btn.setText(Messages.PVP_WAGER_BTN);
         this.icon = new UIContainer(getChildByName("icon") as MovieClip);
         this.icon.removeAllWhenDisabled = true;
         setLabelText("wagerLabel",Messages.PVP_WAGER_LABEL);
         setLabelText("prizeLabel",Messages.PVP_PRIZE_LABEL);
         this.availableField = TextField(getChildByName("available"));
      }
      
      private function onBtnClick(e:Event) : void
      {
         if(UserProfileHelper.isEnoughMoney(MoneyType.INGAME,this.wager))
         {
            this.chooseMode();
         }
      }
      
      // Patch (2v2): ask 1v1 or 2v2. OK is 1v1, the cancel button 2v2; closing the window (X) asks for nothing.
      // The window says "cancel" for both the cancel button and X, so a press on the cancel button is noted.
      private function chooseMode() : void
      {
         var self:PvpCell = this;
         var team:Boolean = false;
         var box:MessageBox = MessageBox.createMsg(text("PVP_MODE_TITLE","Бій на ставку"),text("PVP_MODE_TEXT","Оберіть режим бою: 1 на 1 або 2 на 2."),MessageBox.OK_CANCEL_BTN,function():void
         {
            self.startRequest(false);
         });
         box.setOkBtnText(text("PVP_MODE_1V1","1 на 1"));
         box.setCancelBtnText(text("PVP_MODE_2V2","2 на 2"));
         var cancelClip:DisplayObject = box.getChildByName("cancel");
         if(Boolean(cancelClip))
         {
            cancelClip.addEventListener(MouseEvent.MOUSE_DOWN,function(e:MouseEvent):void
            {
               team = true;
            });
            cancelClip.addEventListener(MouseEvent.ROLL_OUT,function(e:MouseEvent):void
            {
               team = false;
            });
         }
         box.addEventListener(UIEvent.WINDOW_CANCEL,function(e:Event):void
         {
            if(team)
            {
               self.startRequest(true);
            }
         });
         box.show();
      }
      
      // A message from the language file, or the fallback while a browser still has an older copy of the file.
      private static function text(key:String, fallback:String) : String
      {
         var s:String = Messages.getMessage(key);
         return !s || s == key ? fallback : s;
      }
      
      private function startRequest(team:Boolean) : void
      {
         if(Application.isUserBusy())
         {
            return;
         }
         Singletons.arenaController.getArenaMenu().hide();
         Singletons.battleRequestController.wagerRequest(this.wager,team);
      }
      
      public function setData(wager:PvpWager) : void
      {
         this.wager = wager;
         setTextFieldText("wager",wager.ingamePrice);
         setTextFieldText("prize",wager.award);
         if(Boolean(wager.ratingBonus))
         {
            setLabelText("bonusLabel",Messages.PVP_BONUS_LABEL);
            setTextFieldText("bonus",wager.ratingBonus * 100 + "%");
         }
      }
      
      public function update() : void
      {
         var availableByLvl:Boolean = Application.userProfile.getLevel() >= this.wager.minLevel;
         var availableByTeam:Boolean = this.wager != PvpWager.WAGER_3 || Application.userProfile.getTeam().length > 1;
         var available:Boolean = availableByLvl && availableByTeam && Social.settings.isPvpAllowed();
         this.setAvailable(available);
         if(!Social.settings.isPvpAllowed())
         {
            this.availableField.text = Messages.COMING_SOON;
         }
         else if(!availableByLvl)
         {
            this.availableField.text = Strings.format(Messages.PVP_WAGER_AVAILABLE_FOR,{"level":this.wager.minLevel.toString()});
         }
         else if(!availableByTeam)
         {
            this.availableField.text = Messages.PVP_WAGER_AVAILABLE_FOR_TEAM;
         }
      }
      
      private function setAvailable(available:Boolean) : void
      {
         this.btn.setVisible(available);
         this.availableField.visible = !available;
         this.icon.setDisabled(!available);
         getChildByName("wager").alpha = available ? 1 : 0.6;
         getChildByName("prize").alpha = available ? 1 : 0.6;
         getChildByName("wagerLabel").alpha = available ? 1 : 0.6;
         getChildByName("prizeLabel").alpha = available ? 1 : 0.6;
         getChildByName("wagerIcon").alpha = available ? 1 : 0.6;
         getChildByName("prizeIcon").alpha = available ? 1 : 0.6;
         var f:DisplayObject = getChildByName("bonusLabel");
         if(Boolean(f))
         {
            f.alpha = available ? 1 : 0.6;
         }
         f = getChildByName("bonus");
         if(Boolean(f))
         {
            f.alpha = available ? 1 : 0.6;
         }
      }
      
      override public function setDisabled(disabled:Boolean) : void
      {
         super.setDisabled(disabled);
         getContainerAsContainer().mouseChildren = !disabled;
      }
   }
}

