package ru.pragmatix.wormix.objects.graphics
{
   import flash.display.DisplayObject;
   import flash.display.MovieClip;
   import ru.pragmatix.flox.core.FpsCounter;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.race.Race;
   import ru.pragmatix.wormix.model.race.RaceRecord;
   
   public class CharGraphics extends AbstractGraphics
   {
      
      private static const MIN_FPS:uint = 20;
      
      private var log:Logger = new Logger(CharGraphics);
      
      private var charRecord:RaceRecord;
      
      private var hat:HatRecord;

      private var artifact:HatRecord;

      protected var eyes:MovieClip;
      
      protected var leftHand:MovieClip;
      
      protected var rightHand:MovieClip;
      
      protected var locked:Boolean = false;
      
      public function CharGraphics(ch:RaceRecord = null)
      {
         super();
         if(Boolean(ch))
         {
            this.setChar(ch);
         }
         container.mouseEnabled = true;
      }
      
      public function getConfigName() : String
      {
         return this.charRecord.configName;
      }
      
      public function setHat(hat:HatRecord) : void
      {
         // An artifact is held in the hand, not worn on the head.
         if(Boolean(hat) && hat.isArtifact)
         {
            return;
         }
         this.hat = hat;
      }

      // The artifact (1.25.0) is drawn in the "offHand" clip of the left hand (graphics of 1.25.0's
      // charActions.swf). Races without hands (worm, bull) don't show it.
      public function setArtifact(artifact:HatRecord) : void
      {
         this.artifact = Boolean(artifact) && artifact.isArtifact ? artifact : null;
      }

      public function getArtifact() : HatRecord
      {
         return this.artifact;
      }

      private function prepareArtifact(clip:MovieClip) : void
      {
         var offHand:MovieClip = null;
         var hand:DisplayObject = null;
         var artifactClip:MovieClip = null;
         var variants:MovieClip = null;
         var i:int = 0;
         if(!this.leftHand || !(this.leftHand.getChildByName("offHand") is MovieClip))
         {
            return;
         }
         offHand = this.leftHand.getChildByName("offHand") as MovieClip;
         hand = this.leftHand.getChildByName("hand");
         if(Boolean(this.artifact))
         {
            artifactClip = getClipByName(this.artifact.getClipClassName());
            // The clip has a variant for each race: keep ours only.
            variants = artifactClip.getChildByName("hand") as MovieClip;
            if(Boolean(variants))
            {
               i = 0;
               while(i < variants.numChildren)
               {
                  variants.getChildAt(i).visible = variants.getChildAt(i).name == this.charRecord.configName;
                  i++;
               }
            }
         }
         // Empty the slot (it holds a sample shield in the graphics), then put the artifact in.
         while(offHand.numChildren > 0 && (!artifactClip || offHand.getChildAt(0) != artifactClip))
         {
            offHand.removeChildAt(0);
         }
         if(Boolean(artifactClip) && !offHand.contains(artifactClip))
         {
            offHand.addChild(artifactClip);
         }
         if(Boolean(hand))
         {
            hand.visible = !this.artifact || this.artifact.handVisible;
            // handFront: the paw is drawn over the artifact (a sword's hilt), else behind it (a shield).
            if(Boolean(this.artifact))
            {
               if(this.artifact.handFront == (this.leftHand.getChildIndex(hand) < this.leftHand.getChildIndex(offHand)))
               {
                  this.leftHand.swapChildren(hand,offHand);
               }
            }
         }
      }
      
      public function setChar(ch:RaceRecord) : void
      {
         this.charRecord = ch;
         if(Boolean(this.charRecord.eyesClip) && Boolean(getClip().getChildByName(this.charRecord.eyesClip)))
         {
            this.eyes = getClip().getChildByName(this.charRecord.eyesClip) as MovieClip;
         }
         this.stay();
      }
      
      public function getCharRecord() : RaceRecord
      {
         return this.charRecord;
      }
      
      public function stay() : void
      {
         this.play(this.charRecord.stayClip);
      }
      
      public function stayUp() : void
      {
         this.play(this.charRecord.stayUpClip);
      }
      
      public function stayDown() : void
      {
         this.play(this.charRecord.stayDownClip);
      }
      
      public function goUp() : void
      {
         this.play(this.charRecord.goUpClip);
      }
      
      public function goDown() : void
      {
         this.play(this.charRecord.goDownClip);
      }
      
      public function go() : void
      {
         this.play(this.charRecord.goClip);
      }
      
      public function poisoned() : void
      {
         this.play(this.charRecord.poisonedClip,false);
      }
      
      public function damage() : uint
      {
         if(Boolean(this.hat))
         {
            return this.play(this.charRecord.damageInHat);
         }
         return this.play(this.charRecord.damage);
      }
      
      public function greateDamage() : uint
      {
         if(Boolean(this.hat))
         {
            return this.play(this.charRecord.greatDamageInHat);
         }
         return this.play(this.charRecord.greatDamage);
      }
      
      public function prepareToJump() : uint
      {
         return this.play(this.charRecord.prepareToJumpClip,true);
      }
      
      public function jump() : void
      {
         this.play(this.charRecord.jumpClip,true);
      }
      
      public function endJump() : void
      {
         this.play(this.charRecord.endJumpClip,true);
      }
      
      public function jumpBack() : void
      {
         this.play(this.charRecord.jumpBackClip,true);
      }
      
      public function breath(isSad:Boolean) : void
      {
         this.playOrStay(isSad ? this.charRecord.sadClip : this.charRecord.breathClip,this.charRecord.stayClip);
      }
      
      public function breathUp(isSad:Boolean) : void
      {
         this.playOrStay(isSad ? this.charRecord.sadUpClip : this.charRecord.breathUpClip,this.charRecord.stayUpClip);
      }
      
      public function breathDown(isSad:Boolean) : void
      {
         this.playOrStay(isSad ? this.charRecord.sadDownClip : this.charRecord.breathDownClip,this.charRecord.stayDownClip);
      }
      
      public function playRandomStateAction() : uint
      {
         var index:int = 0;
         var actions:Array = Boolean(this.hat) ? this.charRecord.stateActionsInHat : this.charRecord.stateActions;
         if(Boolean(actions) && Boolean(actions.length))
         {
            index = Math.floor(Math.random() * (actions.length - 1));
            return this.playOrStay(actions[index],this.charRecord.stayClip);
         }
         return 0;
      }
      
      public function clap() : uint
      {
         return this.play(this.charRecord.clapClip);
      }
      
      public function jetpack() : void
      {
         this.play(this.charRecord.jetpack);
      }
      
      public function jetpackStay() : void
      {
         this.log.debug("jet:" + this.charRecord.jetpackStay,"weapon");
         this.play(this.charRecord.jetpackStay);
      }
      
      public function hammer() : uint
      {
         return this.play(this.charRecord.hammer,true);
      }
      
      public function bat() : uint
      {
         return this.play(this.charRecord.bat,true);
      }
      
      public function rope() : uint
      {
         return this.play(this.charRecord.rope,true);
      }
      
      public function applyShoker() : uint
      {
         var clip:MovieClip = getClipByName(this.charRecord.shocker);
         container.addChild(clip);
         return clip.totalFrames;
      }
      
      public function removeShoker() : void
      {
         container.removeChild(getClipByName(this.charRecord.shocker));
      }
      
      public function applyDefrost() : uint
      {
         var clip:MovieClip = getClipByName(this.charRecord.defrost);
         container.addChild(clip);
         return clip.totalFrames;
      }
      
      public function removeDefrost() : void
      {
         container.removeChild(getClipByName(this.charRecord.defrost));
      }
      
      public function damaged() : void
      {
         this.play(this.charRecord.damaged);
      }
      
      public function happy() : void
      {
         if(Boolean(this.charRecord.happy))
         {
            this.play(this.charRecord.happy);
         }
      }
      
      public function playSpecial(clipName:String, restart:Boolean = false) : int
      {
         return this.play(clipName,restart);
      }
      
      private function playOrStay(clipName:String, stayClipName:String, restart:Boolean = false) : int
      {
         return this.play(FpsCounter.getFPS() > MIN_FPS ? clipName : stayClipName,restart);
      }
      
      override protected function play(clipName:String, restart:Boolean = false) : int
      {
         return !this.locked ? int(super.play(clipName,restart)) : 0;
      }
      
      override protected function prepareClipBeforePlay(clip:MovieClip) : void
      {
         var head:MovieClip = null;
         var hatContainer:MovieClip = null;
         var hairs:MovieClip = null;
         var ears:DisplayObject = null;
         var hatClip:DisplayObject = null;
         super.prepareClipBeforePlay(clip);
         if(this.charRecord.getId() != Race.BULL.getId())
         {
            head = clip.getChildByName("head") as MovieClip;
            if(!head)
            {
               head = clip;
            }
            if(Boolean(this.charRecord.leftHand) && Boolean(clip.getChildByName(this.charRecord.leftHand)))
            {
               this.leftHand = clip.getChildByName(this.charRecord.leftHand) as MovieClip;
            }
            if(Boolean(this.charRecord.rightHand) && Boolean(clip.getChildByName(this.charRecord.rightHand)))
            {
               this.rightHand = clip.getChildByName(this.charRecord.rightHand) as MovieClip;
            }
            if(Boolean(this.charRecord.leftHand) && clip.getChildByName(this.charRecord.leftHand) == this.leftHand)
            {
               this.prepareArtifact(clip);
            }
            hatContainer = head.getChildByName(this.charRecord.hatClip) as MovieClip;
            if(Boolean(this.hat))
            {
               hatClip = getClipByName(this.hat.getClipClassName());
               if(Boolean(hatContainer) && Boolean(hatContainer.numChildren) && !hatContainer.contains(hatClip))
               {
                  hatContainer.removeChildAt(0);
               }
            }
            else if(Boolean(hatContainer) && Boolean(hatContainer.numChildren))
            {
               hatContainer.removeChildAt(0);
            }
            if(Boolean(this.hat) && Boolean(hatContainer) && !hatContainer.numChildren)
            {
               hatContainer.addChild(hatClip);
            }
            hairs = head.getChildByName("hair") as MovieClip;
            if(Boolean(hairs))
            {
               hairs.visible = !this.hat || !this.hat.hideHair;
            }
            if(Boolean(this.charRecord.earsClip) && (Boolean(ears = clip.getChildByName(this.charRecord.earsClip))))
            {
               ears.visible = !(Boolean(this.hat) && Boolean(hatContainer) && this.hat.getIsHideEars());
            }
         }
      }
      
      public function stopEyes() : void
      {
         if(this.eyes != null)
         {
            this.eyes.stop();
         }
      }
      
      private function hideLeftHand() : void
      {
         if(this.leftHand != null)
         {
            this.leftHand.visible = false;
         }
      }
      
      private function hideRightHand() : void
      {
         if(this.rightHand != null)
         {
            this.rightHand.visible = false;
         }
      }
      
      private function hideAllHands() : void
      {
         this.hideLeftHand();
         this.hideRightHand();
      }
      
      public function hideHands(pattern:String) : void
      {
         switch(pattern)
         {
            case "right":
               this.hideRightHand();
               break;
            case "left":
               this.hideLeftHand();
               break;
            case "both":
               this.hideAllHands();
         }
      }
      
      public function showAllHands() : void
      {
         if(this.leftHand != null)
         {
            this.leftHand.visible = true;
         }
         if(this.rightHand != null)
         {
            this.rightHand.visible = true;
         }
      }
      
      public function lock() : void
      {
         this.locked = true;
      }
      
      public function unlock() : void
      {
         this.locked = false;
      }
   }
}

