package §9"`§
{
   import flash.external.ExternalInterface;
   import § !O§.§`"$§;
   import § #`§.§1#n§;
   import §""J§.§0#o§;
   import §""L§.*;
   import §"#?§.§1!J§;
   import §#C§.§]"'§;
   import §%#]§.§#"m§;
   import §%#]§.§>!V§;
   import §%#l§.§&"M§;
   import §%#l§.§7!y§;
   import §%^§.§9§;
   import §&!Y§.§8"9§;
   import §+$I§.§&$2§;
   import §+0§.§ $ §;
   import §,!8§.§!"z§;
   import §,!8§.§""A§;
   import §,!8§.§1I§;
   import §,!8§.§9k§;
   import §,$5§.*;
   import §-#§.§]$I§;
   import §0!9§.§>"<§;
   import §1#"§.§%!]§;
   import §1#"§.§9#w§;
   import §1#"§.§>$9§;
   import §1b§.§;!c§;
   import §3#G§.§4Q§;
   import §3R§.§0"1§;
   import §3R§.§8"$§;
   import §4"+§.§@J§;
   import §4"+§.§^$3§;
   import §4#p§.§3o§;
   import §5!`§.§ ##§;
   import §5!`§.CheatDetectedEvent;
   import §5"U§.§;"M§;
   import §6!O§.§2!-§;
   import §6!O§.§3"e§;
   import §7I§.§+#?§;
   import §7L§.§@$I§;
   import §7T§.*;
   import §9!u§.*;
   import §;"S§.NextTurnEvent;
   import §;7§.§+$G§;
   import §<$=§.*;
   import §="z§.§,#f§;
   import §=p§.§4#z§;
   import §>#f§.CollisionEvent;
   import §>i§.§3c§;
   import §?=§.§3$%§;
   import §?=§.§>$T§;
   import §?=§.§?#+§;
   import §@!9§.§^"E§;
   import §[#M§.§1$K§;
   import §[#M§.§6#q§;
   import §[#M§.§^#t§;
   import §[z§.§6#,§;
   import §^"r§.§ 4§;
   import §^"r§.§<"i§;
   import §^#c§.§5N§;
   import §^1§.§+!`§;
   import §`!g§.*;
   import com.greensock.*;
   import com.greensock.core.*;
   import com.greensock.plugins.*;
   import config.achievements.*;
   import flash.display.*;
   import flash.events.*;
   import flash.external.ExternalInterface;
   import flash.geom.Point;
   import flash.geom.Rectangle;
   import flash.net.Socket;
   import flash.text.StyleSheet;
   import flash.text.TextField;
   import flash.utils.*;
   import ru.pragmatix.flox.gui.controls.§0b§;
   import ru.pragmatix.flox.gui.controls.§?"W§;
   import ru.pragmatix.flox.serialization.§;"k§;
   import ru.pragmatix.flox.social.controller.*;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.controller.§?$M§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.menu.*;
   import ru.pragmatix.wormix.gui.window.§<!X§;
   import ru.pragmatix.wormix.messages.clans.main.IncomingClanMessage;
   import ru.pragmatix.wormix.messages.clans.response.*;
   import ru.pragmatix.wormix.messages.clans.structures.ClanChatMessageTO;
   import ru.pragmatix.wormix.messages.clans.structures.ClanNewsTO;
   import ru.pragmatix.wormix.model.rewards.§3H§;
   import ru.pragmatix.wormix.model.user.§7X§;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.physics.§"!-§;
   import ru.pragmatix.wormix.physics.§&"R§;
   import ru.pragmatix.wormix.physics.§@$ §;
   import ru.pragmatix.wormix.social.§#!K§;
   import ru.pragmatix.wormix.utils.§5!#§;
   import ru.pragmatix.wormix.utils.§[#;§;
   
   public class §?#G§ extends §?#+§
   {
      // FFDec compiles calls of IUIComponent's methods (setDisabled, removeAllWhenDisabled) as public names, which
      // don't find them: name the interface's namespace, as the original code does.
      namespace iuc = "ru.pragmatix.flox.gui.controls:IUIComponent";
      
      private var oneFriendRewardPanel:§0$ §;
      
      private var allFriendComebackRewardPanel:§0$ §;
      
      public function §?#G§()
      {
         super(§1!J§.§'!h§(§;"M§.§^;§));
         §3"e§.§,#P§(container["title"],§%!]§.§<"9§,§^!R§.RETURN_FRIENDS_MENU_TITLE);
         container["msg"]["text"].text = §^!R§.RETURN_FRIENDS_MENU_MSG;
         container["msg"].visible = false;
         §##<§.setText(§^!R§.RETURN_FRIENDS_MENU_BTN,§%!]§.§<"9§);
         prev.iuc::removeAllWhenDisabled = true;
         next.iuc::removeAllWhenDisabled = true;
         §3"e§.§,#P§(container["descr1"],§%!]§.§<"9§,§^!R§.RETURN_FRIENDS_MENU_DESCR_1);
         §3"e§.§,#P§(container["descr2"],§%!]§.§<"9§,§^!R§.RETURN_FRIENDS_MENU_DESCR_2);
         container["label1"].text = §^!R§.RETURN_FRIENDS_MENU_LABEL_1;
         container["label2"].text = §^!R§.RETURN_FRIENDS_MENU_LABEL_2;
         §3"e§.§,#P§(container["oneFriendCount"],§%!]§.§<"9§,"x" + 1);
         §3"e§.§,#P§(container["allFriendsCount"],§%!]§.§<"9§,"x" + 0);
         this.oneFriendRewardPanel = new §0$ §(container["oneFriendRewardPanel"]);
         this.oneFriendRewardPanel.§&$7§(§?$M§.§?h§);
         this.allFriendComebackRewardPanel = new §0$ §(container["allFriendComebackRewardPanel"]);
         this.allFriendComebackRewardPanel.§&$7§(new §3H§([]));
      }
      
      override protected function §?L§() : void
      {
         super.§?L§();
         §!##§ = false;
         §=f§ = false;
         §]"_§ = §^!R§.BUTTON_CLOSE;
         §-!e§ = true;
         §4$4§ = true;
      }
      
      override protected function §!#P§() : void
      {
         var _loc1_:UserProfile = null;
         if(§7"H§().length)
         {
            _loc1_ = UserProfile(§7"H§()[0]);
            §#!K§.§]1§.§^!Q§(_loc1_,this.§6"R§);
         }
      }
      
      private function §6"R§(param1:§ 4§) : void
      {
         var _loc2_:UserProfile = null;
         if(param1.success)
         {
            this.showLoading(true);
            _loc2_ = UserProfile(§7"H§()[0]);
            §-"d§.§'"%§.§0"D§(_loc2_,this.§&"j§);
         }
         else
         {
            §?"W§.§5"A§(§2!-§.TITLE_ERROR,§9#0§.§,!v§(param1));
         }
      }
      
      private function §&"j§() : void
      {
         this.showLoading(false);
         this.update();
      }
      
      // Patch (standalone-125): "Верни бойцов" sent VK invitations to friends who left the game. Its button (the
      // bird with a letter, next to the friends list) now opens the game's chat (wwwroot/play.html, window.wormixChat).
      override public function show(param1:Boolean = true) : void
      {
         try
         {
            if(ExternalInterface.available)
            {
               ExternalInterface.call("wormixChat");
               return;
            }
         }
         catch(e:Error)
         {
         }
         this.showLoading(true);
         if(Boolean(§,!f§) && !§,!f§.§ #t§().length == 0)
         {
            this.showLoading(false);
         }
         else
         {
            §-"d§.§'"%§.§6!K§(this.§<#+§);
         }
         super.show(param1);
      }
      
      override public function showLoading(param1:Boolean) : void
      {
         super.showLoading(param1);
         §6$1§.visible = !param1;
      }
      
      private function §<#+§(param1:Array, param2:uint) : void
      {
         var _loc3_:§3H§ = null;
         §,!f§ = new §1#7§(param1,§6$1§.numChildren,param2);
         if(param2 > 0)
         {
            _loc3_ = new §3H§([new §^#t§(§0"1§.§%!e§,§?$M§.§?h§.§]#K§() * param2),new §^#t§(§0"1§.§>!M§,§?$M§.§?h§.§8"3§() * param2),new §6#q§(§?$M§.§?h§.§`a§() * param2),new §1$K§(0,§?$M§.§?h§.§##p§() * param2)]);
            this.allFriendComebackRewardPanel.§&$7§(_loc3_);
            §3"e§.§,#P§(container["allFriendsCount"],§%!]§.§<"9§,"x" + param2);
         }
         this.update();
         this.showLoading(false);
         container["msg"].visible = param2 <= 0;
         §##<§.iuc::setDisabled(param2 <= 0);
      }
      
      override public function update() : void
      {
         this.showLoading(true);
         super.update();
      }
      
      override protected function §-"$§(param1:Array) : void
      {
         super.§-"$§(param1);
         this.showLoading(false);
      }
      
      override public function setData(param1:Array) : void
      {
      }
      
      override protected function §=u§(param1:DisplayObject) : §3$%§
      {
         §3"e§.§,#P§(param1["disabled"]["text"],§%!]§.§<"9§,§^!R§.RETURN_FRIENDS_MENU_DISABLE_TEXT_BTN);
         return new §!m§(param1 as MovieClip,null,false,false);
      }
      
      override protected function §@#3§(param1:§3$%§, param2:Object, param3:uint) : void
      {
         var _loc5_:String = null;
         var _loc4_:UserProfile = UserProfile(param2);
         if(_loc4_)
         {
            param1.iuc::setDisabled(§-"d§.§'"%§.§7#@§(_loc4_));
            _loc5_ = §7X§.§4$C§(_loc4_);
            §0b§(param1).setQtipPanel(new §4Q§(_loc5_),§@$I§.§ #3§);
            §0b§(param1).setQtip(_loc5_);
         }
         super.§@#3§(param1,param2,param3);
      }
   }
}

