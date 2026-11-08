package §7L§
{
   import § !O§.§1#;§;
   import § !O§.§;#]§;
   import § "n§.§1"v§;
   import § S§.§,!A§;
   import §!"p§.BankMenu;
   import §""L§.*;
   import §"#?§.§1!J§;
   import §"#U§.§2$5§;
   import §"#U§.§[!n§;
   import §#C§.AtomBomb;
   import §%"D§.§0o§;
   import §%#7§.*;
   import §'!§.§<"X§;
   import §+0§.§9J§;
   import §+0§.§?"T§;
   import §,!8§.§1I§;
   import §0!9§.§="4§;
   import §0!9§.§>"<§;
   import §0!P§.§1!i§;
   import §0"<§.§<!+§;
   import §0#1§.§7"D§;
   import §1#"§.§%!]§;
   import §1#"§.§=!j§;
   import §1V§.*;
   import §3"C§.§"!I§;
   import §3R§.§0"1§;
   import §4!X§.§,#H§;
   import §4"+§.§@J§;
   import §4"+§.§^$3§;
   import §4"6§.*;
   import §4#p§.§+!W§;
   import §5"U§.§;"M§;
   import §6!O§.§3"e§;
   import §6!O§.§;$#§;
   import §6!y§.§;!g§;
   import §6k§.§%§;
   import §7!`§.*;
   import §7!o§.*;
   import §7#F§.§8#P§;
   import §9!u§.*;
   import §9"j§.§1[§;
   import §9"j§.§9"X§;
   import §;"I§.*;
   import §;"S§.DamageEvent;
   import §<$=§.§'!Y§;
   import §<$=§.§3"o§;
   import §="z§.§,#f§;
   import §?"Z§.§2$!§;
   import §?=§.§3$%§;
   import §?=§.§>$T§;
   import §@#M§.*;
   import §[#M§.§!3§;
   import §[#M§.§'4§;
   import §[#b§.§]W§;
   import §[z§.§,W§;
   import §[z§.§6#,§;
   import §]`§.§ !S§;
   import §]`§.§+$§;
   import §]s§.§#§;
   import §]s§.§]#-§;
   import §]w§.§;!K§;
   import §^"$§.§,!7§;
   import §^"r§.§ 4§;
   import §^"r§.§0"U§;
   import §^#c§.§5N§;
   import §^$M§.§6]§;
   import §^1§.§+!`§;
   import §`!f§.§"!8§;
   import §`!g§.§="w§;
   import §`$K§.§0"_§;
   import §`$K§.§6$Q§;
   import api.com.odnoklassniki.Odnoklassniki;
   import flash.display.BitmapData;
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.display.Stage;
   import flash.errors.IllegalOperationError;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.events.TimerEvent;
   import flash.geom.Point;
   import flash.geom.Rectangle;
   import flash.text.TextField;
   import flash.text.TextFormat;
   import flash.utils.ByteArray;
   import flash.utils.Dictionary;
   import flash.utils.IDataOutput;
   import flash.utils.Timer;
   import ru.pragmatix.flox.gui.controls.§0b§;
   import ru.pragmatix.flox.gui.controls.§7$ §;
   import ru.pragmatix.flox.gui.controls.§?"W§;
   import ru.pragmatix.flox.social.controller.*;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.game.§7$J§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.window.§#$Q§;
   import ru.pragmatix.wormix.messages.clans.main.server.*;
   import ru.pragmatix.wormix.messages.clans.request.PostClanNewsRequest;
   import ru.pragmatix.wormix.messages.clans.request.edit.ChangeClanJoinRatingRequest;
   import ru.pragmatix.wormix.messages.client.DistributePoints;
   import ru.pragmatix.wormix.messages.client.Login;
   import ru.pragmatix.wormix.messages.server.NeedMoneyResult;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.objects.mutators.*;
   import ru.pragmatix.wormix.physics.§@$ §;
   import ru.pragmatix.wormix.pvp.messages.client.PvpActionEx;
   import ru.pragmatix.wormix.pvp.messages.ex.PvpBattleType;
   import ru.pragmatix.wormix.serialization.client.*;
   import ru.pragmatix.wormix.social.§#!K§;
   import ru.pragmatix.wormix.social.facebook.controller.bank.FbRegisterPaymentResult;
   import ru.pragmatix.wormix.utils.§<!f§;
   import ru.pragmatix.wormix.weapons.*;
   
   public class §#$N§ extends §7$ §
   {
      
      public var params:§["3§;
      
      private var container:DisplayObject;
      
      private var §^$I§:Boolean = false;
      
      private var §?§:Boolean = false;
      
      private var hint:DisplayObject;
      
      private var §`#P§:Timer;
      
      private var hideTimer:Timer;
      
      private var §6"&§:§9!1§;
      
      public function §#$N§(param1:DisplayObject, param2:§["3§ = null)
      {
         super();
         this.container = param1;
         this.§ $+§(param2);
      }
      
      public function §'!+§(param1:§9!1§, param2:§["3§ = null) : void
      {
         this.§6"&§ = param1;
         this.§ $+§(param2);
         this.container.addEventListener(MouseEvent.MOUSE_MOVE,this.§'$"§);
         this.container.addEventListener(MouseEvent.ROLL_OUT,this.§`"K§);
         this.container.addEventListener(Event.REMOVED_FROM_STAGE,this.onOwnerRemoved);
      }
      
      // Patch: the hint lives on the scene, not inside its owner. When the owner leaves the screen under
      // the mouse (arsenal closed after choosing a weapon), Ruffle sends no ROLL_OUT, so the hint stayed.
      // Remove it, and cancel one that is still waiting to appear.
      private function onOwnerRemoved(param1:Event) : void
      {
         this.§^$I§ = false;
         if(this.§`#P§)
         {
            this.§`#P§.stop();
            this.§`#P§ = null;
         }
         if(this.hideTimer)
         {
            this.hideTimer.stop();
            this.hideTimer = null;
         }
         this.removeHintImmediately();
      }
      
      public function § $+§(param1:§["3§) : void
      {
         if(param1)
         {
            this.params = param1;
         }
         else
         {
            this.params = new §["3§();
         }
      }
      
      public function §+"f§() : §9!1§
      {
         return this.§6"&§;
      }
      
      public function §""Y§() : void
      {
         if(this.§6"&§)
         {
            this.container.removeEventListener(MouseEvent.MOUSE_MOVE,this.§'$"§);
            this.container.removeEventListener(MouseEvent.ROLL_OUT,this.§`"K§);
            this.container.removeEventListener(Event.REMOVED_FROM_STAGE,this.onOwnerRemoved);
            this.removeHintImmediately();
            this.§6"&§ = null;
         }
      }
      
      public function §'$"§(param1:MouseEvent) : void
      {
         var cntRect:Rectangle;
         var e:MouseEvent = param1;
         if(this.§6"&§ == null || this.§?§)
         {
            return;
         }
         this.hint = this.§6"&§.§4"B§();
         cntRect = this.container.getRect(§;$#§.scene);
         switch(this.params.§2#N§)
         {
            case §@$I§.DEFAULT:
               this.hint.x = e.stageX + 5;
               this.hint.y = e.stageY;
               if(this.hint.x + this.hint.width > §;$#§.stageWidth)
               {
                  this.hint.x = e.stageX - this.hint.width - this.params.§%o§;
               }
               if(this.hint.y + this.hint.height > §;$#§.stageHeight)
               {
                  this.hint.y = §;$#§.stageHeight - this.hint.height - this.params.§;#x§;
               }
               break;
            case §@$I§.§ #3§:
               §,!7§.§ !3§(this.hint,cntRect.x,cntRect.right);
               this.hint.y = cntRect.bottom + 5;
               break;
            case §@$I§.§^!P§:
               §,!7§.§ !3§(this.hint,cntRect.x,cntRect.right);
               this.hint.y = cntRect.top - this.hint.height - 5;
               break;
            case §@$I§.LEFT:
               §,!7§.§&"#§(this.hint,cntRect.y,cntRect.bottom);
               this.hint.x = cntRect.x - this.hint.width - 5;
               break;
            case §@$I§.RIGHT:
               §,!7§.§&"#§(this.hint,cntRect.y,cntRect.bottom);
               this.hint.x = cntRect.right + 5;
               break;
            case §@$I§.§;"w§:
               this.hint.x = this.params.x;
               this.hint.y = this.params.y;
               break;
            default:
               if(this.params.§2#N§ is §2$,§)
               {
                  if(§2$,§(this.params.§2#N§).isAbsolute)
                  {
                     this.hint.x = §2$,§(this.params.§2#N§).x;
                     this.hint.y = §2$,§(this.params.§2#N§).y;
                  }
                  else
                  {
                     §,!7§.§ !3§(this.hint,cntRect.x,cntRect.right);
                     §,!7§.§&"#§(this.hint,cntRect.y,cntRect.bottom);
                     this.hint.x += §2$,§(this.params.§2#N§).x;
                     this.hint.y += §2$,§(this.params.§2#N§).y;
                  }
               }
         }
         if(this.params.§2#N§ != §@$I§.DEFAULT)
         {
            if(this.hint.x + this.hint.width > §;$#§.stageWidth)
            {
               this.hint.x = §;$#§.stageWidth - this.hint.width - this.params.§%o§;
            }
            else if(this.hint.x < 0)
            {
               this.hint.x = this.params.§%o§;
            }
         }
         this.§^$I§ = true;
         if(this.§`#P§)
         {
            this.§`#P§.stop();
         }
         this.§`#P§ = new Timer(this.params.§[#S§);
         this.§`#P§.addEventListener(TimerEvent.TIMER,function():void
         {
            if(§^$I§)
            {
               §?§ = true;
               §+c§().addChild(hint);
               §,h§();
               if(params.§#s§)
               {
                  hint.addEventListener(MouseEvent.MOUSE_MOVE,§@!!§);
                  hint.addEventListener(MouseEvent.ROLL_OUT,§3a§);
               }
            }
            §`#P§.stop();
            §`#P§ = null;
         });
         this.§`#P§.start();
      }
      
      protected function §`"K§(param1:MouseEvent) : void
      {
         this.§^$I§ = false;
         this.removeHint();
      }
      
      public function removeHint() : void
      {
         if(this.§?§)
         {
            if(this.hideTimer)
            {
               this.hideTimer.stop();
            }
            this.hideTimer = new Timer(this.params.§<!_§);
            this.hideTimer.addEventListener(TimerEvent.TIMER,function():void
            {
               removeHintImmediately();
               hideTimer.stop();
               hideTimer = null;
            });
            this.hideTimer.start();
         }
      }
      
      public function removeHintImmediately() : void
      {
         if(!this.§?§)
         {
            this.§^$I§ = false;
            if(this.§`#P§)
            {
               this.§`#P§.stop();
            }
         }
         else
         {
            this.§?§ = false;
            this.§^$I§ = false;
            if(this.hint)
            {
               if(this.§+c§().contains(this.hint))
               {
                  this.§+c§().removeChild(this.hint);
               }
               this.hint.removeEventListener(MouseEvent.MOUSE_MOVE,this.§@!!§);
               this.hint.removeEventListener(MouseEvent.ROLL_OUT,this.§3a§);
               this.hint = null;
            }
            removeEventListener(§;$#§.scene.stage,MouseEvent.MOUSE_DOWN,this.§%"g§);
         }
      }
      
      private function §@!!§(param1:MouseEvent) : void
      {
         if(this.params.§#s§)
         {
            if(this.hideTimer)
            {
               this.hideTimer.stop();
               this.hideTimer = null;
            }
         }
      }
      
      private function §3a§(param1:MouseEvent) : void
      {
         this.removeHintImmediately();
      }
      
      private function §+c§() : DisplayObjectContainer
      {
         return Boolean(this.params) && Boolean(this.params.§,! §) ? this.params.§,! § : §;$#§.scene;
      }
      
      private function §,h§() : void
      {
         addEventListener(§;$#§.scene.stage,MouseEvent.MOUSE_DOWN,this.§%"g§,false,int.MAX_VALUE);
      }
      
      private function §%"g§(param1:MouseEvent) : void
      {
         if(!this.container.hitTestPoint(param1.stageX,param1.stageY))
         {
            this.removeHintImmediately();
         }
      }
   }
}

