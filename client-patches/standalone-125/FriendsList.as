package §9"`§
{
   import flash.external.ExternalInterface;
   import § !O§.§2J§;
   import § !O§.§;#]§;
   import § S§.§,!A§;
   import §!!6§.§^![§;
   import §!!=§.*;
   import §"!+§.*;
   import §"#?§.§1!J§;
   import §%#l§.*;
   import §%9§.§!"b§;
   import §%9§.§,#j§;
   import §%9§.§-$E§;
   import §%9§.§2#J§;
   import §%9§.§>!v§;
   import §%9§.AchievTabPage;
   import §'!§.*;
   import §'"4§.§9!o§;
   import §+0§.§ $ §;
   import §,!8§.§!"z§;
   import §,!8§.§1I§;
   import §,!8§.§9k§;
   import §0!9§.§#"h§;
   import §0!9§.§>"<§;
   import §0!P§.§1!i§;
   import §1#"§.§%!]§;
   import §1;§.*;
   import §1V§.*;
   import §1W§.§^$"§;
   import §4!,§.*;
   import §4"+§.§#]§;
   import §4"D§.§ $4§;
   import §4"D§.§!!S§;
   import §4#p§.§3o§;
   import §5"U§.§;"M§;
   import §6!O§.§3"e§;
   import §6!y§.*;
   import §6k§.§%§;
   import §7#F§.§8#P§;
   import §7I§.§8[§;
   import §7L§.§@$I§;
   import §8#e§.§]$E§;
   import §9"j§.§1[§;
   import §9"j§.§9"X§;
   import §;"I§.§^!o§;
   import §=#A§.*;
   import §?"Z§.§2$!§;
   import §?#`§.§3!p§;
   import §?=§.§3$%§;
   import §?=§.§?#+§;
   import §@#n§.§3!y§;
   import §["y§.§9!k§;
   import §[z§.§6#,§;
   import §]"x§.*;
   import §]`§.§ !S§;
   import §^"$§.§,!7§;
   import §^$M§.§6]§;
   import §^1§.§+!`§;
   import §`!f§.§3!h§;
   import §`!g§.§="w§;
   import config.achievements.§3$Q§;
   import config.achievements.§7#s§;
   import flash.display.BitmapData;
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.display.SimpleButton;
   import flash.errors.IllegalOperationError;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.events.SecurityErrorEvent;
   import flash.external.ExternalInterface;
   import flash.geom.Point;
   import flash.geom.Rectangle;
   import flash.text.TextField;
   import flash.text.TextFormat;
   import flash.ui.Keyboard;
   import org.gestouch.core.*;
   import ru.pragmatix.flox.gui.controls.§%]§;
   import ru.pragmatix.flox.gui.controls.§>Q§;
   import ru.pragmatix.flox.logger.*;
   import ru.pragmatix.wormix.controller.§%"s§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.controller.§4$D§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.menu.arena.boss.§6`§;
   import ru.pragmatix.wormix.messages.clans.structures.ClanTO;
   import ru.pragmatix.wormix.messages.client.IncreaseAchievements;
   import ru.pragmatix.wormix.messages.server.IncreaseAchievementResult;
   import ru.pragmatix.wormix.messages.server.WipeProfileResult;
   import ru.pragmatix.wormix.model.items.crafted.CraftedEquipFamily;
   import ru.pragmatix.wormix.model.rewards.§-#I§;
   import ru.pragmatix.wormix.model.user.§7X§;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.pvp.messages.ex.*;
   import ru.pragmatix.wormix.social.§#!K§;
   import ru.pragmatix.wormix.utils.§<!f§;
   import ru.pragmatix.wormix.utils.§[#;§;
   import ru.pragmatix.wormix.weapons.*;
   
   use namespace gestouch_internal;
   
   public class §;!t§ extends §?#+§
   {
      
      private var §&$M§:FriendsControl;
      
      public function §;!t§(param1:MovieClip)
      {
         super(param1);
         hideGroupButton(param1);
         §-"$§([]);
         this.§,!f§ = new §1"-§(§6$1§.numChildren);
         this.§1!@§();
         §=",§(1);
         update();
      }
      
      // Patch (standalone-125): "+" in an empty cell invited VK friends. It now asks the page to share the game's
      // address and the player's id (wwwroot/play.html, window.wormixInvite).
      private static function §,#?§(param1:MouseEvent) : void
      {
         try
         {
            if(ExternalInterface.available)
            {
               ExternalInterface.call("wormixInvite");
            }
         }
         catch(e:Error)
         {
         }
      }

      // Patch (standalone-125): the main menu's VK group button ("В", next to the news). The main menu itself can't
      // be recompiled (FFDec can't resolve its obfuscated super calls); it builds this list right after the
      // button, so the list hides it, and moves the news button into its place as the menu does without a group.
      private static function hideGroupButton(list:DisplayObject) : void
      {
         var menu:DisplayObjectContainer = Boolean(list) ? list.parent : null;
         var group:DisplayObject = Boolean(menu) ? menu.getChildByName("groupBtnContainer") : null;
         var news:DisplayObject = Boolean(menu) ? menu.getChildByName("newsBtn") : null;
         if(Boolean(group) && group.visible)
         {
            group.visible = false;
            if(news)
            {
               news.x = group.x - 20;
            }
         }
      }
      
      override protected function §?L§() : void
      {
         §"$?§ = true;
         super.§?L§();
      }
      
      override public function setData(param1:Array) : void
      {
         throw new IllegalOperationError();
      }
      
      private function §1!@§() : void
      {
         var _loc3_:DisplayObject = null;
         this.§&$M§ = new FriendsControl();
         addChildAt(this.§&$M§,0);
         var _loc1_:int = 0;
         var _loc2_:int = §^p§().numChildren;
         while(_loc1_ < _loc2_)
         {
            _loc3_ = §^p§().getChildAt(_loc1_);
            if(_loc3_ is SimpleButton && _loc3_.name != "next" && _loc3_.name != "prev")
            {
               _loc3_.addEventListener(MouseEvent.CLICK,§,#?§);
            }
            _loc1_++;
         }
      }
      
      override protected function §@#3§(param1:§3$%§, param2:Object, param3:uint) : void
      {
         var _loc5_:uint = 0;
         var _loc4_:UserProfile = param2 as UserProfile;
         if(_loc4_)
         {
            _loc5_ = (§,!f§.§<!a§() - 1) * §,!f§.§6"E§() + param1.getCellIndex();
            _loc4_.§5!Q§(_loc5_ < §%"s§.§]$<§());
            _loc4_.§;#?§(§3o§.FRIEND);
         }
         super.§@#3§(param1,param2,param3);
      }
      
      override protected function §=u§(param1:DisplayObject) : §3$%§
      {
         var _loc2_:MovieClip = param1 as MovieClip;
         §%!+§(_loc2_.getChildByName("addBtn"),MouseEvent.CLICK,§,#?§);
         return new §8%§(MovieClip(param1));
      }
   }
}

