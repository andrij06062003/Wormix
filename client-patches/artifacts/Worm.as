package ru.pragmatix.wormix.objects
{
   import flash.display.DisplayObject;
   import flash.display.MovieClip;
   import flash.events.Event;
   import flash.geom.Point;
   import flash.text.TextField;
   import ru.pragmatix.flox.gui.controls.hint.Hintable;
   import ru.pragmatix.flox.gui.controls.hint.IHintProvider;
   import ru.pragmatix.flox.loader.ClassLoader;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.secure.ProtectedBoolean;
   import ru.pragmatix.flox.secure.ProtectedInt;
   import ru.pragmatix.flox.utils.Math2;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.events.DamageEvent;
   import ru.pragmatix.wormix.events.GameEvent;
   import ru.pragmatix.wormix.events.GameObjectHpChangeEvent;
   import ru.pragmatix.wormix.events.KilledWormEvent;
   import ru.pragmatix.wormix.events.PhysicEvent;
   import ru.pragmatix.wormix.force.Force;
   import ru.pragmatix.wormix.fx.Effects;
   import ru.pragmatix.wormix.fx.Explosions;
   import ru.pragmatix.wormix.fx.SmokeTrail;
   import ru.pragmatix.wormix.game.ai.AIArsenal;
   import ru.pragmatix.wormix.game.squad.Squad;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.loader.Classes;
   import ru.pragmatix.wormix.model.Sounds;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.items.ItemsStore;
   import ru.pragmatix.wormix.model.race.Race;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   import ru.pragmatix.wormix.model.worm.Traits;
   import ru.pragmatix.wormix.objects.abilities.HighSpeed;
   import ru.pragmatix.wormix.objects.abilities.MightyJump;
   import ru.pragmatix.wormix.objects.abilities.UnderwaterBreath;
   import ru.pragmatix.wormix.objects.attribute.AttributeJump;
   import ru.pragmatix.wormix.objects.attribute.AttributeSpeed;
   import ru.pragmatix.wormix.objects.buffs.BuffsController;
   import ru.pragmatix.wormix.objects.buffs.buff.DodgeBuff;
   import ru.pragmatix.wormix.objects.buffs.buff.RegenerationBuff;
   import ru.pragmatix.wormix.objects.buffs.debuff.PoisonDebuff;
   import ru.pragmatix.wormix.objects.graphics.AbstractGraphics;
   import ru.pragmatix.wormix.objects.graphics.CharGraphics;
   import ru.pragmatix.wormix.objects.mutators.IHintInfo;
   import ru.pragmatix.wormix.objects.sound.CharSounds;
   import ru.pragmatix.wormix.physics.CollideType;
   import ru.pragmatix.wormix.physics.PhysParams;
   import ru.pragmatix.wormix.physics.Physics;
   import ru.pragmatix.wormix.physics.RopePhysicModel;
   import ru.pragmatix.wormix.physics.WormPhysicModel;
   import ru.pragmatix.wormix.secure.CheatDetectedEvent;
   import ru.pragmatix.wormix.secure.Jammer;
   import ru.pragmatix.wormix.utils.EventMaster;
   import ru.pragmatix.wormix.weapons.Arsenal;
   import ru.pragmatix.wormix.weapons.SheepWeapon;
   import ru.pragmatix.wormix.weapons.SuperSheepWeapon;
   import ru.pragmatix.wormix.weapons.Weapon;
   import ru.pragmatix.wormix.weapons.ammo.DamageType;
   import ru.pragmatix.wormix.weapons.damage.Damager;
   
   public class Worm extends GameObject implements IHintProvider, IHintInfo
   {
      
      private static const log:Logger = new Logger(Worm);
      
      public static const LEFT:int = -1;
      
      public static const RIGHT:int = 1;
      
      public var weapon:Weapon = null;
      
      public var lastWeapon:Weapon = null;
      
      public var defaultBreathClip:String = null;
      
      protected var walkSpeed:AttributeSpeed;
      
      protected var jump:AttributeJump;
      
      private var bonusSpeed:Boolean = false;
      
      public var squad:Squad;
      
      public var record:UserProfile;
      
      private var currentHp:ProtectedInt;
      
      public var lastAngleUsed:Number = 0;
      
      private var immobile:ProtectedBoolean = new ProtectedBoolean(false);
      
      public var preventDamage:Boolean = false;
      
      public var direction:int = -1;
      
      private var freeRotate:Boolean = false;
      
      public var graphics:CharGraphics;
      
      private var trail:SmokeTrail;
      
      public var buffs:BuffsController;
      
      private var storedHat:HatRecord;
      
      private var revived:Boolean;
      
      private var noDropHat:Boolean = false;
      
      private var noDropGrave:Boolean = false;
      
      public var aiArsenal:AIArsenal = null;
      
      public var aiMissFactor:int = -1;
      
      public var aiShotPower:int = 1;
      
      private var dmgOnPrevTurn:uint = 0;
      
      private var normalPhysModel:WormPhysicModel;
      
      private var ropePhysModel:RopePhysicModel;
      
      private var hintable:Hintable;
      
      private var hint:MovieClip;
      
      private var infoDisabled:Boolean = false;
      
      public function Worm(record:UserProfile, x:int, y:int, revived:Boolean = false)
      {
         this.record = record;
         this.revived = revived;
         super();
         canBeDamaged = true;
         this.walkSpeed = new AttributeSpeed();
         this.jump = new AttributeJump(this);
         this.currentHp = new ProtectedInt(record.getTotalHp());
         this.x = x;
         this.y = y;
         this.graphics = new CharGraphics(record.getCharType());
         this.storedHat = record.getHat();
         this.graphics.setHat(record.getHat());
         this.graphics.setArtifact(record.getArtifact());
         this.graphics.breath(false);
         this.clip = this.graphics.getClip();
         this.hintable = new Hintable(clip);
         this.hintable.setHintProvider(this);
         this.buffs = new BuffsController(this);
         this.updatePhysParams();
         this.lastWeapon = Arsenal.takeBaseWeapon(record);
         addEventListener(GameEvent.DAMAGED,this.defaultOnDamaged);
         if(revived)
         {
            this.graphics.colorTransform(-50,-50,-50);
         }
         Application.curScene.gameObjectProcessor.manage(this);
      }
      
      override protected function initPhysicModel() : void
      {
         physParams = new PhysParams();
         physParams.addForce(Force.GRAVITY);
         physParams.collidesWith = PhysParams.COLLIDES_WITH_GROUND;
         physParams.collideType = CollideType.WORM;
         physParams.optimizedMovement = false;
         physParams.sinkable = true;
         this.ropePhysModel = new RopePhysicModel(physParams);
         this.ropePhysModel.addEventListener(GameEvent.SINK,defaultOnSink);
         this.normalPhysModel = new WormPhysicModel(physParams);
         this.normalPhysModel.addEventListener(PhysicEvent.STABLE,this.onStable);
         this.normalPhysModel.addEventListener(PhysicEvent.UNSTABLE,this.onUnstable);
         this.normalPhysModel.addEventListener(PhysicEvent.FALLING,this.onFalling);
         physicModel = this.normalPhysModel;
      }
      
      public function setNormalPhysicModel() : void
      {
         var freePoint:Point = null;
         var _x:int = this.x;
         var _y:int = this.y;
         physicModel = this.normalPhysModel;
         x = _x;
         y = _y;
         setClipShift(0,0);
         if(Physics.wormHitTest(x,y))
         {
            freePoint = Physics.nearestFreePoint(x,y,Physics.wormHitTest,30);
            if(freePoint != null)
            {
               x = freePoint.x;
               y = freePoint.y;
            }
            else
            {
               Damager.customDamage(null,x,y,null,0,0,Physics.WORM_HALF_HEIGHT + 1,false);
            }
         }
      }
      
      public function setRopePhysicModel(joint:Point) : void
      {
         var _x:int = this.x;
         var _y:int = this.y;
         physicModel = this.ropePhysModel;
         x = _x;
         y = _y;
         this.ropePhysModel.reset(joint,this);
      }
      
      public function updatePhysParams() : void
      {
         if(!this.bonusSpeed && (this.record.getCharType().hasAbility(HighSpeed) || this.record.getBonusStats().hasTrait(Traits.SPEED)))
         {
            this.walkSpeed.setBaseSpeed(Traits.BONUS_SPEED.value);
            this.bonusSpeed = true;
         }
         if(this.crawling != 0)
         {
            this.crawling = Math2.sign(this.crawling) * this.walkSpeed.getSpeed();
         }
         if(this.record.getBonusStats().hasTrait(Traits.MIGHTY_JUMP) || this.record.getCharType().hasAbility(MightyJump))
         {
            this.jump.setJumpPower(this.jump.curDefaultMightyJumpPower.value);
         }
         else
         {
            this.jump.setJumpPower(this.jump.curDefaultJumpPower.value);
         }
         if(this.record.getBonusStats().hasTrait(Traits.DOUBLE_JUMP))
         {
            if(this.record.getCharType().hasAbility(HighSpeed))
            {
               this.jump.setDoubleJumpPower(this.jump.curDefaultDoubleJumpHighSpeedPower.value);
            }
            else
            {
               this.jump.setDoubleJumpPower(this.jump.curDefaultDoubleJumpPower.value);
            }
         }
         else if(this.record.getCharType().hasAbility(HighSpeed))
         {
            this.jump.setDoubleJumpPower(this.jump.curDefaultJumpPower.value);
         }
         physParams.sinkable = !(this.record.getCharType().hasAbility(UnderwaterBreath) || this.record.getBonusStats().hasTrait(Traits.WATER_BREATH));
         if(this.record.getBonusStats().dodge > 0)
         {
            this.buffs.place(new DodgeBuff(this.record.getBonusStats().dodge));
         }
         if(this.record.getBonusStats().regeneration > 0)
         {
            this.buffs.place(new RegenerationBuff(this.record.getBonusStats().regeneration,-1,false,true));
         }
      }
      
      protected function onStable(e:Event) : void
      {
         physicModel.stop();
         this.updateBreathClip();
         this.freeRotate = false;
         WormPhysicModel(physicModel).rollable = false;
         this.readyGraphics();
         if(this.trail != null)
         {
            this.removeTrail();
         }
         dispatchEvent(new Event(GameEvent.WORM_STABLE));
      }
      
      protected function onUnstable(e:Event) : void
      {
         if(this.crawling != 0)
         {
            this.stopCrawling();
         }
         physicModel.stop();
         this.graphics.hideAllClips();
         if(Boolean(this.weapon))
         {
            this.weapon.deactivate();
         }
         dispatchEvent(new Event(GameEvent.WORM_UNSTABLE));
      }
      
      protected function onFalling(e:Event) : void
      {
         this.stopCrawling();
         addSpeedXY(this.direction == LEFT ? -1 : 1,0);
         EventMaster.once(this,GameEvent.WORM_STABLE,this.jump.stabilize);
      }
      
      private function onFrame(e:Event) : void
      {
         if(this.freeRotate)
         {
            if(physicModel is WormPhysicModel && this.rolling != 0)
            {
               this.graphics.getClip().rotation = 0;
            }
            else if(speed < 10 || Math.abs(speedX) < 1)
            {
               this.graphics.getClip().rotation = angle * 180 / Math.PI + 90;
            }
            else
            {
               this.graphics.getClip().rotation = this.graphics.getClip().rotation + Math2.sign(speedX) * speed;
            }
         }
         this.jump.onFrame();
      }
      
      public function setHat(drop:HatDropped) : void
      {
         var hat:HatRecord = ItemsStore.getHatById(drop.hatId,this.record.getCharType());
         if(Singletons.btc.gameMaster.playersTurn() && drop.getOwner().squad.getTeamId() != this.squad.getTeamId())
         {
            Singletons.btc.battleStats.incrementHatAchieve(drop.getId());
         }
         this.dropHat(this.direction * -3,-6);
         var oldHp:uint = uint(this.currentHp.value);
         var rate:Number = oldHp / (this.record.getHp() + this.record.getBonusStats().hp);
         this.currentHp.value = Math.max(Math.floor((this.record.getHp() + this.record.getBonusStats().hp) * rate),1);
         var delta:int = this.currentHp.value - oldHp;
         dispatchEvent(new GameObjectHpChangeEvent(this,delta));
         this.record.setHat(hat);
         this.graphics.setHat(hat);
         this.updatePhysParams();
         this.updateBreathClip();
         drop.disposed = true;
      }
      
      override public function nextState() : void
      {
         super.nextState();
         this.onFrame(null);
         if(stable && this.crawling != 0)
         {
            this.graphicsOnNextState();
         }
      }
      
      protected function graphicsOnNextState() : void
      {
         var testX:int = 0;
         var testY:int = 0;
         if(this.record.getCharType() == Race.WORM.getRaceRecord())
         {
            testX = x - 10 * this.direction;
            testY = y + Physics.WORM_HALF_HEIGHT;
            if(Physics.getGroundDiff(testX,testY) > 4)
            {
               this.graphics.goDown();
            }
            else if(Physics.getGroundDiff(testX,testY) < -4)
            {
               this.graphics.goUp();
            }
            else
            {
               this.graphics.go();
            }
         }
         else
         {
            this.graphics.go();
         }
      }
      
      public function updateBreathClip() : void
      {
         this.graphics.getClip().rotation = 0;
         this.actionBreathClip(this.buffs.hasBuff(PoisonDebuff.NAME) || this.currentHp.value < 30);
      }
      
      protected function actionBreathClip(isSad:Boolean) : void
      {
         var testX:int = 0;
         var testY:int = 0;
         if(this.defaultBreathClip != null)
         {
            this.graphics[this.defaultBreathClip]();
         }
         else if(this.record.getCharType().configName == "worm")
         {
            testX = x - this.direction * 10;
            testY = y + Physics.WORM_HALF_HEIGHT;
            if(Physics.getGroundDiff(testX,testY) > 4)
            {
               this.graphics.breathDown(isSad);
            }
            else if(Physics.getGroundDiff(testX,testY) < -4)
            {
               this.graphics.breathUp(isSad);
            }
            else
            {
               this.graphics.breath(isSad);
            }
         }
         else
         {
            this.graphics.breath(isSad);
         }
      }
      
      public function attachClip(mc:MovieClip, dx:int = 0, dy:int = 0, needRotate:Boolean = true) : void
      {
         if(!disposed)
         {
            this.graphics.attachClip(AbstractGraphics.ONSTABLE,mc,dx,dy,stable,needRotate);
         }
      }
      
      public function detachClip(mc:MovieClip) : void
      {
         this.graphics.detachClip(AbstractGraphics.ONSTABLE,mc);
      }
      
      public function attachPermanent(mc:MovieClip, dx:int = 0, dy:int = 0, needRotate:Boolean = true) : void
      {
         if(!disposed)
         {
            this.graphics.attachClip(AbstractGraphics.PERMANENT,mc,dx,dy,true,needRotate);
         }
      }
      
      public function detachPermanent(mc:MovieClip) : void
      {
         this.graphics.detachClip(AbstractGraphics.PERMANENT,mc);
      }
      
      public function get rolling() : int
      {
         return WormPhysicModel(physicModel).rolling;
      }
      
      public function set rolling(val:int) : void
      {
         WormPhysicModel(physicModel).rolling = val;
      }
      
      public function get crawling() : int
      {
         return physicModel is WormPhysicModel ? WormPhysicModel(physicModel).crawling : 0;
      }
      
      public function set crawling(val:int) : void
      {
         WormPhysicModel(physicModel).crawling = val;
      }
      
      public function get keepUnstable() : Boolean
      {
         return physicModel.keepUnstable;
      }
      
      public function set keepUnstable(val:Boolean) : void
      {
         physicModel.keepUnstable = val;
      }
      
      override public function swapClip(clip:MovieClip) : void
      {
      }
      
      override public function defaultOnDamaged(e:DamageEvent) : void
      {
         if(e.power > 800 && Singletons.btc.gameMaster.playersTurn())
         {
            Application.curScene.dispatchEvent(new CheatDetectedEvent(Jammer.DAMAGE_HACK_BAN_TYPE,"damage hacked"));
            return;
         }
         super.defaultOnDamaged(e);
         if(this.preventDamage)
         {
            return;
         }
         var oldRolling:int = 0;
         if(physicModel is WormPhysicModel)
         {
            this.attachTrail();
            if(Boolean(this.crawling))
            {
               this.stopCrawling();
            }
            if(this.rolling != 0)
            {
               oldRolling = this.rolling;
               this.rolling = 0;
            }
            if(speedY > -2 + Physics.EPSILON)
            {
               setSpeedXY(speedX,-2);
            }
            if(Math.abs(speedX) < 0.2 + Physics.EPSILON)
            {
               setSpeedXY(Math2.sign(speedX) * 0.2,speedY);
            }
         }
         if(Singletons.btc.getShowBlood() && e.source.damageType != DamageType.FIRE)
         {
            Effects.bloodsSplatter(x,y,Math.atan2(e.source.y - y,e.source.x - x));
         }
         var dmg:uint = Math.ceil(e.power);
         var color:String = Boolean(e.source.owner) && e.source.owner.squad.getTeamId() == this.squad.getTeamId() ? Effects.TEXT_COLOR_ORANGE : Effects.TEXT_COLOR_RED;
         if(dmg > 0)
         {
            Effects.tweenedText(dmg.toString(),color,this.x,this.y - 10,20);
         }
         if(dmg > this.currentHp.value)
         {
            dmg = uint(this.currentHp.value);
         }
         this.changeHp(-dmg);
         this.dmgOnPrevTurn += dmg;
         if(this.currentHp.value <= 0 && dmg != 0)
         {
            if(Singletons.btc.gameMaster.activeWorm != this)
            {
               Singletons.btc.gameMaster.activeWorm.dispatchEvent(new KilledWormEvent(this));
            }
            disposed = true;
         }
         if(physicModel is WormPhysicModel)
         {
            if(!this.freeRotate)
            {
               this.graphics.getClip().rotation = 0;
               this.graphics.damaged();
               if(Math.abs(speed) > 3)
               {
                  this.rotateUntilStable();
               }
            }
         }
         this.jump.canDoubleJump.value = false;
         CharSounds.playDamagedSound(this);
      }
      
      public function disableDrop() : void
      {
         this.noDropHat = true;
         this.noDropGrave = true;
      }
      
      public function disableDropGrave() : void
      {
         this.noDropGrave = true;
      }
      
      public function disableDropHat() : void
      {
         this.noDropHat = true;
      }
      
      override public function defaultOnDisposed(e:Event) : void
      {
         var delta:int = 0;
         var grave:Grave = null;
         var allies:Array = null;
         Application.curScene.removeEventListener(GameEvent.FRAME,this.onFrame);
         if(this.currentHp.value > 0)
         {
            delta = -this.currentHp.value;
            this.changeHp(delta);
         }
         this.hintable.removeHintProvider();
         if(this.weapon != null)
         {
            this.weapon.detach();
         }
         this.graphics.hideAllClips();
         this.buffs.clearAll();
         if(!Application.curScene.gameObjectProcessor.isShuttingDown() && !this.noDropGrave)
         {
            if(!this.revived)
            {
               grave = new Grave(this.record.getGraveType(this.record.getCharType()),x,y);
               grave.setSpeedXY(speedX,speedY);
            }
            else
            {
               Effects.explode(Explosions.LARGE,x,y,50,Sounds.GROUND_LOOP);
            }
            this.spawnRaceSoul(this.record,x,y);
            Singletons.btc.battleStats.incrementOnUnitDisposed(this);
         }
         if(Boolean(Singletons.btc.gameMaster))
         {
            allies = Singletons.btc.gameMaster.getAllies(this,false);
            if(allies.length == 0)
            {
               if(this.weapon != null)
               {
                  this.weapon.detach();
               }
               Singletons.btc.gameMaster.nextTurnOnStable();
            }
            if(!this.noDropHat)
            {
               this.dropHat(-speedX,1.5 * speedY);
            }
         }
         this.record.setHat(this.storedHat);
         super.defaultOnDisposed(e);
      }
      
      private function spawnRaceSoul(p:UserProfile, xPos:int, yPos:int) : void
      {
         if(!sunken && lastDamager != null && lastDamager.damageType == DamageType.FIRE)
         {
            Effects.createBurningSoul(xPos,yPos);
         }
         else
         {
            Effects.createSoul(ClassLoader.getClass(p.getSoulType(p.getCharType())),xPos,yPos,10);
         }
      }
      
      public function dropHat(speedX:int, speedY:int) : void
      {
         var hatY:int = 0;
         var hat:HatDropped = null;
         if(Boolean(this.record.getHat()))
         {
            hatY = Application.curScene.hitTest(x,y - Physics.WORM_HALF_HEIGHT) ? y : int(y - Physics.WORM_HALF_HEIGHT);
            hat = new HatDropped(this,this.record.getHat(),x,hatY);
            hat.setSpeedXY(speedX,speedY);
         }
         if(this.buffs.hasBuff(RegenerationBuff.NAME_START_TURN))
         {
            this.buffs.clearBuff(RegenerationBuff.NAME_START_TURN);
         }
         if(this.buffs.hasBuff(DodgeBuff.NAME))
         {
            this.buffs.clearBuff(DodgeBuff.NAME);
         }
         if(this.bonusSpeed && this.record.getBonusStats().hasTrait(Traits.SPEED))
         {
            this.bonusSpeed = false;
            this.walkSpeed.setBaseSpeed(-Traits.BONUS_SPEED.value);
         }
      }
      
      override public function get height() : int
      {
         return Physics.WORM_HEIGHT;
      }
      
      override public function get width() : int
      {
         return Physics.WORM_WIDTH;
      }
      
      public function turnRight() : void
      {
         this.turn(RIGHT);
      }
      
      public function turnLeft() : void
      {
         this.turn(LEFT);
      }
      
      protected function turn(newDirection:int) : void
      {
         if(this.direction != newDirection)
         {
            this.graphics.getClip().rotation = 0;
            if(newDirection == LEFT)
            {
               this.graphics.turnLeft();
            }
            else
            {
               this.graphics.turnRight();
            }
            this.direction = newDirection;
         }
      }
      
      public function rotateUntilStable(roll:Boolean = true) : void
      {
         var self:Worm = null;
         this.freeRotate = true;
         WormPhysicModel(physicModel).rollable = roll;
         self = this;
         EventMaster.once(this,GameEvent.WORM_STABLE,function(e:Event):void
         {
            self.freeRotate = false;
            WormPhysicModel(self.physicModel).rollable = false;
            updateBreathClip();
         });
      }
      
      public function changeWeapon(id:uint) : void
      {
         var forbiddenIds:Array = [70,77,78,79];
         if(!disposed)
         {
            if(this.weapon != null)
            {
               this.weapon.detach();
            }
            if(forbiddenIds.indexOf(id) != -1)
            {
               Application.curScene.dispatchEvent(new CheatDetectedEvent(Jammer.CHEAT_ENGINE_BAN_TYPE,"forbidden weapons"));
               return;
            }
            this.weapon = Arsenal.take(id);
            this.weapon.attach(this);
         }
      }
      
      public function reattachWeapon() : void
      {
         var lastId:uint = 0;
         if(this.lastWeapon != null)
         {
            lastId = uint(Weapon(this.lastWeapon).recordId);
            if(Boolean(Singletons.btc.gameMaster.getCurrentProfile().getWeaponBackpack().getCountItem(lastId)))
            {
               this.changeWeapon(lastId);
            }
         }
      }
      
      public function attachTrail() : void
      {
         if(this.trail == null)
         {
            this.trail = Effects.attachSmokeTrail(this,Explosions.SMALL,10,2);
         }
      }
      
      private function removeTrail() : void
      {
         if(this.trail != null)
         {
            this.trail.remove();
            this.trail = null;
         }
      }
      
      public function morphToBull() : void
      {
         this.graphics.setChar(Race.BULL.getRaceRecord());
         if(Boolean(this.weapon) && !(this.weapon is SheepWeapon || this.weapon is SuperSheepWeapon))
         {
            this.weapon.getRenderer().deactivate(this);
            this.weapon.getRenderer().activate(this);
         }
      }
      
      public function get hp() : int
      {
         return this.currentHp.value;
      }
      
      public function changeHp(delta:int) : void
      {
         this.currentHp.value += delta;
         dispatchEvent(new GameObjectHpChangeEvent(this,delta));
      }
      
      public function isRevived() : Boolean
      {
         return this.revived;
      }
      
      public function getWalkSpeed() : AttributeSpeed
      {
         return this.walkSpeed;
      }
      
      public function getJump() : AttributeJump
      {
         return this.jump;
      }
      
      private function crawl(newDirection:int) : void
      {
         if(stable && !this.crawling && !this.immobile.value)
         {
            this.graphics.go();
         }
         if(stable && (newDirection == LEFT && this.crawling >= 0 || newDirection == RIGHT && this.crawling <= 0))
         {
            if(Boolean(this.weapon))
            {
               this.weapon.deactivate();
            }
            this.turn(newDirection);
            this.graphics.hideAllClips();
            if(!this.immobile.value)
            {
               this.crawling = Math2.sign(newDirection) * this.walkSpeed.getSpeed();
            }
            else
            {
               this.graphics.showAllClips();
               this.updateBreathClip();
               if(Boolean(this.weapon))
               {
                  this.weapon.activate();
               }
            }
         }
      }
      
      public function crawlRight() : void
      {
         this.crawl(RIGHT);
      }
      
      public function crawlLeft() : void
      {
         this.crawl(LEFT);
      }
      
      public function stopCrawling() : void
      {
         if(!disposed && this.crawling != 0)
         {
            this.crawling = 0;
            this.updateBreathClip();
            this.readyGraphics();
         }
      }
      
      protected function readyGraphics() : void
      {
         if(Boolean(this.weapon))
         {
            this.weapon.activate();
         }
         this.graphics.showAllClips();
      }
      
      public function getHint() : DisplayObject
      {
         if(!this.hint)
         {
            this.hint = ClassLoader.getNewInstance(Classes.WORM_HINT);
            TextField(this.hint.getChildByName("levelLabel")).text = Messages.LEVEL_LABEL;
            TextField(this.hint.getChildByName("armorLabel")).text = Messages.ARMOR;
            TextField(this.hint.getChildByName("attackLabel")).text = Messages.ATTACK;
            TextField(this.hint.getChildByName("raceLabel")).text = Messages.RACE;
         }
         TextField(this.hint.getChildByName("level")).text = this.record.getLevel().toString();
         TextField(this.hint.getChildByName("armor")).text = UserProfileHelper.getValueIncludeStats(this.record.getArmor(),this.record.getBonusStats().armor);
         TextField(this.hint.getChildByName("attack")).text = UserProfileHelper.getValueIncludeStats(this.record.getAttack(),this.record.getBonusStats().attack);
         TextField(this.hint.getChildByName("race")).text = this.record.getCharType().getName();
         return this.hint;
      }
      
      public function disableInfo() : void
      {
         this.infoDisabled = true;
      }
      
      public function enableInfo() : void
      {
         this.infoDisabled = false;
      }
      
      public function isInfoDisabled() : Boolean
      {
         return this.infoDisabled;
      }
      
      public function resetDmgOnPrevTurn() : void
      {
         this.dmgOnPrevTurn = 0;
      }
      
      public function getDmgOnPrevTurn() : uint
      {
         return this.dmgOnPrevTurn;
      }
      
      public function getNameHintInfoPanel() : String
      {
         return this.record.getCharName();
      }
      
      public function getSquadNumber() : uint
      {
         return this.squad.getNumber();
      }
      
      public function isImmobile() : Boolean
      {
         return this.immobile.value;
      }
      
      public function setImmobile(val:Boolean) : void
      {
         this.immobile.value = val;
      }
   }
}

