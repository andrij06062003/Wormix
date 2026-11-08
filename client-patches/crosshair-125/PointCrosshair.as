package §!!=§
{
   import §%#]§.§>!V§;
   import §%#l§.*;
   import §&!Y§.§8"9§;
   import §+0§.§"$L§;
   import §5"U§.§;"M§;
   import §6!O§.§;$#§;
   import §6k§.§%§;
   import §;"S§.§,$6§;
   import §@#n§.§[q§;
   import §[#b§.§+#c§;
   import §[#b§.§[" §;
   import §[#b§.§]W§;
   import §[z§.§6#,§;
   import §^1§.§+!`§;
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.geom.Point;
   import flash.ui.Mouse;
   import flash.utils.getTimer;
   import ru.pragmatix.flox.logger.§[$Q§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.weapons.§1=§;
   
   public class §'2§ extends §`$9§
   {
      
      private var log:§[$Q§ = new §[$Q§(§'2§);
      
      private var § #x§:MovieClip;
      
      public var §+!p§:Boolean = false;
      
      public var targetX:int;
      
      public var targetY:int;
      
      private var § 2§:Function;
      
      private var §>"r§:Function;
      
      private const §]$Q§:uint = 80;
      
      private const §6"P§:uint = 60;
      
      private const §?!%§:uint = 140;
      
      private const §,#%§:uint = 50;
      
      private const §#"L§:uint = 50;
      
      private const §]!^§:int = 300;
      
      private const §4$J§:int = 5;
      
      private var §6#w§:int;
      
      private var §>$L§:int;
      
      private var §<"H§:int;
      
      public var §8"F§:Boolean = false;
      
      private var §`#`§:Boolean = false;
      
      public function §'2§(param1:Function = null, param2:Function = null)
      {
         super();
         this.§ 2§ = param1;
         this.§>"r§ = param2;
         this.§ #x§ = §&#M§.§&#K§;
      }
      
      public function setCrosshairClip(param1:MovieClip) : void
      {
         if(§-"d§.§`"R§.§!!Q§.§-$J§())
         {
            if(this.§8"F§)
            {
               this.§3!#§();
            }
            this.§ #x§ = param1;
            if(this.§8"F§)
            {
               this.§-!+§();
            }
         }
      }
      
      private function §-!+§() : void
      {
         if(!this.§`#`§)
         {
            Mouse.hide();
            this.§ #x§.x = Application.curScene.mouseX;
            this.§ #x§.y = Application.curScene.mouseY;
            Application.curScene.addChild(this.§ #x§);
            this.§`#`§ = true;
         }
      }
      
      private function §3!#§() : void
      {
         if(this.§`#`§)
         {
            Application.curScene.removeChild(this.§ #x§);
            Mouse.show();
            this.§`#`§ = false;
         }
      }
      
      override public function activate(param1:§+!`§) : void
      {
         if(!this.§8"F§)
         {
            super.activate(param1);
            this.§8"F§ = true;
            if(§-"d§.§`"R§.§!!Q§.§-$J§())
            {
               this.§-!+§();
               Application.curScene.addEventListener(Event.ENTER_FRAME,this.§#"l§,false,int.MIN_VALUE);
               Application.curScene.addEventListener(MouseEvent.CLICK,this.§&#5§,false);
               §%#6§.§5!W§.activate();
            }
            if(param1.direction == §+!`§.LEFT)
            {
               §1=§(param1.weapon).rotateWeaponClip(Math.PI);
            }
            else
            {
               §1=§(param1.weapon).rotateWeaponClip(0);
            }
            this.§6#w§ = this.§>$L§ = this.§<"H§ = 0;
         }
      }
      
      private function §&#5§(param1:MouseEvent) : void
      {
         var _loc2_:Point = new Point(this.§ #x§.x,this.§ #x§.y);
         if(getTimer() - this.§6#w§ < this.§]!^§ && Math.abs(_loc2_.x - this.§>$L§) < this.§4$J§ && Math.abs(_loc2_.y - this.§<"H§) < this.§4$J§)
         {
            param1.stopPropagation();
            Application.curScene.dispatchEvent(new §]W§([new §[" §(§+#c§.§,!E§,_loc2_.x,_loc2_.y)]));
         }
         this.§6#w§ = getTimer();
         this.§>$L§ = _loc2_.x;
         this.§<"H§ = _loc2_.y;
      }
      
      override public function point(param1:int, param2:int) : void
      {
         if(this.§ 2§ != null && !this.§ 2§(param1,param2))
         {
            §>!V§.play(§8"9§.§'"1§);
            return;
         }
         this.§+!p§ = true;
         §>!V§.play(§8"9§.§,"§);
         var _loc3_:§[q§ = new §[q§();
         _loc3_.point = new Point(param1,param2);
         dispatchEvent(new §,$6§(_loc3_));
      }
      
      private function §#"l§(param1:Event) : void
      {
         if(!this.§8"F§)
         {
            Application.curScene.removeEventListener(Event.ENTER_FRAME,this.§#"l§);
            this.§3!#§();
            return;
         }
         var _loc2_:Point = new Point(Application.curScene.mouseX,Application.curScene.mouseY);
         var _loc3_:Point = Application.curScene.localToGlobal(_loc2_);
         if(_loc3_.y < this.§]$Q§ && (_loc3_.x < this.§6"P§ || _loc3_.x > §;$#§.stageWidth - this.§?!%§) || _loc3_.x < this.§,#%§ || _loc3_.x > §;$#§.stageWidth - this.§#"L§)
         {
            this.§3!#§();
         }
         else
         {
            this.§-!+§();
            this.§ #x§.x = _loc2_.x;
            this.§ #x§.y = _loc2_.y;
            Application.curScene.setChildIndex(this.§ #x§,Application.curScene.numChildren - 1);
            if(this.§>"r§ != null)
            {
               this.§>"r§(this.§ #x§.x,this.§ #x§.y);
            }
         }
      }
      
      override public function deactivate() : void
      {
         if(this.§8"F§)
         {
            super.deactivate();
            this.§8"F§ = false;
            if(§-"d§.§`"R§.§!!Q§.§-$J§())
            {
               §%#6§.§5!W§.deactivate();
            }
            Application.curScene.removeEventListener(Event.ENTER_FRAME,this.§#"l§);
            Application.curScene.removeEventListener(MouseEvent.CLICK,this.§&#5§,false);
            this.§3!#§();
         }
      }
   }
}

