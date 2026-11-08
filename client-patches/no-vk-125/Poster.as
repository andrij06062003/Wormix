package §1Q§
{
   import § !O§.§1#;§;
   import § !O§.§;#]§;
   import § !O§.§`"$§;
   import §!!=§.§+!V§;
   import §""J§.§0#o§;
   import §"#?§.§1!J§;
   import §"#U§.§5!+§;
   import §"#U§.§<q§;
   import §+0§.§ $ §;
   import §+0§.§"$L§;
   import §-v§.§'3§;
   import §-v§.§?"Q§;
   import §0!9§.§>"<§;
   import §0!P§.§1!i§;
   import §1!2§.§?g§;
   import §1W§.§4$T§;
   import §3R§.§8"$§;
   import §4"+§.§@J§;
   import §4#p§.§+!W§;
   import §4F§.§+$!§;
   import §5"U§.§;"M§;
   import §6!O§.§;$#§;
   import §7!`§.*;
   import §9!u§.*;
   import §;"S§.DamageEvent;
   import §="z§.§,#f§;
   import §="z§.Bonus;
   import §?=§.§3$%§;
   import §@#n§.§[q§;
   import §[z§.§6#,§;
   import §^"r§.§ 4§;
   import §^1§.§+!`§;
   import com.greensock.TweenLite;
   import com.greensock.events.LoaderEvent;
   import com.greensock.loading.ImageLoader;
   import com.greensock.loading.LoaderMax;
   import com.greensock.loading.core.LoaderCore;
   import flash.display.DisplayObject;
   import flash.display.MovieClip;
   import flash.display.Sprite;
   import flash.events.*;
   import flash.geom.Point;
   import flash.net.*;
   import flash.text.TextField;
   import flash.utils.*;
   import ru.pragmatix.flox.logger.§[$Q§;
   import ru.pragmatix.flox.social.controller.*;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.messages.clans.request.ListClansRequest;
   import ru.pragmatix.wormix.messages.clans.request.QuitClanRequest;
   import ru.pragmatix.wormix.messages.clans.response.QuitClanResponse;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.physics.§"!-§;
   import ru.pragmatix.wormix.pvp.messages.server.PvpStartTurn;
   import ru.pragmatix.wormix.pvp.messages.server.PvpSystemMessage;
   import ru.pragmatix.wormix.utils.§9!^§;
   import ru.pragmatix.wormix.weapons.*;
   
   public class §##5§
   {
      
      protected var §9$2§:§-#H§;
      
      protected var §@!y§:§9!z§;
      
      protected var §!#%§:§5!f§;
      
      protected var §""X§:String;
      
      protected var log:§[$Q§ = new §[$Q§(§##5§);
      
      protected var context:§1#;§;
      
      public function §##5§(param1:§1#;§, param2:§5y§, param3:String)
      {
         super();
         this.§""X§ = param3;
         this.context = param1;
         // No VK here: no activity posts, wall posts or notifications (each is checked for null).
         this.§9$2§ = null;
         this.§@!y§ = null;
         this.§!#%§ = null;
      }
      
      protected function notify(param1:§<q§, param2:String, param3:Object = null, param4:String = null, param5:Function = null) : void
      {
         if(this.§!#%§)
         {
            this.§!#%§.notify(param1,§;#]§.format(param2,param3),param5,this.§""X§ + param4);
         }
      }
      
      protected function postActivity(param1:String, param2:Object = null, param3:String = null, param4:String = null) : void
      {
         if(this.§9$2§)
         {
            this.§9$2§.§!!C§(§;#]§.format(param1,param2),this.§""X§ + param3,param4);
         }
      }
      
      protected function §8!O§(param1:String, param2:Object = null, param3:String = null, param4:Function = null) : void
      {
         if(this.§@!y§)
         {
            this.§%!=§(null,param1,param2,param3,param4);
         }
      }
      
      private function §%!=§(param1:String, param2:String, param3:Object = null, param4:String = null, param5:Function = null) : void
      {
         var _loc6_:String = null;
         var _loc7_:String = null;
         var _loc8_:Function = null;
         var _loc9_:Object = null;
         var _loc10_:Object = null;
         var _loc11_:LoaderMax = null;
         if(this.§@!y§)
         {
            _loc6_ = this.context.§ "7§() + this.§""X§ + param4;
            _loc7_ = this.context.§4#k§() + this.§""X§ + param4;
            _loc8_ = §`"$§.§+!"§(this.§4#s§,this,[§;#]§.format(param2,param3),param1,param4,param5]);
            _loc9_ = {
               "name":"mainLoad",
               "onProgress":this.§6"7§,
               "onComplete":_loc8_,
               "onError":this.§6l§
            };
            _loc10_ = {
               "name":param4,
               "container":this,
               "alpha":0,
               "scaleMode":"proportionalInside",
               "alternateURL":_loc7_
            };
            _loc11_ = new LoaderMax(_loc9_);
            _loc11_.append(new ImageLoader(_loc6_,_loc10_));
            _loc11_.load();
         }
      }
      
      private function §4#s§(param1:String, param2:String, param3:String, param4:Function = null) : void
      {
         var _loc5_:DisplayObject = LoaderMax.getContent(param3).rawContent;
         this.§^#n§(param1,param2,_loc5_,param4);
         LoaderCore(LoaderMax.getLoader(param3)).dispose(true);
      }
      
      private function §6"7§(param1:LoaderEvent) : void
      {
      }
      
      private function §6l§(param1:LoaderEvent) : void
      {
         this.log.error("error occured with " + param1.target + ": " + param1.text);
      }
      
      protected function §^#n§(param1:String, param2:String, param3:DisplayObject, param4:Function = null) : void
      {
         this.§@!y§.§+#-§(param1,param2,param3,param4);
      }
   }
}

