package §=#A§
{
   import §!"5§.§;#%§;
   import §"#?§.§1!J§;
   import §#!s§.§6$A§;
   import §&#q§.§#=§;
   import §+!l§.*;
   import §,!8§.§""A§;
   import §,!8§.§-!H§;
   import §,!8§.§0#v§;
   import §,!8§.§9k§;
   import §0!9§.§="4§;
   import §0!9§.§>"<§;
   import §0#1§.§7"D§;
   import §1#"§.§%!]§;
   import §1#"§.§>$9§;
   import §1W§.§["[§;
   import §3R§.§0"1§;
   import §4"D§.§!!S§;
   import §5"U§.§;"M§;
   import §6!O§.§3"e§;
   import §6k§.§%§;
   import §7!o§.Race;
   import §;"I§.*;
   import §="z§.§,#f§;
   import §="z§.Bonus;
   import §?"Z§.§2$!§;
   import §["y§.§9!k§;
   import §`!g§.§ $#§;
   import §`!g§.§!"0§;
   import §`!g§.§-! §;
   import config.achievements.*;
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.errors.IllegalOperationError;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.text.TextField;
   import flash.utils.Dictionary;
   import ru.pragmatix.flox.social.controller.§%$M§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.menu.§5#K§;
   import ru.pragmatix.wormix.messages.structures.ShopItemStructure;
   import ru.pragmatix.wormix.model.items.crafted.CraftedEquipFamily;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.social.§#!K§;
   
   public class CraftEquipCompleteDialog extends §#=§
   {
      
      private static const §%x§:String = "#58FF00";
      
      private var §%#P§:§9k§;
      
      public function CraftEquipCompleteDialog(param1:CraftedEquipFamily = null, param2:int = -1)
      {
         var _loc3_:DisplayObject = null;
         super(§1!J§.§'!h§(§;"M§.§8#=§));
         _loc3_ = §!"0§.§!"1§(§^p§(),"postBtn",§!"0§.§-b§);
         var _loc4_:§="4§ = §>"<§.§2$1§(_loc3_);
         _loc4_.§"!O§(this.§&!&§);
         _loc4_.setText(§^!R§.BTN_SHARE_NAME,§%!]§.§<"9§);
         // Sharing posted to VK, which isn't there.
         _loc3_.visible = false;
         this.init(param1,param2);
      }
      
      private static function §'$0§(param1:Dictionary, param2:Dictionary) : String
      {
         var _loc4_:§,#f§ = null;
         var _loc5_:Bonus = null;
         var _loc3_:String = "";
         for each(_loc4_ in param1)
         {
            _loc5_ = _loc4_.§,#<§();
            if(_loc4_.§6y§())
            {
               _loc3_ += _loc4_.§"!0§() + ": " + _loc4_.getFormattedValue() + " " + §@!A§(param2,_loc5_) + "\n";
            }
            else if(!param2[_loc5_.getName()])
            {
               _loc3_ += _loc4_.§"!0§() + "\n";
            }
         }
         return _loc3_;
      }
      
      private static function §6"Y§(param1:Dictionary, param2:Dictionary) : String
      {
         var _loc4_:§,#f§ = null;
         var _loc5_:int = 0;
         var _loc6_:§,#f§ = null;
         var _loc3_:String = "";
         for each(_loc4_ in param2)
         {
            if(_loc4_.§6y§())
            {
               _loc3_ += §-! §.§2$S§(_loc4_.§"!0§() + ": " + _loc4_.getFormattedValue() + "\n",§%x§);
            }
            else
            {
               _loc5_ = _loc4_.getValue();
               _loc6_ = param1[_loc4_.§,#<§().getName()] as §,#f§;
               if(_loc6_)
               {
                  _loc5_ += _loc6_.getValue();
               }
               _loc3_ += §-! §.§2$S§(_loc4_.§"!0§(_loc5_) + "\n",§%x§);
            }
         }
         return _loc3_;
      }
      
      public static function §<$0§(param1:§;#%§, param2:TextField) : void
      {
         param2.textColor = uint("0x" + param1.§,#<§().§"C§().slice(1));
         §3"e§.§,#P§(param2,§%!]§.§<"9§,param1.§,]§());
      }
      
      private static function §@!A§(param1:Dictionary, param2:Bonus) : String
      {
         var _loc3_:String = "";
         if(param1[param2.getName()])
         {
            _loc3_ = §,#f§(param1[param2.getName()]).getFormattedValue();
            if(_loc3_.charAt(0) != "+")
            {
               _loc3_ = "+" + _loc3_;
            }
            delete param1[param2.getName()];
         }
         return §-! §.§2$S§(_loc3_,§%x§);
      }
      
      public function init(param1:CraftedEquipFamily, param2:int) : void
      {
         this.§%#P§ = §2$!§.§@$E§(param2,§9!k§.getProfile().getRace()) as §-!H§;
         §##<§.setText(§7"D§.§0!M§("CRAFT_COMPLETE_POPUP_BTN"),§%!]§.§<"9§);
         var _loc3_:§5#K§ = new §5#K§(§=!Z§("itemIconContainer").getChildByName("icon"),true);
         _loc3_.setItem(this.§%#P§);
         var _loc4_:Dictionary = this.§5"[§(param1);
         var _loc5_:Dictionary = this.§6#=§(param1,param2);
         var _temp_1:* = §'$0§(_loc4_,_loc5_);
         var _loc6_:String = §'$0§(_loc4_,_loc5_);
         _loc6_ = _loc6_ + §6"Y§(_loc4_,_loc5_);
         §""F§("description").htmlText = "<b>" + _loc6_ + "</b>";
      }
      
      private function §5"[§(param1:CraftedEquipFamily) : Dictionary
      {
         var _loc3_:§,#f§ = null;
         var _loc2_:Dictionary = new Dictionary();
         for each(_loc3_ in param1.§;"r§().§5t§())
         {
            _loc2_[_loc3_.§,#<§().getName()] = _loc3_;
         }
         return _loc2_;
      }
      
      private function §6#=§(param1:CraftedEquipFamily, param2:int) : Dictionary
      {
         var _loc4_:§;#%§ = null;
         var _loc5_:Vector.<§,#f§> = null;
         var _loc6_:§,#f§ = null;
         var _loc3_:Dictionary = new Dictionary();
         for each(_loc4_ in param1.§;F§())
         {
            if(_loc4_.§;5§() == param2)
            {
               §<$0§(_loc4_,§""F§("title"));
               _loc5_ = _loc4_.§0#D§().§5t§();
               for each(_loc6_ in _loc5_)
               {
                  _loc3_[_loc6_.§,#<§().getName()] = _loc6_;
               }
               return _loc3_;
            }
         }
         return _loc3_;
      }
      
      private function §&!&§(param1:Event) : void
      {
         §#!K§.§]1§.§;"V§(this.§%#P§);
         hide();
      }
   }
}

