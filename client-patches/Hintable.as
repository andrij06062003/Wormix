package ru.pragmatix.flox.gui.controls.hint
{
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.events.Event;
   import flash.events.MouseEvent;
   import flash.events.TimerEvent;
   import flash.geom.Rectangle;
   import flash.utils.Timer;
   import ru.pragmatix.flox.gui.core.UISettings;
   import ru.pragmatix.flox.gui.utils.InterfaceUtils;
   import ru.pragmatix.flox.logger.Logger;
   
   public class Hintable
   {
      
      private static var log:Logger = new Logger(Hintable);
      
      private var container:DisplayObject;
      
      public var params:HintParams;
      
      private var hint:DisplayObject;
      
      private var showTimer:Timer;
      
      private var needAddHint:Boolean = false;
      
      private var hintProvider:IHintProvider;
      
      private var hideTimer:Timer;
      
      private var hintAdded:Boolean = false;
      
      public function Hintable(container:DisplayObject, params:HintParams = null)
      {
         super();
         this.container = container;
         this.setParams(params);
      }
      
      private function getHintContainer() : DisplayObjectContainer
      {
         return Boolean(this.params) && Boolean(this.params.hintContainer) ? this.params.hintContainer : UISettings.scene;
      }
      
      public function setParams(params:HintParams) : void
      {
         if(Boolean(params))
         {
            this.params = params;
         }
         else
         {
            this.params = new HintParams();
         }
      }
      
      public function removeHintImmediately() : void
      {
         if(this.hintAdded)
         {
            this.hintAdded = false;
            this.needAddHint = false;
            if(Boolean(this.hint))
            {
               if(this.getHintContainer().contains(this.hint))
               {
                  this.getHintContainer().removeChild(this.hint);
               }
               this.hint.removeEventListener(MouseEvent.MOUSE_MOVE,this.hintOnMouseMove);
               this.hint.removeEventListener(MouseEvent.ROLL_OUT,this.hintOnMouseOut);
               this.hint = null;
            }
         }
      }
      
      public function getHintProvider() : IHintProvider
      {
         return this.hintProvider;
      }
      
      private function hintOnMouseOut(event:MouseEvent) : void
      {
         this.removeHintImmediately();
      }
      
      public function setHintProvider(hintProvider:IHintProvider, params:HintParams = null) : void
      {
         this.hintProvider = hintProvider;
         this.setParams(params);
         this.container.addEventListener(MouseEvent.MOUSE_MOVE,this.cntOnMouseMove);
         this.container.addEventListener(MouseEvent.ROLL_OUT,this.cntOnMouseOut);
         this.container.addEventListener(Event.REMOVED_FROM_STAGE,this.cntOnRemovedFromStage);
      }
      
      // Patch: the hint lives on the scene, not inside its owner. When the owner leaves the screen under
      // the mouse (arsenal closed after choosing a weapon), Ruffle sends no ROLL_OUT, so the hint stayed.
      // Remove it, and cancel one that is still waiting to appear.
      private function cntOnRemovedFromStage(e:Event) : void
      {
         this.needAddHint = false;
         if(Boolean(this.showTimer))
         {
            this.showTimer.stop();
            this.showTimer = null;
         }
         if(Boolean(this.hideTimer))
         {
            this.hideTimer.stop();
            this.hideTimer = null;
         }
         this.removeHintImmediately();
      }
      
      public function cntOnMouseMove(e:MouseEvent) : void
      {
         var cntRect:Rectangle;
         if(this.hintProvider == null || this.hintAdded)
         {
            return;
         }
         this.hint = this.hintProvider.getHint();
         cntRect = this.container.getRect(UISettings.scene);
         switch(this.params.hintMode)
         {
            case HintMode.DEFAULT:
               this.hint.x = e.stageX + 5;
               this.hint.y = e.stageY;
               if(this.hint.x + this.hint.width > UISettings.stageWidth)
               {
                  this.hint.x = e.stageX - this.hint.width - this.params.horisontalDist;
               }
               if(this.hint.y + this.hint.height > UISettings.stageHeight)
               {
                  this.hint.y = UISettings.stageHeight - this.hint.height - this.params.verticalDist;
               }
               break;
            case HintMode.UNDER:
               InterfaceUtils.centerX(this.hint,cntRect.x,cntRect.right);
               this.hint.y = cntRect.bottom + 5;
               break;
            case HintMode.ABOVE:
               InterfaceUtils.centerX(this.hint,cntRect.x,cntRect.right);
               this.hint.y = cntRect.top - this.hint.height - 5;
               break;
            case HintMode.LEFT:
               InterfaceUtils.centerY(this.hint,cntRect.y,cntRect.bottom);
               this.hint.x = cntRect.x - this.hint.width - 5;
               break;
            case HintMode.RIGHT:
               InterfaceUtils.centerY(this.hint,cntRect.y,cntRect.bottom);
               this.hint.x = cntRect.right + 5;
               break;
            default:
               if(this.params.hintMode is CustomHintMode)
               {
                  if(CustomHintMode(this.params.hintMode).isAbsolute)
                  {
                     this.hint.x = CustomHintMode(this.params.hintMode).x;
                     this.hint.y = CustomHintMode(this.params.hintMode).y;
                  }
                  else
                  {
                     InterfaceUtils.centerX(this.hint,cntRect.x,cntRect.right);
                     InterfaceUtils.centerY(this.hint,cntRect.y,cntRect.bottom);
                     this.hint.x += CustomHintMode(this.params.hintMode).x;
                     this.hint.y += CustomHintMode(this.params.hintMode).y;
                  }
               }
         }
         if(this.params.hintMode != HintMode.DEFAULT)
         {
            if(this.hint.x + this.hint.width > UISettings.stageWidth)
            {
               this.hint.x = UISettings.stageWidth - this.hint.width - this.params.horisontalDist;
            }
            else if(this.hint.x < 0)
            {
               this.hint.x = this.params.horisontalDist;
            }
         }
         this.needAddHint = true;
         if(Boolean(this.showTimer))
         {
            this.showTimer.stop();
         }
         this.showTimer = new Timer(this.params.showHintDelay);
         this.showTimer.addEventListener(TimerEvent.TIMER,function():void
         {
            if(needAddHint)
            {
               hintAdded = true;
               getHintContainer().addChild(hint);
               if(Boolean(params.hintContainer))
               {
                  log.debug("hint added in parent name \'" + hint.parent.name + "\' x=" + hint.x + " y = " + hint.y,"hint");
               }
               if(params.visibleOnMouseOver)
               {
                  hint.addEventListener(MouseEvent.MOUSE_MOVE,hintOnMouseMove);
                  hint.addEventListener(MouseEvent.ROLL_OUT,hintOnMouseOut);
               }
            }
            showTimer.stop();
            showTimer = null;
         });
         this.showTimer.start();
      }
      
      public function removeHint() : void
      {
         if(this.hintAdded)
         {
            if(Boolean(this.hideTimer))
            {
               this.hideTimer.stop();
            }
            this.hideTimer = new Timer(this.params.hideHintDelay);
            this.hideTimer.addEventListener(TimerEvent.TIMER,function():void
            {
               removeHintImmediately();
               hideTimer.stop();
               hideTimer = null;
            });
            this.hideTimer.start();
         }
      }
      
      protected function cntOnMouseOut(e:MouseEvent) : void
      {
         this.needAddHint = false;
         this.removeHint();
      }
      
      private function hintOnMouseMove(event:MouseEvent) : void
      {
         if(this.params.visibleOnMouseOver)
         {
            if(Boolean(this.hideTimer))
            {
               this.hideTimer.stop();
               this.hideTimer = null;
            }
         }
      }
      
      public function removeHintProvider() : void
      {
         if(Boolean(this.hintProvider))
         {
            this.container.removeEventListener(MouseEvent.MOUSE_MOVE,this.cntOnMouseMove);
            this.container.removeEventListener(MouseEvent.ROLL_OUT,this.cntOnMouseOut);
            this.container.removeEventListener(Event.REMOVED_FROM_STAGE,this.cntOnRemovedFromStage);
            this.removeHintImmediately();
            this.hintProvider = null;
         }
      }
   }
}

