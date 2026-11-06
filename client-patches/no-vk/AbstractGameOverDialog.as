package ru.pragmatix.wormix.gui.window.gameover
{
   import flash.display.DisplayObject;
   import flash.display.MovieClip;
   import flash.display.Sprite;
   import flash.events.Event;
   import flash.text.TextField;
   import ru.pragmatix.flox.gui.controls.button.ButtonFactory;
   import ru.pragmatix.flox.gui.controls.button.IUIButton;
   import ru.pragmatix.flox.gui.controls.window.UIWindow;
   import ru.pragmatix.flox.gui.utils.InterfaceUtils;
   import ru.pragmatix.flox.social.controller.SocialType;
   import ru.pragmatix.flox.social.post.IPostToStatus;
   import ru.pragmatix.flox.sound.UISound;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.model.Sounds;
   import ru.pragmatix.wormix.model.battle.BattleResult;
   import ru.pragmatix.wormix.model.battle.BattleResultInfo;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.battle.EndBattleStatus;
   import ru.pragmatix.wormix.objects.sound.SoundFactory;
   import ru.pragmatix.wormix.social.Social;
   
   public class AbstractGameOverDialog extends UIWindow implements IPostToStatus
   {
      
      private var statusPostSelected:Boolean;
      
      public var createNoteSelected:Boolean;
      
      private var createNoteBtn:IUIButton;
      
      private var statusBtn:IUIButton;
      
      private var resultInfo:BattleResultInfo;
      
      private var startInfo:BattleStartInfo;
      
      private var winnerNameField:TextField;
      
      public function AbstractGameOverDialog(container:Sprite, info:BattleStartInfo, result:BattleResultInfo)
      {
         super(container);
         this.startInfo = info;
         this.resultInfo = this.resultInfo;
         this.initTitle(result);
         this.createNoteBtn = ButtonFactory.createButton(getChildByName("createNote"));
         this.createNoteBtn.addHandler(this.onCreateNoteClick);
         this.createNoteBtn.setText(Messages.CREATE_NOTE_BTN);
         this.statusBtn = ButtonFactory.createButton(getChildByName("status"));
         this.statusBtn.addHandler(this.onStatusClick);
         this.statusBtn.setText(Messages.WIN_GAME_SHARE);
         if(Boolean(getChildByNameAsTextField("winnerName")))
         {
            this.winnerNameField = getChildByNameAsTextField("winnerName");
            this.winnerNameField.visible = false;
         }
         // "Create note" and "Share" posted to VK, which isn't there: only the OK button is left (as in hot seat).
         this.createNoteBtn.setVisible(false);
         this.statusBtn.setVisible(false);
         InterfaceUtils.centerXBy(okBtn.getContainer(),panel);
         setSoundOnShow(result.result.equals(BattleResult.WINNER) ? SoundFactory.getSoundScheme(Application.userProfile.getCharType().configName).getWinSound() : Sounds.LOSE);
      }
      
      private function onStatusClick(e:Event) : void
      {
         this.statusPostSelected = true;
         hide();
      }
      
      protected function onCreateNoteClick(e:Event) : void
      {
         this.createNoteSelected = true;
         hide();
      }
      
      override protected function initParams() : void
      {
         super.initParams();
         okButtonText = Messages.BUTTON_OK;
         destroyOnRemove = true;
      }
      
      protected function initTitle(result:BattleResultInfo) : void
      {
         var winnerTitle:MovieClip = getChildByNameAsMovieClip("winner_title");
         var looserTitle:MovieClip = getChildByNameAsMovieClip("loser_title");
         var surrenderTitle:MovieClip = getChildByNameAsMovieClip("op_surrender_title");
         looserTitle.visible = result.result.equals(BattleResult.LOSER);
         surrenderTitle.visible = result.status == EndBattleStatus.OPPONENT_SURRENDER;
         winnerTitle.visible = !looserTitle.visible && !surrenderTitle.visible;
         if(this.startInfo.battleType == BattleType.HOT_SEAT)
         {
            TextField(winnerTitle.text).text = (result.result.equals(BattleResult.DRAW_GAME) ? Messages.DRAW_TITLE : Messages.WINNER_TITLE) + "!";
            TextField(looserTitle.text).text = (result.result.equals(BattleResult.DRAW_GAME) ? Messages.DRAW_TITLE : Messages.WINNER_TITLE) + "!";
            TextField(surrenderTitle.text).text = Messages.OPPONENT_SURRENDER_TITLE + "!";
         }
         else
         {
            TextField(winnerTitle.text).text = (result.result.equals(BattleResult.DRAW_GAME) ? Messages.DRAW_TITLE : Messages.WINNER_TITLE) + "!";
            TextField(looserTitle.text).text = Messages.LOOSER_TITLE + "!";
            TextField(surrenderTitle.text).text = Messages.OPPONENT_SURRENDER_TITLE + "!";
         }
      }
      
      protected function distributeButtonUnder(obj:DisplayObject) : void
      {
         InterfaceUtils.distributeUnderCmp(okBtn,obj,10);
         InterfaceUtils.distributeUnderCmp(this.createNoteBtn,obj,10);
         InterfaceUtils.distributeUnderCmp(this.statusBtn,obj,10);
         MovieClip(panel).height = okBtn.y + okBtn.height + 20;
      }
      
      override public function destroy() : void
      {
         this.createNoteBtn.destroy();
         this.statusBtn.destroy();
         this.startInfo = null;
         super.destroy();
      }
      
      public function getStatusPostSelected() : Boolean
      {
         return this.statusPostSelected;
      }
   }
}

