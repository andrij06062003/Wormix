package §%9§
{
   import § !O§.*;
   import §"#?§.§1!J§;
   import §#C§.§9!5§;
   import §&!Y§.§8"9§;
   import §&#q§.§#=§;
   import §'!§.§39§;
   import §0!9§.§="4§;
   import §0!9§.§>"<§;
   import §0!P§.§1!i§;
   import §0"<§.§<!+§;
   import §1#"§.§%![§;
   import §1#"§.§%!]§;
   import §1#"§.§["$§;
   import §3R§.§0"1§;
   import §4"+§.§@J§;
   import §4"+§.§^§;
   import §4"D§.§ $4§;
   import §4"D§.§];§;
   import §4#p§.§+!W§;
   import §4#p§.§3o§;
   import §4#p§.Team;
   import §4F§.§+$!§;
   import §4F§.§2$=§;
   import §4F§.§8"#§;
   import §4F§.§^"q§;
   import §6$M§.*;
   import §6k§.§%§;
   import §="z§.§%!?§;
   import §>"'§.*;
   import §>#f§.§<"U§;
   import §@#n§.§3!y§;
   import §]" §.§7#§;
   import §]"x§.§'5§;
   import §^"$§.§,!7§;
   import §^1§.§+!`§;
   import §`!g§.§-! §;
   import §`!g§.§^$6§;
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.events.IEventDispatcher;
   import flash.events.MouseEvent;
   import flash.events.TimerEvent;
   import flash.text.TextField;
   import flash.text.TextFieldAutoSize;
   import flash.utils.Timer;
   import ru.pragmatix.flox.gui.controls.§-s§;
   import ru.pragmatix.flox.gui.controls.§0b§;
   import ru.pragmatix.wormix.controller.§%"s§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.game.§7$J§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.window.§&!f§;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.pvp.messages.server.PvpSystemMessage;
   import ru.pragmatix.wormix.social.§#!K§;
   import ru.pragmatix.wormix.utils.§[#;§;
   import ru.pragmatix.wormix.weapons.*;
   
   public class §1!S§ extends §#=§
   {
      
      private var §]?§:§`P§;
      
      private var awardLabel:TextField;
      
      protected var §<§:TextField;
      
      protected var §`#;§:TextField;
      
      private var §]#f§:MovieClip;
      
      private var §!b§:§="4§;
      
      private var back:Array = [];
      
      protected var §<#N§:§ $4§;
      
      private var level:int;
      
      public function §1!S§(param1:§ $4§, param2:int)
      {
         super(§1!J§.§'!h§("AchievLevelUpMessage"));
         this.§<#N§ = param1;
         this.level = param2;
         this.§[!2§(this.§<#N§);
         this.awardLabel = §""F§("awardLabel");
         this.awardLabel.text = " ";
         this.§<§ = §""F§("nameAchiev");
         this.§`#;§ = §""F§("descrAchiev");
         this.§]#f§ = §=!Z§("priceContainer");
         this.§!b§ = §>"<§.§2$1§(getChildByName("shareAchiev_btn"));
         this.§!b§.setText(§^!R§.BTN_SHARE_NAME,§%!]§.§<"9§);
         // Sharing posted to VK, which isn't there.
         getChildByName("shareAchiev_btn").visible = false;
         var _loc3_:int = 0;
         while(_loc3_ < 3)
         {
            this.back.push(§=!Z§("backLevel" + _loc3_));
            this.back[_loc3_].visible = false;
            _loc3_++;
         }
         this.§]!j§(this.§<#N§);
         this.§4$5§(param1.levels[param2]);
         this.back[param2].visible = true;
         y = Application.instance.stageHeight - height - height * 0.7;
      }
      
      protected function §[!2§(param1:§ $4§) : void
      {
         this.§]?§ = new §`P§(§=!Z§("achievIcon"),this.level);
         this.§]?§.setIconLevel(this.level);
         this.§]?§.setAchievIcon(param1.levels[this.level].icon);
      }
      
      protected function §]!j§(param1:§ $4§) : void
      {
         this.§<§.text = §;#]§.format(§^!R§.ACHIEV_LEVELUP_TITLE,{"name":param1.levels[this.level].name});
         this.§`#;§.text = param1.§>$N§(this.level);
      }
      
      protected function §2!T§(param1:MouseEvent = null) : void
      {
         §#!K§.§]1§.§3$7§(this.§<#N§,this.level,this.§]?§.§&#s§());
         this.§8"f§(param1);
      }
      
      private function §4$5§(param1:§];§) : void
      {
         var _loc2_:§0b§ = null;
         var _loc4_:§0b§ = null;
         this.§!#L§(param1);
         this.§=#G§();
         _loc2_ = new §0b§(§=!Z§("priceContainer." + param1.reward.type));
         _loc2_.setVisible(true);
         var _loc3_:TextField = _loc2_.§""F§("value");
         _loc3_.autoSize = TextFieldAutoSize.LEFT;
         _loc3_.text = String(param1.reward.count);
         _loc2_.x = 0;
         §,!7§.§"3§(this.§]#f§,this.§`#;§);
         _loc4_ = new §0b§(§=!Z§("points"));
         _loc4_.setVisible(true);
         var _loc5_:TextField = _loc4_.§""F§("value");
         _loc5_.autoSize = TextFieldAutoSize.LEFT;
         _loc5_.text = String(param1.points);
      }
      
      private function §!#L§(param1:§];§) : void
      {
         if(param1.reward.type == §#$L§.§2!?§.value)
         {
            Application.userProfile.§`8§(param1.reward.count,§0"1§.§%!e§);
         }
         else if(param1.reward.type == §#$L§.RUBY.value)
         {
            Application.userProfile.§`8§(param1.reward.count,§0"1§.§>!M§);
         }
         else if(param1.reward.type == §#$L§.REACTION.value)
         {
            Application.userProfile.§0i§(Application.userProfile.§`a§() + param1.reward.count);
         }
         Application.mainMenu.§-"[§();
      }
      
      private function §=#G§() : void
      {
         var _loc1_:MovieClip = null;
         for each(_loc1_ in this.§]#f§)
         {
            _loc1_.visible = false;
         }
      }
      
      override protected function §?L§() : void
      {
         super.§?L§();
         §8!+§ = true;
         §?"R§ = §-s§.§,$H§;
         §9!#§ = §8"9§.§6"#§;
         §=f§ = true;
      }
      
      override protected function §8"f§(param1:MouseEvent) : void
      {
         this.§!b§.§ Q§(this.§2!T§);
         super.§8"f§(param1);
         dispatchEvent(new Event(Event.CLOSE));
      }
      
      override public function show(param1:Boolean = true) : void
      {
         this.§!b§.§"!O§(this.§2!T§);
         super.show(false);
      }
      
      override public function hide() : void
      {
         super.hide();
      }
   }
}

