package §+!U§
{
   import § !O§.§;#]§;
   import §"#?§.§1!J§;
   import §%#l§.§&"M§;
   import §%9§.§#$L§;
   import §&#q§.§#=§;
   import §+0§.§ $ §;
   import §+0§.§9J§;
   import §+0§.§?"T§;
   import §,!8§.§-!H§;
   import §,!8§.§1I§;
   import §,!8§.§9k§;
   import §,"z§.*;
   import §-v§.§<""§;
   import §0!9§.§="4§;
   import §0!9§.§>"<§;
   import §1#"§.§%!]§;
   import §1#"§.§<"3§;
   import §4"D§.§5$;§;
   import §4"D§.§];§;
   import §5!`§.§ ##§;
   import §5!`§.CheatDetectedEvent;
   import §5"U§.§;"M§;
   import §7!o§.Race;
   import §9$"§.*;
   import §=p§.§4#z§;
   import §>"n§.*;
   import §>#f§.CollisionEvent;
   import §?"Z§.§2$!§;
   import §@#n§.§3!y§;
   import §[#M§.§3#o§;
   import §[#M§.§]§;
   import §[$@§.§;u§;
   import §[z§.§6#,§;
   import §]`§.§ !S§;
   import §^1§.§+!`§;
   import §`!g§.§!"0§;
   import config.achievements.Achievs;
   import flash.display.DisplayObject;
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.geom.Point;
   import flash.text.TextField;
   import flash.text.TextFieldAutoSize;
   import flash.utils.ByteArray;
   import flash.utils.IDataInput;
   import ru.pragmatix.flox.gui.controls.§-s§;
   import ru.pragmatix.flox.gui.controls.§0b§;
   import ru.pragmatix.flox.gui.controls.§?"W§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.messages.server.IncomingPayment;
   import ru.pragmatix.wormix.model.user.§7X§;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.physics.§@$ §;
   
   public class LevelUpDialog extends §#=§
   {
      
      public var §9Y§:Boolean;
      
      public var §""u§:Boolean;
      
      private var §9U§:§="4§;
      
      private var shopBtn:§="4§;
      
      private var §0!Z§:§="4§;
      
      private var §;!n§:§="4§;
      
      private var §6$§:§="4§;
      
      private var userProfile:UserProfile;
      
      private var §'#m§:§&!?§;
      
      public var resultInfo:§<""§;
      
      private var rewardPanel:§>G§;
      
      private const §+"U§:Number = 426;
      
      public function LevelUpDialog(param1:§<""§, param2:Array)
      {
         var _loc5_:Array = null;
         var _loc6_:Race = null;
         var _loc7_:§9k§ = null;
         super(§1!J§.§'!h§(§;"M§.§="x§));
         this.resultInfo = param1;
         this.userProfile = Application.userProfile;
         var _loc3_:DisplayObject = §!"0§.§!"1§(§^p§(),"postBtn");
         this.§9U§ = §>"<§.§2$1§(_loc3_);
         this.§9U§.§"!O§(this.§,!1§);
         this.§9U§.setText(§^!R§.WIN_GAME_SHARE,§%!]§.§<"9§);
         // Sharing posted to VK, which isn't there.
         _loc3_.visible = false;
         this.shopBtn = §>"<§.§2$1§(getChildByName("shopBtn"));
         this.shopBtn.§"!O§(this.§6!h§);
         this.shopBtn.setText(§^!R§.LEVEL_UP_DIALOG_BUTTON_SHOP,§%!]§.§<"9§);
         this.§0!Z§ = §>"<§.§2$1§((getChildByName("hpInfo") as MovieClip).getChildByName("btn"));
         this.§0!Z§.setQtip(§^!R§.LEVEL_UP_DIALOG_LIFE_TIPS);
         this.§;!n§ = §>"<§.§2$1§((getChildByName("paramsInfo") as MovieClip).getChildByName("btn"));
         this.§;!n§.setQtip(§^!R§.LEVEL_UP_DIALOG_PARAMS_TIPS);
         this.§6$§ = §>"<§.§2$1§((getChildByName("commandInfo") as MovieClip).getChildByName("btn"));
         this.§6$§.setQtip(§^!R§.LEVEL_UP_DIALOG_COMMAND_TIPS);
         var _loc4_:§="4§ = §>"<§.§2$1§((getChildByName("fuzyInfo") as MovieClip).getChildByName("btn"));
         _loc4_.setQtip(§^!R§.COIN_QTIP);
         this.§'#m§ = new §&!?§(getChildByName("artsItemsContainer") as MovieClip,"x");
         if(!§2$!§.§4!q§(Application.userProfile))
         {
            this.§'#m§.setVisible(false);
         }
         else
         {
            _loc5_ = [];
            for each(_loc6_ in Race.§>!?§())
            {
               if(_loc6_.§6#`§().§4$M§() == Application.userProfile.getLevel())
               {
                  _loc5_.push(§;u§.§2u§()[_loc6_.§6#`§().getId()]);
               }
            }
            for each(_loc7_ in §2$!§.§4!q§(Application.userProfile))
            {
               if(_loc7_ is §1I§)
               {
                  _loc5_[_loc5_.length] = new §]#4§(_loc7_.getId(),0);
               }
               else if(_loc7_ is §-!H§)
               {
                  _loc5_[_loc5_.length] = new §3#o§(_loc7_.getId());
               }
            }
            this.§'#m§.§"W§(_loc5_);
         }
         this.rewardPanel = new §>G§(getChildByName("rewardItemsContainer") as MovieClip);
         if(param2.length == 0)
         {
            this.rewardPanel.setVisible(false);
            (getChildByName("dummy") as MovieClip).visible = true;
         }
         else
         {
            (getChildByName("dummy") as MovieClip).visible = false;
            this.rewardPanel.§"W§(param2);
         }
         §?"R§ = §-s§.§9!9§;
         this.§##O§();
      }
      
      private function §,!1§(param1:Event) : void
      {
         this.§9Y§ = true;
         this.hide();
      }
      
      private function §6!h§(param1:Event) : void
      {
         this.§""u§ = true;
         this.hide();
      }
      
      override protected function §?L§() : void
      {
         super.§?L§();
         §&#g§ = §^!R§.BUTTON_OK;
         §=f§ = true;
      }
      
      private function §##O§() : void
      {
         var _loc1_:uint = 0;
         var _loc2_:uint = 0;
         var _loc3_:uint = 0;
         _loc1_ = uint(Application.userProfile.getLevel());
         (getChildByName("title") as TextField).text = §;#]§.format(§^!R§.LEVEL_UP_DIALOG_TITLE,{"level":_loc1_});
         (getChildByName("description") as TextField).text = §^!R§.LEVEL_UP_DIALOG_DESCRIPTION;
         (getChildByName("bonusDescription") as TextField).text = §^!R§.LEVEL_UP_DIALOG_BONUS_TEXT;
         (getChildByName("newThingsDescription") as TextField).text = §^!R§.LEVEL_UP_DIALOG_THINGS_TEXT;
         _loc2_ = §<"3§.§4!Y§(_loc1_);
         _loc3_ = §<"3§.§4!Y§(_loc1_ - 1);
         var _loc4_:uint = _loc2_ - _loc3_;
         var _loc5_:Array = [];
         _loc5_.push(this.§9a§("fuzyInfo","+" + this.resultInfo.§!#8§));
         _loc5_.push(this.§9a§("hpInfo","+" + String(_loc4_)));
         _loc5_.push(this.§9a§("paramsInfo","+" + §7X§.§4!H§(Application.userProfile)));
         var _loc6_:uint = §<"3§.§]#l§(_loc1_) - §<"3§.§]#l§(_loc1_ - 1);
         if(_loc6_ > 0)
         {
            _loc5_.push(this.§9a§("commandInfo","+" + String(_loc6_)));
         }
         else
         {
            (getChildByName("commandInfo") as MovieClip).visible = false;
         }
         this.§]!M§(_loc5_);
      }
      
      private function §9a§(param1:String, param2:String) : MovieClip
      {
         ((getChildByName(param1) as MovieClip).getChildByName("text") as TextField).text = param2;
         ((getChildByName(param1) as MovieClip).getChildByName("text") as TextField).autoSize = TextFieldAutoSize.LEFT;
         return getChildByName(param1) as MovieClip;
      }
      
      private function §]!M§(param1:Array) : void
      {
         var _loc3_:MovieClip = null;
         var _loc4_:Number = NaN;
         var _loc5_:uint = 0;
         var _loc6_:uint = 0;
         var _loc2_:Number = 0;
         for each(_loc3_ in param1)
         {
            _loc2_ += _loc3_.width;
         }
         _loc4_ = this.§+"U§ * 0.5 - _loc2_ * 0.5;
         _loc5_ = param1.length;
         _loc6_ = 0;
         while(_loc6_ < _loc5_)
         {
            if(_loc6_ != 0)
            {
               param1[_loc6_].x = param1[_loc6_ - 1].x + param1[_loc6_ - 1].width + 10;
            }
            else
            {
               param1[_loc6_].x = _loc4_;
            }
            _loc6_++;
         }
      }
      
      override public function show(param1:Boolean = true) : void
      {
         if(§-"d§.§%"r§.§;#!§())
         {
            §-"d§.§6$$§.§%$<§(Application.mainMenu.mainMenu.§ #B§());
         }
         super.show(param1);
      }
      
      override public function destroy() : void
      {
         this.rewardPanel.destroy();
         this.§'#m§.destroy();
         this.§9U§.destroy();
         this.shopBtn.destroy();
         super.destroy();
      }
      
      override public function hide() : void
      {
         super.hide();
         Application.mainMenu.mainMenu.§?"P§.§",§();
      }
   }
}

