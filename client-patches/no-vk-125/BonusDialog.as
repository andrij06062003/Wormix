package §"!+§
{
   import § "0§.§!$3§;
   import §!!=§.§+!V§;
   import §!!=§.§[p§;
   import §"#?§.§1!J§;
   import §"#U§.§<q§;
   import §%"D§.§0o§;
   import §%#7§.§]e§;
   import §%L§.§>$I§;
   import §%L§.§`$'§;
   import §&!Y§.§8"9§;
   import §&#q§.§#=§;
   import §'!§.*;
   import §'""§.§2#V§;
   import §+0§.§"$L§;
   import §,!8§.§9k§;
   import §,#$§.§3"9§;
   import §-v§.§ §;
   import §0!9§.§="4§;
   import §0!9§.§>"<§;
   import §0!P§.§1!i§;
   import §1#"§.§%!]§;
   import §1;§.*;
   import §2!g§.§3!w§;
   import §3R§.§&"D§;
   import §4"+§.SelectObjectEvent;
   import §4#p§.§3o§;
   import §5"U§.§;"M§;
   import §6k§.§%§;
   import §8#e§.§3$G§;
   import §8#e§.§]$E§;
   import §9"j§.§9"X§;
   import §="z§.§%!?§;
   import §="z§.§,#f§;
   import §=#5§.§<W§;
   import §>i§.§3c§;
   import §?"Z§.§2$!§;
   import §@#n§.§3!y§;
   import §[$@§.§]§;
   import §[z§.§ t§;
   import §[z§.§6#,§;
   import §^"$§.§,!7§;
   import §^1§.§+!`§;
   import §`!g§.§!"0§;
   import §`!g§.§'!8§;
   import config.achievements.Achievs;
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.errors.IllegalOperationError;
   import flash.events.KeyboardEvent;
   import flash.events.MouseEvent;
   import flash.geom.Point;
   import flash.text.TextField;
   import flash.text.TextFieldAutoSize;
   import flash.utils.ByteArray;
   import flash.utils.Dictionary;
   import flash.utils.IDataOutput;
   import ru.pragmatix.flox.gui.controls.§-s§;
   import ru.pragmatix.flox.gui.controls.§0b§;
   import ru.pragmatix.flox.logger.*;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.controller.§4$D§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.menu.§5#K§;
   import ru.pragmatix.wormix.gui.window.§<!X§;
   import ru.pragmatix.wormix.messages.server.ProfilesResult;
   import ru.pragmatix.wormix.messages.server.RemoveFromGroupResult;
   import ru.pragmatix.wormix.model.rewards.§-#I§;
   import ru.pragmatix.wormix.model.user.§7X§;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.physics.§@$ §;
   import ru.pragmatix.wormix.utils.§]R§;
   import ru.pragmatix.wormix.weapons.*;
   
   public class §>#x§ extends §#=§
   {
      
      protected var §8j§:§0b§;
      
      protected var message:TextField;
      
      protected var §0#W§:Array = [];
      
      protected var bonus:§-#I§;
      
      protected var postBtn:§="4§;
      
      protected var §8!7§:DisplayObjectContainer;
      
      protected var § $,§:int;
      
      protected var §7g§:Function;
      
      public function §>#x§(param1:MovieClip, param2:§-#I§)
      {
         super(param1);
         this.bonus = param2;
         this.init();
         this.§"$E§();
      }
      
      public static function §>#^§(param1:§-#I§, param2:Function = null) : §>#x§
      {
         var _loc3_:MovieClip = null;
         _loc3_ = §1!J§.§'!h§(§;"M§.§!"8§);
         var _loc4_:§>#x§ = new §>#x§(_loc3_,param1);
         _loc4_.§,r§();
         if(param2 != null)
         {
            _loc4_.§7g§ = param2;
         }
         else
         {
            §,!7§.§!#h§(_loc4_.§##<§);
            _loc4_.postBtn.getContainer().parent.visible = false;
         }
         return _loc4_;
      }
      
      public static function §-$,§(param1:int) : §9"X§
      {
         var _loc2_:§3$G§ = null;
         var _loc3_:§9"X§ = null;
         for each(_loc2_ in §]$E§.§@!k§())
         {
            if(_loc2_.prevUpgradeId == param1)
            {
               _loc3_ = Application.userProfile.§]!&§().§@$E§(_loc2_.upgradeId);
               if(_loc3_)
               {
                  return _loc3_;
               }
               §-$,§(_loc2_.upgradeId);
            }
         }
         return null;
      }
      
      protected function init() : void
      {
         this.§8j§ = new §0b§(getChildByName("awardsPanel"));
         this.§ $,§ = this.§8j§.y;
         this.message = §""F§("message");
         this.message.autoSize = TextFieldAutoSize.CENTER;
         this.message.embedFonts = true;
         this.message.wordWrap = true;
         this.message.defaultTextFormat = §%!]§.§<"9§;
         this.message.text = "";
         var _loc1_:DisplayObject = §!"0§.§!"1§(§^p§(),"postBtn");
         this.postBtn = §>"<§.§2$1§(_loc1_);
         this.postBtn.setText(§^!R§.BTN_SHARE_NAME,§%!]§.§<"9§);
         this.postBtn.§"!O§(this.§^"P§);
         this.§8!7§ = this.postBtn.getContainer().parent;
         // Sharing posted to VK, which isn't there.
         this.§8!7§.visible = false;
      }
      
      public function §,r§() : void
      {
         §@#'§(this.bonus.§8"I§);
         this.message.htmlText = "<b>" + this.bonus.§+#;§ + "</b>";
         if(this.message.numLines < 3)
         {
            this.§8j§.y = this.§ $,§ - (3 - this.message.numLines) * 12;
         }
         else
         {
            this.§8j§.y = this.§ $,§;
         }
         this.§!!K§();
         this.§!#n§();
      }
      
      private function §!!K§() : void
      {
         var _loc1_:uint = 0;
         if(this.bonus.reaction)
         {
            _loc1_ = Application.userProfile.§`a§() - this.bonus.reaction;
            Application.userProfile.§0i§(_loc1_);
            Application.mainMenu.§-"[§();
         }
      }
      
      private function §!#n§() : void
      {
         var _loc2_:int = 0;
         var _loc1_:int = int(this.bonus.reagents[§2#V§.CHRISTMAS_KEY]);
         if(_loc1_ != 0)
         {
            _loc2_ = Application.userProfile.§0"^§().§="J§(§2#V§.CHRISTMAS_KEY) - _loc1_;
            Application.mainMenu.§2!l§().§!$0§(_loc2_);
         }
      }
      
      override protected function §?L§() : void
      {
         super.§?L§();
         §?"R§ = §-s§.§,$H§;
         §9!#§ = §8"9§.§6"#§;
         §=f§ = true;
         § 1§(§^!R§.BUTTON_OK);
      }
      
      protected function §"$E§() : void
      {
         var _loc1_:§0b§ = null;
         var _loc2_:§0b§ = null;
         var _loc3_:§0b§ = null;
         var _loc4_:§0b§ = null;
         var _loc5_:String = null;
         var _loc6_:int = 0;
         var _loc7_:Boolean = false;
         var _loc8_:§9k§ = null;
         var _loc9_:§9"X§ = null;
         var _loc10_:uint = 0;
         var _loc11_:§0b§ = null;
         var _loc12_:§5#K§ = null;
         var _loc13_:TextField = null;
         if(this.bonus.money != 0)
         {
            Application.userProfile.§<$#§(Application.userProfile.§]#K§() - this.bonus.money);
            _loc1_ = new §0b§(§1!J§.§'!h§(§,#O§.§"#,§));
            _loc1_.§""F§("count").text = this.bonus.money.toString();
            this.§8j§.addChild(_loc1_);
            this.§0#W§.push(_loc1_);
         }
         if(this.bonus.realMoney != 0)
         {
            Application.userProfile.§="'§(Application.userProfile.§8"3§() - this.bonus.realMoney);
            _loc2_ = new §0b§(§1!J§.§'!h§(§,#O§.§<#x§));
            _loc2_.§""F§("count").text = this.bonus.realMoney.toString();
            this.§8j§.addChild(_loc2_);
            this.§0#W§.push(_loc2_);
         }
         if(this.bonus.battlesCount != 0)
         {
            _loc3_ = new §0b§(§1!J§.§'!h§(§,#O§.§+#J§));
            _loc3_.§""F§("count").text = this.bonus.battlesCount.toString();
            _loc3_.setQtip(§^!R§.BATTLE_QTIP);
            this.§8j§.addChild(_loc3_);
            this.§0#W§.push(_loc3_);
         }
         if(this.bonus.reaction != 0)
         {
            _loc4_ = new §0b§(§1!J§.§'!h§(§,#O§.§+"^§));
            _loc4_.§""F§("count").text = this.bonus.reaction.toString();
            this.§8j§.addChild(_loc4_);
            this.§0#W§.push(_loc4_);
         }
         if(§'!8§.§ !T§(this.bonus.items) != 0)
         {
            for(_loc5_ in this.bonus.items)
            {
               _loc6_ = int(_loc5_);
               _loc7_ = true;
               if(_loc6_ >= 2000)
               {
                  _loc8_ = §2$!§.§+!5§(_loc6_);
               }
               else if(_loc6_ >= 1000)
               {
                  _loc8_ = §2$!§.§""2§(_loc6_,Application.userProfile.getRace());
               }
               else
               {
                  _loc8_ = §2$!§.§-m§(_loc6_);
                  _loc9_ = Application.userProfile.§]!&§().§@$E§(_loc6_);
                  if(!_loc9_)
                  {
                     _loc9_ = §-$,§(_loc6_);
                  }
                  if(Boolean(_loc9_) && _loc9_.isInfinite())
                  {
                     _loc7_ = false;
                  }
               }
               if(_loc7_)
               {
                  _loc10_ = uint(this.bonus.items[_loc5_]);
                  _loc11_ = new §0b§(§1!J§.§'!h§(§,#O§.§&"[§));
                  _loc12_ = new §5#K§(_loc11_.getChildByName("icon"),true);
                  _loc12_.setItem(_loc8_);
                  _loc13_ = _loc11_.§^p§().getChildByName("count") as TextField;
                  _loc13_.text = _loc10_.toString();
                  this.§8j§.addChild(_loc11_);
                  this.§0#W§.push(_loc11_);
               }
            }
         }
         this.§3$,§();
         this.align();
         Application.mainMenu.§-"[§();
      }
      
      protected function align() : void
      {
         var _loc5_:DisplayObject = null;
         var _loc6_:int = 0;
         var _loc7_:int = 0;
         var _loc8_:Boolean = false;
         var _loc9_:* = 0;
         var _loc1_:uint = 4;
         var _loc2_:uint = 57;
         var _loc3_:uint = 0;
         var _loc4_:int = 0;
         while(_loc4_ < this.§0#W§.length)
         {
            _loc3_ = Math.floor(_loc4_ / _loc1_);
            _loc5_ = this.§0#W§[_loc4_].getContainer();
            if(_loc4_ == _loc3_ * _loc1_)
            {
               _loc5_.x = 0;
            }
            else
            {
               §,!7§.§"3§(_loc5_,this.§0#W§[_loc4_ - 1].getContainer(),7);
            }
            _loc5_.y = _loc3_ * _loc2_;
            _loc4_++;
         }
         if(_loc3_ > 0)
         {
            _loc6_ = _loc2_ * _loc3_ + 5;
            §^p§().getChildByName("bgPanel").height = §^p§().getChildByName("bgPanel").height + _loc6_;
            §##<§.y += _loc6_;
            this.§8!7§.y += _loc6_;
            _loc7_ = Math.abs(_loc3_ * _loc1_ - _loc4_);
            _loc8_ = _loc7_ == 0;
            if(!_loc8_)
            {
               _loc9_ = int(this.§0#W§.length - 1);
               while(_loc9_ >= this.§0#W§.length - _loc7_)
               {
                  this.§0#W§[_loc9_].getContainer().x = this.§0#W§[_loc9_].getContainer().x + (_loc1_ - _loc7_) / 2 * this.§0#W§[0].width;
                  _loc9_--;
               }
            }
         }
         §,!7§.§!#h§(this.§8j§);
      }
      
      protected function §3$,§() : void
      {
         var _loc2_:String = null;
         var _loc3_:int = 0;
         var _loc4_:MovieClip = null;
         var _loc5_:§0b§ = null;
         var _loc6_:DisplayObject = null;
         var _loc1_:Dictionary = this.bonus.reagents;
         if(Boolean(_loc1_) && §'!8§.§ !T§(_loc1_) > 0)
         {
            for(_loc2_ in _loc1_)
            {
               _loc3_ = int(_loc2_);
               _loc4_ = §1!J§.§'!h§(§]$E§.§#"z§(_loc3_).getMenuIcon());
               _loc5_ = new §0b§(§1!J§.§'!h§(§,#O§.§+# §));
               this.§8j§.addChild(_loc5_);
               this.§0#W§.push(_loc5_);
               _loc6_ = _loc5_.getChildByName("reagentIcon");
               _loc4_.width = _loc6_.width;
               _loc4_.height = _loc6_.height;
               _loc4_.x = _loc6_.x;
               _loc4_.y = _loc6_.y;
               _loc5_.§!!c§(_loc6_);
               _loc5_.§<"[§(_loc4_);
               _loc5_.§""F§("count").text = _loc1_[_loc2_];
            }
         }
      }
      
      protected function §^"P§(param1:MouseEvent) : void
      {
         if(this.§7g§ != null)
         {
            this.§7g§(this.bonus);
         }
         this.onOkClick(null);
      }
      
      override protected function §8"f§(param1:MouseEvent) : void
      {
         this.onOkClick(null);
         super.§8"f§(param1);
      }
      
      override protected function onOkClick(param1:MouseEvent) : void
      {
         var _loc2_:uint = 0;
         if(this.bonus.reaction)
         {
            _loc2_ = Application.userProfile.§`a§() + this.bonus.reaction;
            Application.userProfile.§0i§(_loc2_);
         }
         if(this.bonus.money)
         {
            Application.userProfile.§<$#§(Application.userProfile.§]#K§() + this.bonus.money);
         }
         if(this.bonus.realMoney)
         {
            Application.userProfile.§="'§(Application.userProfile.§8"3§() + this.bonus.realMoney);
         }
         Application.mainMenu.§-"[§();
         super.onOkClick(param1);
      }
      
      override public function destroy() : void
      {
         super.destroy();
         if(this.§8j§)
         {
            this.§8j§.destroy();
         }
      }
   }
}

