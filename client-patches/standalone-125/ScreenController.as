package ru.pragmatix.wormix.controller
{
   import flash.external.ExternalInterface;
   import § !O§.Cookie;
   import §%"c§.§@"$§;
   import §&#q§.§>";§;
   import §4"+§.§^$3§;
   import §6!O§.§;$#§;
   import §;"I§.§2!9§;
   import §["y§.§9!k§;
   import flash.display.StageAlign;
   import flash.display.StageDisplayState;
   import flash.display.StageScaleMode;
   import flash.events.Event;
   import flash.events.FullScreenEvent;
   import flash.geom.Rectangle;
   import flash.system.Capabilities;
   import ru.pragmatix.flox.gui.controls.§?"W§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.social.§#!K§;
   import ru.pragmatix.wormix.utils.§5!#§;
   
   public class §"$-§ extends §@"$§
   {
      // FFDec compiles a call of IUIComponent.getContainer() as a public name, which doesn't find it: name the
      // interface's namespace, as the original code does.
      namespace iuc = "ru.pragmatix.flox.gui.controls:IUIComponent";
      
      public static const §%!n§:Number = 1;
      
      public static const §"$S§:Number = 1.2;
      
      public static const § "H§:Number = 0.55;
      
      public static const §6"Q§:uint = 1;
      
      private var §-"i§:Boolean = false;
      
      private var §?!d§:§2!9§;
      
      private var curScreenResolution:§2!9§;
      
      private const COOKIE_NAME_SCREEN_RESOLUTION_W:String = "COOKIE_NAME_SCREEN_RESOLUTION_W";
      
      private const COOKIE_NAME_SCREEN_RESOLUTION_H:String = "COOKIE_NAME_SCREEN_RESOLUTION_H";
      
      private var §4!h§:Boolean = false;
      
      public function §"$-§()
      {
         super(Application.instance.stage);
         stage.addEventListener(FullScreenEvent.FULL_SCREEN,this.§ $7§);
         init();
         this.§?!d§ = new §2!9§(Capabilities.screenResolutionX,Capabilities.screenResolutionY);
         var _loc1_:int = int(Cookie.getValue(this.COOKIE_NAME_SCREEN_RESOLUTION_W,-1));
         var _loc2_:int = int(Cookie.getValue(this.COOKIE_NAME_SCREEN_RESOLUTION_H,-1));
         if(_loc1_ == -1 || _loc2_ == -1)
         {
            this.curScreenResolution = this.§?!d§;
         }
         else
         {
            this.curScreenResolution = new §2!9§(_loc1_,_loc2_);
         }
      }
      
      private static function §4$8§() : void
      {
         §-"d§.§6$$§.§4#V§(Application.mainMenu.mainMenu.§ #B§());
         var _loc1_:§9!k§ = Application.mainMenu.§0K§;
         if(Boolean(_loc1_) && Boolean(_loc1_.§?C§))
         {
            §-"d§.§6$$§.§4#V§(_loc1_.§?C§.iuc::getContainer());
         }
      }
      
      private function § $7§(param1:FullScreenEvent) : void
      {
         Application.mainMenu.mainMenu.§?"P§.§",§();
         if(this.§-"i§)
         {
            this.§@"`§();
            this.§ _§(this.curScreenResolution);
         }
         if(stage.displayState == StageDisplayState.NORMAL)
         {
            this.§4#Z§(§#!K§.settings.§5"8§());
         }
         else
         {
            this.§4#Z§(false);
         }
         this.§<!%§();
         update();
         §>";§.§>!7§();
      }
      
      // Patch (standalone-125): the full screen button. The game's own full screen lays the menus out for the
      // screen's size, but the page shows the game at a fixed 900x600 (Ruffle's forceScale), so the buttons on
      // the right were cut off. The page puts the game's box in full screen instead (wwwroot/play.html,
      // window.wormixFullscreen), where it keeps its 900x600 layout.
      public function §+!h§() : void
      {
         try
         {
            if(ExternalInterface.available)
            {
               ExternalInterface.call("wormixFullscreen");
               return;
            }
         }
         catch(e:Error)
         {
         }
         var MIN_REQUIRED_VERSION:Number = 11.3;
         var curVersion:String = Capabilities.version.split(" ")[1];
         var version:Number = Number(curVersion.split(",")[0] + "." + curVersion.split(",")[1]);
         if(version < MIN_REQUIRED_VERSION)
         {
            §?"W§.§5#E§(§^!R§.BUTTON_CANCEL,§^!R§.UPDATE_YOUR_PLAYER,§?"W§.§3#r§).show();
            return;
         }
         try
         {
            if(stage.displayState == StageDisplayState.NORMAL)
            {
               stage.displayState = StageDisplayState.FULL_SCREEN_INTERACTIVE;
            }
            else
            {
               stage.displayState = StageDisplayState.NORMAL;
            }
         }
         catch(e:SecurityError)
         {
            §?"W§.§5#E§(§^!R§.BUTTON_CANCEL,§^!R§.UPDATE_YOUR_PLAYER,§?"W§.§3#r§).show();
         }
      }
      
      private function §[!8§() : void
      {
         if(this.§-"i§)
         {
            if(this.§;#!§())
            {
               this.§ _§(this.curScreenResolution);
            }
         }
         else if(this.§>#i§())
         {
            Application.mainMenu.getContainer().scrollRect = §<p§();
         }
         else
         {
            Application.mainMenu.getContainer().scrollRect = new Rectangle(0,0,Application.§[$S§,Application.§5Q§);
         }
         if(this.§;#!§())
         {
            stage.scaleMode = StageScaleMode.NO_SCALE;
            §;$#§.§8!b§(Capabilities.screenResolutionX,Capabilities.screenResolutionY);
            stage.align = StageAlign.TOP_LEFT;
            return;
         }
         stage.scaleMode = StageScaleMode.NO_SCALE;
         if(this.§>#i§() || this.§-"i§)
         {
            §;$#§.§8!b§(stage.stageWidth,stage.stageHeight);
            stage.align = StageAlign.TOP_LEFT;
         }
         else
         {
            §;$#§.§8!b§(Application.§[$S§,Application.§5Q§);
            stage.align = StageAlign.TOP;
         }
      }
      
      public function §4$A§() : void
      {
         this.§-"i§ = true;
         this.§[!8§();
         if(this.§;#!§())
         {
            §-"d§.§6$$§.§-#W§();
         }
         §6"=§(§#!K§.settings.§2"t§());
         §5!#§.§@!T§(1,function():void
         {
            changeResolution(curScreenResolution);
            Application.curScene.§'!d§.§<!%§();
            update();
         });
      }
      
      public function §@"`§() : void
      {
         var _loc1_:uint = §#!K§.settings.§+%§();
         var _loc2_:uint = §#!K§.settings.§[#1§();
         var _loc3_:Rectangle = §<p§();
         if(_loc3_.width > _loc1_)
         {
            _loc3_.width = _loc1_;
         }
         if(_loc3_.height > _loc2_)
         {
            _loc3_.height = _loc2_;
         }
         §;$#§.§,!'§(_loc3_);
         Application.curScene.§=#r§(this.§;#!§());
         Application.curScene.§'!d§.§<!%§();
         Application.curScene.§7$R§();
      }
      
      public function changeResolution(param1:§2!9§) : void
      {
         this.curScreenResolution = param1;
         Cookie.setValue(this.COOKIE_NAME_SCREEN_RESOLUTION_W,this.curScreenResolution.§,$J§());
         Cookie.setValue(this.COOKIE_NAME_SCREEN_RESOLUTION_H,this.curScreenResolution.§?!l§());
         if(this.§-"i§)
         {
            this.§ _§(this.curScreenResolution);
            if(this.§;#!§())
            {
               this.§@"`§();
            }
         }
         §;$#§.§8!b§(stage.stageWidth,stage.stageHeight);
         update();
      }
      
      public function §2$R§() : void
      {
         this.§-"i§ = false;
         this.§<!%§();
      }
      
      public function §<!%§() : void
      {
         if(this.§7k§())
         {
            this.§ _§(new §2!9§(Application.instance.stage.stageWidth,Application.instance.stage.stageHeight));
         }
         else
         {
            this.§ _§(this.§?!d§);
         }
         §4$8§();
         §6"=§(this.§7k§());
         this.§[!8§();
         Application.mainMenu.§?#M§.§'!w§();
         if(this.§-"i§)
         {
            §-"d§.§6$$§.§-#W§();
         }
         else if(this.§>#i§())
         {
            §-"d§.§6$$§.§%$<§(Application.mainMenu.mainMenu.§ #B§());
         }
      }
      
      public function §;#!§() : Boolean
      {
         return stage.displayState == StageDisplayState.FULL_SCREEN_INTERACTIVE;
      }
      
      public function §^L§() : §2!9§
      {
         return this.curScreenResolution;
      }
      
      public function §33§() : §2!9§
      {
         return this.§?!d§;
      }
      
      public function §^H§() : void
      {
         if(this.§;#!§())
         {
            this.§+!h§();
         }
      }
      
      private function § _§(param1:§2!9§) : void
      {
         stage.fullScreenSourceRect = new Rectangle(0,0,param1.§,$J§(),param1.§?!l§());
      }
      
      public function §?+§() : §2!9§
      {
         var _loc1_:§2!9§ = null;
         if(this.§;#!§())
         {
            _loc1_ = this.§^L§();
         }
         else if(§#!K§.settings.§2"t§())
         {
            _loc1_ = new §2!9§(Application.instance.stage.stageWidth,Application.instance.stage.stageHeight);
         }
         else
         {
            _loc1_ = new §2!9§(Application.§[$S§,Application.§5Q§);
         }
         return _loc1_;
      }
      
      public function getCorrectResolution2() : §2!9§
      {
         var _loc1_:§2!9§ = null;
         if(this.§;#!§())
         {
            _loc1_ = this.§33§();
         }
         else if(this.§7k§())
         {
            _loc1_ = new §2!9§(Application.instance.stage.stageWidth,Application.instance.stage.stageHeight);
         }
         else
         {
            _loc1_ = new §2!9§(Application.§[$S§,Application.§5Q§);
         }
         return _loc1_;
      }
      
      override protected function §@$L§(param1:Rectangle) : void
      {
         if(this.§-"i§)
         {
            this.§@"`§();
         }
         else
         {
            this.§<!%§();
         }
         if(Application.curScene)
         {
            Application.curScene.dispatchEvent(new Event(§^$3§.§5$,§));
         }
      }
      
      public function §4#Z§(param1:Boolean) : void
      {
         this.§4!h§ = param1;
      }
      
      public function §7k§() : Boolean
      {
         return this.§4!h§;
      }
      
      public function §>#i§() : Boolean
      {
         return this.§;#!§() || this.§7k§();
      }
      
      public function §4"!§() : Boolean
      {
         if(this.§?+§().§,$J§() / this.§?+§().§?!l§() > 1.4)
         {
            return true;
         }
         return false;
      }
   }
}

