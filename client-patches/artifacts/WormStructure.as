package ru.pragmatix.wormix.messages.structures
{
   import ru.pragmatix.wormix.model.user.UserProfile;
   
   public class WormStructure
   {
      
      public var ownerId:uint;
      
      public var socialOwnerId:String;
      
      public var armor:int;
      
      public var attack:int;
      
      public var level:int;
      
      public var experience:int;
      
      public var hat:int;
      
      // Equipped artifact (item id 2001+), 0 = none. Sent by the server after socialOwnerId.
      public var artifact:int;
      
      public function WormStructure()
      {
         super();
      }
      
      public function initFromProfile(record:UserProfile) : void
      {
         this.ownerId = record.getId();
         this.socialOwnerId = record.getSocialId();
         this.armor = record.getArmor();
         this.attack = record.getAttack();
         this.level = record.getLevel();
         this.experience = record.getExperience();
      }
      
      public function toString() : String
      {
         return "WormStructure{ownerId=" + String(this.ownerId) + ",socialOwnerId=" + String(this.socialOwnerId) + ",armor=" + String(this.armor) + ",attack=" + String(this.attack) + ",level=" + String(this.level) + ",experience=" + String(this.experience) + ",hat=" + String(this.hat) + ",artifact=" + String(this.artifact) + "}";
      }
   }
}

