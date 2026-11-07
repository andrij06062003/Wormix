package ru.pragmatix.wormix.model.items
{
   import config.craft.UpgradeConfig;
   import ru.pragmatix.flox.core.Arrays;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.wormix.model.race.RaceRecord;
   import ru.pragmatix.wormix.model.race.RaceStore;
   import ru.pragmatix.wormix.model.user.UserProfile;
   
   public class ItemsStore
   {
      
      private static var instance:ItemsStore;
      
      private static var log:Logger = new Logger(ItemRecord);
      
      private var weapons:Object = new Object();
      
      private var hats:Object = new Object();
      
      public function ItemsStore()
      {
         super();
         instance = this;
      }
      
      public static function getInstance() : ItemsStore
      {
         if(!instance)
         {
            instance = new ItemsStore();
         }
         return instance;
      }
      
      public static function getWeapons() : Object
      {
         return ItemsStore.getInstance().weapons;
      }
      
      public static function getNewItemsByLevel(p:UserProfile) : Array
      {
         var weapon:WeaponRecord = null;
         var hat:HatRecord = null;
         var items:Array = new Array();
         for each(weapon in getWeapons())
         {
            if(!weapon.hideInShop && weapon.getRequiredLevel() == p.getLevel() && !weapon.getDropWeight() && weapon.getId() < UpgradeConfig.START_ID)
            {
               items.push(weapon);
            }
         }
         for each(hat in getHatsForChar(p.getCharType()))
         {
            if(!hat.hideInShop && hat.getRequiredLevel() == p.getLevel())
            {
               items.push(hat);
            }
         }
         return items;
      }
      
      public static function getDroppableWeapons() : Object
      {
         var id:String = null;
         var rec:WeaponRecord = null;
         var result:Object = {};
         for(id in getWeapons())
         {
            rec = WeaponRecord(getWeapons()[id]);
            if(!rec.getInfinite() && Boolean(rec.getDropWeight()))
            {
               result[id] = rec;
            }
         }
         return result;
      }
      
      public static function getWeaponById(id:int) : WeaponRecord
      {
         return ItemsStore.getWeapons()[id];
      }
      
      public static function resetWeapons() : void
      {
         var rec:WeaponRecord = null;
         for each(rec in getWeapons())
         {
            rec.resetWeapon();
         }
      }
      
      public static function addHat(hat:HatRecord) : void
      {
         if(!ItemsStore.getHats()[hat.getCharName()])
         {
            ItemsStore.getHats()[hat.getCharName()] = {};
         }
         ItemsStore.getHats()[hat.getCharName()][hat.getId()] = hat;
      }
      
      public static function getHats() : Object
      {
         return ItemsStore.getInstance().hats;
      }
      
      public static function getHatsForChar(ch:RaceRecord) : Array
      {
         var result:Array = [];
         if(ch.noHats)
         {
            // No hats for this race, but artifacts are held in the hand: those still fit.
            return getArtifacts();
         }
         var hats:Object = {};
         Arrays.copyMap(ItemsStore.getHats()[RaceStore.DEFAULT_CHAR_CONFIG_NAME],hats);
         if(Boolean(ItemsStore.getHats()[ch.configName]))
         {
            Arrays.copyMap(ItemsStore.getHats()[ch.configName],hats);
         }
         return Arrays.getValuesAsArray(hats);
      }
      
      public static function getHatById(id:int, ch:RaceRecord) : HatRecord
      {
         var hat:HatRecord = null;
         if(!id)
         {
            return null;
         }
         if(ch.noHats)
         {
            if(id == 1027 || id == 1034)
            {
               return null;
            }
         }
         if(Boolean(ItemsStore.getHats()[ch.configName]))
         {
            hat = ItemsStore.getHats()[ch.configName][id];
         }
         var raceHats:Object = ItemsStore.getHats()[RaceStore.DEFAULT_CHAR_CONFIG_NAME];
         if(!hat && Boolean(raceHats))
         {
            hat = raceHats[id];
         }
         return hat;
      }
      
      public static function getArtifactById(id:int) : HatRecord
      {
         var raceHats:Object = ItemsStore.getHats()[RaceStore.DEFAULT_CHAR_CONFIG_NAME];
         if(id <= HatRecord.ARTIFACT_START_ID || !raceHats)
         {
            return null;
         }
         var artifact:HatRecord = raceHats[id];
         return Boolean(artifact) && artifact.isArtifact ? artifact : null;
      }
      
      public static function getArtifacts() : Array
      {
         var hat:HatRecord = null;
         var result:Array = [];
         for each(hat in ItemsStore.getHats()[RaceStore.DEFAULT_CHAR_CONFIG_NAME])
         {
            if(hat.isArtifact)
            {
               result.push(hat);
            }
         }
         return result;
      }
      
      public static function getItemById(id:int, ch:RaceRecord) : ItemRecord
      {
         var item:ItemRecord = getWeaponById(id);
         if(!item)
         {
            item = getHatById(id,ch);
         }
         return item;
      }
   }
}

