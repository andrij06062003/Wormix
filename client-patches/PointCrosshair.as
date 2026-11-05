package ru.pragmatix.wormix.crosshair
{
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.geom.Point;
   import flash.ui.Mouse;
   import flash.utils.getTimer;
   import ru.pragmatix.flox.gui.core.UISettings;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.sound.SoundController;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.controls.ControlSets;
   import ru.pragmatix.wormix.events.CrosshairEvent;
   import ru.pragmatix.wormix.events.GameEvent;
   import ru.pragmatix.wormix.model.Sounds;
   import ru.pragmatix.wormix.objects.Worm;
   import ru.pragmatix.wormix.pvp.action.Action;
   import ru.pragmatix.wormix.pvp.action.ActionEvent;
   import ru.pragmatix.wormix.pvp.action.Actions;
   import ru.pragmatix.wormix.weapons.Weapon;
   import ru.pragmatix.wormix.weapons.paramsets.ShotParams;
   
   public class PointCrosshair extends AbstractCrosshair
   {
      
      private var log:Logger = new Logger(PointCrosshair);
      
      private var chClip:MovieClip;
      
      public var aimed:Boolean = false;
      
      public var targetX:int;
      
      public var targetY:int;
      
      private var onClick:Function;
      
      private var onMouseMove:Function;
      
      private const MARGIN_TOP:uint = 80;
      
      private const MARGIN_TOP_LEFT:uint = 60;
      
      private const MARGIN_TOP_RIGHT:uint = 140;
      
      private const MARGIN_LEFT:uint = 50;
      
      private const MARGIN_RIGHT:uint = 50;
      
      private const CLICK_TIME:int = 300;
      
      private const CLICK_OFFSET:int = 5;
      
      private var doubleTimer:int;
      
      private var clickX:int;
      
      private var clickY:int;
      
      public var activated:Boolean = false;
      
      private var crosshairVisible:Boolean = false;
      
      public function PointCrosshair(onClick:Function = null, onMouseMove:Function = null)
      {
         super();
         this.onClick = onClick;
         this.onMouseMove = onMouseMove;
         this.chClip = Crosshairs.DEFAULT_CROSSHAIR;
      }
      
      public function setCrosshairClip(clip:MovieClip) : void
      {
         if(Singletons.btc.gameMaster.playersTurn())
         {
            if(this.activated)
            {
               this.hideCrosshair();
            }
            this.chClip = clip;
            if(this.activated)
            {
               this.showCrosshair();
            }
         }
      }
      
      private function showCrosshair() : void
      {
         if(!this.crosshairVisible)
         {
            Mouse.hide();
            this.chClip.x = Application.curScene.mouseX;
            this.chClip.y = Application.curScene.mouseY;
            Application.curScene.addChild(this.chClip);
            this.crosshairVisible = true;
         }
      }
      
      private function hideCrosshair() : void
      {
         if(this.crosshairVisible)
         {
            if(this.chClip.parent != null)
            {
               this.chClip.parent.removeChild(this.chClip);
            }
            Mouse.show();
            this.crosshairVisible = false;
         }
      }
      
      override public function activate(worm:Worm) : void
      {
         if(!this.activated)
         {
            super.activate(worm);
            this.activated = true;
            if(Singletons.btc.gameMaster.playersTurn())
            {
               this.showCrosshair();
               Application.curScene.addEventListener(Event.ENTER_FRAME,this.moveCrosshair,false,int.MIN_VALUE);
               Application.curScene.addEventListener(MouseEvent.CLICK,this.takeAim,false);
               ControlSets.CROSSHAIR_CONTROL_SET.activate();
            }
            if(worm.direction == Worm.LEFT)
            {
               Weapon(worm.weapon).rotateWeaponClip(Math.PI);
            }
            else
            {
               Weapon(worm.weapon).rotateWeaponClip(0);
            }
            this.doubleTimer = this.clickX = this.clickY = 0;
         }
      }
      
      private function takeAim(e:MouseEvent) : void
      {
         var glPoint:Point = new Point(e.stageX,e.stageY);
         var lcPoint:Point = Application.curScene.globalToLocal(glPoint);
         if(getTimer() - this.doubleTimer < this.CLICK_TIME && Math.abs(lcPoint.x - this.clickX) < this.CLICK_OFFSET && Math.abs(lcPoint.y - this.clickY) < this.CLICK_OFFSET)
         {
            e.stopPropagation();
            Application.curScene.dispatchEvent(new ActionEvent([new Action(Actions.CS_POINT,lcPoint.x,lcPoint.y)]));
         }
         this.doubleTimer = getTimer();
         this.clickX = lcPoint.x;
         this.clickY = lcPoint.y;
      }
      
      override public function point(x:int, y:int) : void
      {
         if(this.onClick != null && !this.onClick(x,y))
         {
            SoundController.play(Sounds.MENU_DENY);
            return;
         }
         this.aimed = true;
         SoundController.play(Sounds.MENU_GRANT);
         var params:ShotParams = new ShotParams();
         params.point = new Point(x,y);
         dispatchEvent(new CrosshairEvent(params));
         dispatchEvent(new Event(GameEvent.WEAPON_SHOOT));
      }
      
      private function moveCrosshair(e:Event) : void
      {
         if(!this.activated)
         {
            Application.curScene.removeEventListener(Event.ENTER_FRAME,this.moveCrosshair);
            this.hideCrosshair();
            return;
         }
         var lcPoint:Point = new Point(Application.curScene.mouseX,Application.curScene.mouseY);
         var glPoint:Point = Application.curScene.localToGlobal(lcPoint);
         if(glPoint.y < this.MARGIN_TOP && (glPoint.x < this.MARGIN_TOP_LEFT || glPoint.x > UISettings.stageWidth - this.MARGIN_TOP_RIGHT) || glPoint.x < this.MARGIN_LEFT || glPoint.x > UISettings.stageWidth - this.MARGIN_RIGHT)
         {
            this.hideCrosshair();
         }
         else
         {
            this.showCrosshair();
            this.chClip.x = lcPoint.x;
            this.chClip.y = lcPoint.y;
            Application.curScene.setChildIndex(this.chClip,Application.curScene.numChildren - 1);
            if(this.onMouseMove != null)
            {
               this.onMouseMove(this.chClip.x,this.chClip.y);
            }
         }
      }
      
      override public function deactivate() : void
      {
         if(this.activated)
         {
            super.deactivate();
            this.activated = false;
            if(Singletons.btc.gameMaster.playersTurn())
            {
               ControlSets.CROSSHAIR_CONTROL_SET.deactivate();
            }
            Application.curScene.removeEventListener(Event.ENTER_FRAME,this.moveCrosshair);
            Application.curScene.removeEventListener(MouseEvent.CLICK,this.takeAim,false);
            this.hideCrosshair();
         }
      }
      
      public function commands() : Object
      {
         return {"point":this.point};
      }
   }
}

