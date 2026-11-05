package ru.pragmatix.wormix.model.worm
{
   import ru.pragmatix.flox.secure.ProtectedInt;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.objects.abilities.IAbility;
   import ru.pragmatix.wormix.objects.abilities.StatBonus;
   import ru.pragmatix.wormix.objects.abilities.StatBonusAbility;
   
   public class BonusStats
   {
      
      private var _hp:ProtectedInt = new ProtectedInt(0);
      
      private var _armor:ProtectedInt = new ProtectedInt(0);
      
      private var _attack:ProtectedInt = new ProtectedInt(0);
      
      private var _dmgReduction:ProtectedInt = new ProtectedInt(0);
      
      private var _poison:ProtectedInt = new ProtectedInt(0);
      
      private var _fireDamageBonus:ProtectedInt = new ProtectedInt(0);
      
      private var _fireDamageReduction:ProtectedInt = new ProtectedInt(0);
      
      private var _leech:ProtectedInt = new ProtectedInt(0);
      
      private var _dodge:ProtectedInt = new ProtectedInt(0);
      
      private var _regeneration:ProtectedInt = new ProtectedInt(0);
      
      private var traits:Array = [];
      
      public function BonusStats()
      {
         super();
      }
      
      public function update(p:UserProfile) : void
      {
         var ability:IAbility = null;
         this.reset();
         for each(ability in p.getCharType().bonus)
         {
            if(ability is StatBonusAbility)
            {
               this.addCharBonus(p,StatBonusAbility(ability).getStatBonus());
            }
         }
         if(Boolean(p.getHat()))
         {
            this.addHatBonus(p.getHat());
         }
         // The artifact's bonuses add up with the hat's.
         if(Boolean(p.getArtifact()))
         {
            this.addHatBonus(p.getArtifact());
         }
      }
      
      private function reset() : void
      {
         this._hp.value = 0;
         this._armor.value = 0;
         this._attack.value = 0;
         this._dmgReduction.value = 0;
         this._poison.value = 0;
         this._fireDamageBonus.value = 0;
         this._fireDamageReduction.value = 0;
         this._leech.value = 0;
         this._dodge.value = 0;
         this._regeneration.value = 0;
         this.traits = [];
      }
      
      private function addCharBonus(p:UserProfile, bonus:StatBonus) : void
      {
         this._hp.value += bonus.hp;
         this._hp.value += Math.round(p.getHp() * bonus.hpPct / 100);
         this._armor.value += bonus.armor;
         this._armor.value += Math.round(p.getArmor() * bonus.armorPct / 100);
         this._attack.value += bonus.attack;
         this._attack.value += Math.round(p.getAttack() * bonus.attackPct / 100);
         this._dmgReduction.value += bonus.dmgReduction;
      }
      
      private function addHatBonus(item:HatRecord) : void
      {
         this._hp.value += item.hp;
         this._armor.value += item.armor;
         this._attack.value += item.attack;
         this._poison.value += item.poison;
         this._fireDamageBonus.value += item.fireDamageBonus;
         this._fireDamageReduction.value += item.fireDamageReduction;
         this._leech.value += item.leech;
         this._dodge.value += item.dodge;
         this._regeneration.value += item.regeneration;
         this.traits = this.traits.concat(item.getTraits());
      }
      
      public function get hp() : uint
      {
         return this._hp.value;
      }
      
      public function get armor() : uint
      {
         return this._armor.value;
      }
      
      public function get attack() : uint
      {
         return this._attack.value;
      }
      
      public function get poison() : uint
      {
         return this._poison.value;
      }
      
      public function get fireDamageBonus() : uint
      {
         return this._fireDamageBonus.value;
      }
      
      public function get fireDamageReduction() : uint
      {
         return this._fireDamageReduction.value;
      }
      
      public function get leech() : uint
      {
         return this._leech.value;
      }
      
      public function hasTrait(trait:uint) : Boolean
      {
         return this.traits.indexOf(trait) != -1;
      }
      
      public function get dmgReduction() : uint
      {
         return this._dmgReduction.value;
      }
      
      public function get dodge() : uint
      {
         return this._dodge.value;
      }
      
      public function get regeneration() : uint
      {
         return this._regeneration.value;
      }
   }
}

