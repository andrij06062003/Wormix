package ru.pragmatix.wormix.model.items
{
   import ru.pragmatix.flox.secure.ProtectedInt;
   import ru.pragmatix.wormix.model.race.RaceRecord;
   import ru.pragmatix.wormix.model.worm.Traits;
   
   public class HatRecord extends ItemRecord
   {
      
      private var _hp:ProtectedInt;
      
      private var _armor:ProtectedInt;
      
      private var _attack:ProtectedInt;
      
      private var _requiredRating:ProtectedInt;
      
      private var _poison:ProtectedInt;
      
      private var _fireDamageBonus:ProtectedInt;
      
      private var _fireDamageReduction:ProtectedInt;
      
      private var _leech:ProtectedInt;
      
      private var _dodge:ProtectedInt;
      
      private var _regeneration:ProtectedInt;
      
      private var traits:Array = [];
      
      private var charConfigName:String;
      
      public var hideHair:Boolean;
      
      public var hideEars:Boolean;
      
      private var _scale:Number;
      
      public var oneDay:Boolean;
      
      // Artifacts (ported from 1.25.0): an off-hand item with its own slot next to the hat.
      // They are kept with the hats ("default" char config), so the shop and the wardrobe list them too.
      public static const ARTIFACT_START_ID:int = 2000;
      
      public var isArtifact:Boolean;
      
      public var handFront:Boolean = true;
      
      public var handVisible:Boolean = true;
      
      public var hideHands:Boolean;
      
      // 1.25.0 bonuses that 1.05.0 has no effect for yet (shockDamage, penetration, ...): name -> value.
      public var extraBonuses:Object = {};
      
      private static const EXTRA_BONUSES:Array = ["shockDamage","shockReduction","airDamage","airReduction","frostDamage","frostReduction","penetration","squadArmor","squadAttack","fortitude","healBonus","slowProtection","waterProtection","poisonProtection"];
      
      public function HatRecord()
      {
         super();
      }
      
      override public function initFromConfig(json:Object) : void
      {
         var bonusName:String = null;
         super.initFromConfig(json);
         if(Boolean(json.artifact))
         {
            this.isArtifact = true;
            // 1.25.0 names for the bonuses 1.05.0 already has.
            if(Boolean(json.fireDamage))
            {
               this._fireDamageBonus = new ProtectedInt(json.fireDamage);
            }
            if(Boolean(json.fireReduction))
            {
               this._fireDamageReduction = new ProtectedInt(json.fireReduction);
            }
            if(Boolean(json.poisonDamage))
            {
               // 1.25.0 gives poison in % (20, 30), 1.05.0 in levels of 20% each.
               this._poison = new ProtectedInt(Math.max(1,Math.round(json.poisonDamage / 20)));
            }
            if(json.handFront === false)
            {
               this.handFront = false;
            }
            if(json.handVisible === false)
            {
               this.handVisible = false;
            }
            if(Boolean(json.hideHands))
            {
               this.hideHands = true;
            }
            for each(bonusName in EXTRA_BONUSES)
            {
               if(Boolean(json[bonusName]))
               {
                  this.extraBonuses[bonusName] = json[bonusName] === true ? 1 : Number(json[bonusName]);
               }
            }
         }
         if(Boolean(json.hp))
         {
            this._hp = new ProtectedInt(json.hp);
         }
         if(Boolean(json.armor))
         {
            this._armor = new ProtectedInt(json.armor);
         }
         if(Boolean(json.attack))
         {
            this._attack = new ProtectedInt(json.attack);
         }
         if(Boolean(json.poison))
         {
            this._poison = new ProtectedInt(json.poison);
         }
         if(Boolean(json.fireDamageBonus))
         {
            this._fireDamageBonus = new ProtectedInt(json.fireDamageBonus);
         }
         if(Boolean(json.fireDamageReduction))
         {
            this._fireDamageReduction = new ProtectedInt(json.fireDamageReduction);
         }
         if(Boolean(json.leech))
         {
            this._leech = new ProtectedInt(json.leech);
         }
         if(Boolean(json.dodge))
         {
            this._dodge = new ProtectedInt(json.dodge);
         }
         if(Boolean(json.regeneration))
         {
            this._regeneration = new ProtectedInt(json.regeneration);
         }
         if(Boolean(json.speed))
         {
            this.traits.push(Traits.SPEED);
         }
         if(Boolean(json.poisonProtection) && !this.isArtifact)
         {
            this.traits.push(Traits.POISON_PROTECTION);
         }
         if(Boolean(json.doubleJump))
         {
            this.traits.push(Traits.DOUBLE_JUMP);
         }
         if(Boolean(json.waterBreath))
         {
            this.traits.push(Traits.WATER_BREATH);
         }
         if(Boolean(json.frostImmunity))
         {
            this.traits.push(Traits.FROST_IMMUNITY);
         }
         if(Boolean(json.frostProtection))
         {
            this.traits.push(Traits.FROST_PROTECTION);
         }
         if(Boolean(json.mightyJump))
         {
            this.traits.push(Traits.MIGHTY_JUMP);
         }
         if(Boolean(json.requiredRating))
         {
            this._requiredRating = new ProtectedInt(json.requiredRating);
         }
         if(Boolean(json.charName))
         {
            this.charConfigName = json.charName;
         }
         if(Boolean(json.scale))
         {
            this._scale = json.scale;
         }
         if(Boolean(json.hideEars))
         {
            this.hideEars = json.hideEars;
         }
         if(Boolean(json.hideHair))
         {
            this.hideHair = json.hideHair;
         }
         if(Boolean(json.oneDay))
         {
            this.oneDay = json.oneDay;
         }
         setInfinite(true);
      }
      
      public function get hp() : uint
      {
         return Boolean(this._hp) ? uint(this._hp.value) : 0;
      }
      
      public function get armor() : uint
      {
         return Boolean(this._armor) ? uint(this._armor.value) : 0;
      }
      
      public function get attack() : uint
      {
         return Boolean(this._attack) ? uint(this._attack.value) : 0;
      }
      
      public function get poison() : uint
      {
         return Boolean(this._poison) ? uint(this._poison.value) : 0;
      }
      
      public function get fireDamageBonus() : uint
      {
         return Boolean(this._fireDamageBonus) ? uint(this._fireDamageBonus.value) : 0;
      }
      
      public function get fireDamageReduction() : uint
      {
         return Boolean(this._fireDamageReduction) ? uint(this._fireDamageReduction.value) : 0;
      }
      
      public function get leech() : uint
      {
         return Boolean(this._leech) ? uint(this._leech.value) : 0;
      }
      
      public function get dodge() : uint
      {
         return Boolean(this._dodge) ? uint(this._dodge.value) : 0;
      }
      
      public function get regeneration() : Number
      {
         return Boolean(this._regeneration) ? this._regeneration.value : 0;
      }
      
      public function getTraits() : Array
      {
         return this.traits;
      }
      
      public function get requiredRating() : uint
      {
         return Boolean(this._requiredRating) ? uint(this._requiredRating.value) : 0;
      }
      
      public function canBeUsed(ch:RaceRecord) : Boolean
      {
         return this.isArtifact || !ch.noHats;
      }
      
      public function getCharName() : String
      {
         return this.charConfigName;
      }
      
      public function getScale() : Number
      {
         return Boolean(this._scale) ? this._scale : 1;
      }
      
      public function getIsHideEars() : Boolean
      {
         return this.hideEars ? this.hideEars : false;
      }
      
      public function generateJavaCodeToTrace() : void
      {
         trace("entities.add(createStuff(" + getId() + ",\"" + getName() + "\"," + (getIngamePrice() == 0 ? "99999" : String(getIngamePrice())) + "," + (getRealPrice() == 0 ? "99999" : String(getRealPrice())) + "," + getRequiredLevel() + "));");
      }
   }
}

