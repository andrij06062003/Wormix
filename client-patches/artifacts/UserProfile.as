package ru.pragmatix.wormix.model.user
{
   import config.HatsConfig;
   import config.craft.UpgradeConfig;
   import ru.pragmatix.flox.logger.Logger;
   import ru.pragmatix.flox.model.MoneyType;
   import ru.pragmatix.flox.model.Price;
   import ru.pragmatix.flox.secure.ProtectedInt;
   import ru.pragmatix.flox.social.model.IUserProfile;
   import ru.pragmatix.flox.social.model.SocialUserData;
   import ru.pragmatix.flox.utils.Math2;
   import ru.pragmatix.wormix.messages.structures.RatingProfileStructure;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.messages.structures.WeaponStructure;
   import ru.pragmatix.wormix.messages.structures.WormStructure;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.items.ItemRecord;
   import ru.pragmatix.wormix.model.items.ItemsStore;
   import ru.pragmatix.wormix.model.items.WeaponRecord;
   import ru.pragmatix.wormix.model.items.backpack.BackpackItem;
   import ru.pragmatix.wormix.model.items.backpack.BaseBackpack;
   import ru.pragmatix.wormix.model.items.backpack.WeaponBackpack;
   import ru.pragmatix.wormix.model.items.craft.CraftStore;
   import ru.pragmatix.wormix.model.items.craft.RecipeCraftRecord;
   import ru.pragmatix.wormix.model.race.Race;
   import ru.pragmatix.wormix.model.race.RaceRecord;
   import ru.pragmatix.wormix.model.worm.BonusStats;
   import ru.pragmatix.wormix.objects.sound.SoundScheme;
   import ru.pragmatix.wormix.settings.DailyBonusPrize;
   import ru.pragmatix.wormix.settings.DailyBonusType;
   import ru.pragmatix.wormix.settings.LevelsGrid;
   import ru.pragmatix.wormix.utils.CraftUtil;
   
   public class UserProfile implements IUserProfile
   {
      
      public static var RACE_BASE:uint = 500;
      
      private static const RACE_LIMIT:int = 50;
      
      private static const STUFF_START_INDEX:int = 1000;
      
      private var id:uint;
      
      private var socialId:String;
      
      private var socialData:SocialUserData;
      
      private var ingameMoney:ProtectedInt = new ProtectedInt(0);
      
      private var realMoney:ProtectedInt = new ProtectedInt(0);
      
      public var rating:ProtectedInt = new ProtectedInt(0);
      
      public var dailyRating:ProtectedInt = new ProtectedInt(0);
      
      private var hp:ProtectedInt = new ProtectedInt(0);
      
      private var armor:ProtectedInt = new ProtectedInt(0);
      
      private var attack:ProtectedInt = new ProtectedInt(0);
      
      private var reactionRate:ProtectedInt = new ProtectedInt(0);
      
      private var level:ProtectedInt = new ProtectedInt(0);
      
      private var experience:ProtectedInt = new ProtectedInt(0);
      
      private var charName:String;
      
      private var teamName:String;
      
      private var online:Boolean;
      
      private var log:Logger = new Logger(UserProfile);
      
      private var team:Array = new Array();
      
      private var hat:HatRecord;
      
      // Equipped artifact (its own slot, next to the hat), null = none.
      private var artifact:HatRecord;
      
      private var bonusStats:BonusStats = new BonusStats();
      
      private var charType:RaceRecord;
      
      public var soundScheme:SoundScheme;
      
      private var weaponBackpack:WeaponBackpack = new WeaponBackpack();
      
      private var wardrobe:BaseBackpack = new BaseBackpack();
      
      private var reagentBackpack:BaseBackpack = new BaseBackpack();
      
      private var recipeBackpack:BaseBackpack = new BaseBackpack();
      
      public var weaponRecordList:Array = [];
      
      public function UserProfile()
      {
         super();
         this.charName = UserProfileHelper.getShortName(this);
      }
      
      public static function getZombieProfile(name:String, level:uint) : UserProfile
      {
         return custom(name,level,Race.ZOMBIE,30 + 2 * level,0,2 * level);
      }
      
      public static function custom(name:String, level:uint, race:Race, hp:uint, armor:uint, attack:uint, hatId:uint = 0, soundScheme:SoundScheme = null) : UserProfile
      {
         var hatRecord:HatRecord = null;
         var profile:UserProfile = new UserProfile();
         profile.charName = name;
         profile.hp = new ProtectedInt(hp);
         profile.armor = new ProtectedInt(armor);
         profile.attack = new ProtectedInt(attack);
         profile.level = new ProtectedInt(level);
         profile.charType = race.getRaceRecord();
         if(hatId != 0)
         {
            hatRecord = ItemsStore.getHatById(hatId,profile.charType);
            profile.setHat(hatRecord);
         }
         profile.soundScheme = soundScheme;
         return profile;
      }
      
      public function initFromStructure(userProfileStructure:UserProfileStructure) : void
      {
         var recipe:RecipeCraftRecord = null;
         var availableIds:Vector.<int> = null;
         var id:int = 0;
         var wormStructure:WormStructure = null;
         var profile:UserProfile = null;
         this.setIds(userProfileStructure.id,userProfileStructure.socialId);
         this.ingameMoney = new ProtectedInt(userProfileStructure.money);
         this.realMoney = new ProtectedInt(userProfileStructure.realMoney);
         this.rating = new ProtectedInt(userProfileStructure.rating);
         this.reactionRate = new ProtectedInt(userProfileStructure.reactionRate);
         this.online = false;
         var i:int = 0;
         this.team = [];
         for(i = 0; i < userProfileStructure.wormsGroup.length; i++)
         {
            wormStructure = WormStructure(userProfileStructure.wormsGroup[i]);
            if(wormStructure.ownerId == this.id)
            {
               this.initFromWormStructure(wormStructure);
               this.team.push(this);
            }
            else
            {
               profile = new UserProfile();
               profile.initFromWormStructure(wormStructure);
               this.team.push(profile);
            }
         }
         this.initWeaponBackpack(userProfileStructure.weaponRecordList);
         this.recipeBackpack = new BaseBackpack();
         var ids:Array = userProfileStructure.recipes;
         for(i = 0; i < ids.length; i++)
         {
            this.getRecipeBackpack().putItem(CraftStore.getRecipeById(Math.abs(ids[i])),true,Math2.sign(ids[i]));
         }
         for each(recipe in CraftStore.getRecipes())
         {
            if(recipe.availableFromStart && recipe.getRequiredLevel() <= this.getLevel() && !this.getRecipeBackpack().getItemById(recipe.getId()))
            {
               this.getRecipeBackpack().putItem(recipe,true,-1);
            }
         }
         availableIds = new Vector.<int>();
         for(i = 0; i < ids.length; i++)
         {
            if(Math2.sign(ids[i]) > 0)
            {
               availableIds.push(ids[i]);
            }
         }
         this.initRecipeBackpack(availableIds);
         for each(id in userProfileStructure.stuff)
         {
            if(id >= HatsConfig.START_ID && Boolean(ItemsStore.getHatById(id,this.getCharType())))
            {
               this.wardrobe.putItem(ItemsStore.getHatById(id,this.getCharType()),true);
            }
         }
      }
      
      public function initReagentBackpack(reagents:Vector.<int>) : void
      {
         var item:ItemRecord = null;
         var length:uint = reagents.length;
         for(var i:int = 0; i < length; i++)
         {
            item = CraftStore.getReagentById(i);
            if(Boolean(item) && reagents[i] > 0)
            {
               this.reagentBackpack.putItem(item,false,reagents[i]);
            }
         }
      }
      
      public function initWeaponBackpack(weaponRecordList:Array) : void
      {
         var weaponStructure:WeaponStructure = null;
         var item:WeaponRecord = null;
         this.weaponBackpack = new WeaponBackpack();
         this.weaponRecordList = [];
         for(var i:int = 0; i < weaponRecordList.length; i++)
         {
            weaponStructure = WeaponStructure(weaponRecordList[i]);
            item = ItemsStore.getWeaponById(weaponStructure.id);
            if(Boolean(item))
            {
               this.weaponBackpack.putItem(ItemsStore.getWeaponById(weaponStructure.id),weaponStructure.count == -1,weaponStructure.count >= 0 || weaponStructure.count == -1 && item.getMaxShots() == 0 ? weaponStructure.count : int(item.getMaxShots()));
            }
         }
      }
      
      private function initRecipeBackpack(recipeIdsList:Vector.<int>) : void
      {
         var recipe:RecipeCraftRecord = null;
         var id:int = 0;
         var levelDown:Boolean = false;
         var prevUpgradeId:int = 0;
         var length:uint = 0;
         var i:uint = 0;
         var allRecipes:Vector.<RecipeCraftRecord> = new Vector.<RecipeCraftRecord>();
         var result:Vector.<RecipeCraftRecord> = new Vector.<RecipeCraftRecord>();
         var maxLevel:int = 0;
         for each(id in recipeIdsList)
         {
            allRecipes.push(CraftStore.getRecipeById(id));
            if(allRecipes[allRecipes.length - 1].level > maxLevel)
            {
               maxLevel = allRecipes[allRecipes.length - 1].level;
            }
         }
         while(maxLevel > 0)
         {
            levelDown = true;
            for each(recipe in allRecipes)
            {
               if(recipe.level == maxLevel)
               {
                  prevUpgradeId = recipe.prevUpgradeId;
                  result.push(recipe);
                  allRecipes.splice(allRecipes.indexOf(recipe),1);
                  do
                  {
                     length = allRecipes.length;
                     for(i = 0; i < length; i++)
                     {
                        if(allRecipes[i].upgradeId == prevUpgradeId)
                        {
                           prevUpgradeId = allRecipes[i].prevUpgradeId;
                           allRecipes.splice(i,1);
                           break;
                        }
                     }
                     if(prevUpgradeId < UpgradeConfig.START_ID)
                     {
                        this.weaponBackpack.removeItem(prevUpgradeId);
                     }
                  }
                  while(prevUpgradeId >= UpgradeConfig.START_ID);
                  levelDown = false;
                  break;
               }
            }
            if(levelDown)
            {
               maxLevel--;
            }
         }
         for each(recipe in result)
         {
            CraftUtil.addUpgradeWeaponInBackpack(recipe.upgradeId,this);
         }
      }
      
      private function initFromWormStructure(wormStructure:WormStructure) : void
      {
         this.level = new ProtectedInt(wormStructure.level);
         this.hp = new ProtectedInt(LevelsGrid.getLevelHp(this.getLevel()));
         this.setArmor(wormStructure.armor);
         this.setAttack(wormStructure.attack);
         this.experience = new ProtectedInt(wormStructure.experience);
         this.init(wormStructure.ownerId,wormStructure.socialOwnerId,wormStructure.hat);
         this.artifact = ItemsStore.getArtifactById(wormStructure.artifact);
         this.bonusStats.update(this);
      }
      
      public function initFromRatingStructure(data:RatingProfileStructure) : void
      {
         var profileFromTeam:UserProfile = null;
         this.weaponBackpack = new WeaponBackpack();
         this.ingameMoney = new ProtectedInt(data.money);
         this.realMoney = new ProtectedInt(data.realMoney);
         this.rating = new ProtectedInt(data.rating);
         this.dailyRating = new ProtectedInt(data.dailyRating);
         this.reactionRate = new ProtectedInt(data.reactionRate);
         var i:int = 0;
         this.team = [];
         this.setArmor(data.armor);
         this.setAttack(data.attack);
         this.setLevel(data.level);
         this.setHp(LevelsGrid.getLevelHp(data.level));
         for(i = 0; i < data.groupCount; i++)
         {
            profileFromTeam = new UserProfile();
            profileFromTeam.setHp(LevelsGrid.getLevelHp(data.level));
            this.team.push(profileFromTeam);
         }
         this.init(data.id,data.socialId,data.hat);
      }
      
      public function initFromSocialData(socialData:SocialUserData) : void
      {
         if(Boolean(socialData) && this.getSocialId() == socialData.getSocialId())
         {
            this.socialData = socialData;
            this.charName = UserProfileHelper.getShortName(this);
            this.teamName = UserProfileHelper.getFullName(this);
         }
      }
      
      public function initTeam(socialData:Object) : void
      {
         var profile:UserProfile = null;
         for each(profile in this.team)
         {
            profile.initFromSocialData(socialData[profile.getId()]);
         }
      }
      
      public function getHat() : HatRecord
      {
         return this.hat;
      }
      
      public function getArtifact() : HatRecord
      {
         return this.artifact;
      }
      
      public function setArtifact(value:HatRecord) : void
      {
         if(Boolean(value) && !value.isArtifact)
         {
            return;
         }
         this.artifact = value;
         this.bonusStats.update(this);
      }
      
      public function setHat(value:HatRecord) : void
      {
         // Artifacts come along the hats (shop, rewards): they go into their own slot.
         if(Boolean(value) && value.isArtifact)
         {
            this.setArtifact(value);
            return;
         }
         if(this.hat == value)
         {
            return;
         }
         if(Boolean(value) && !value.canBeUsed(this.charType))
         {
            return;
         }
         this.hat = value;
         this.bonusStats.update(this);
      }
      
      public function getSocialData() : SocialUserData
      {
         return this.socialData;
      }
      
      private function init(id:uint, socialId:String, hatAndCharCode:uint) : void
      {
         var raceId:uint = 0;
         var hatId:uint = 0;
         this.setIds(id,socialId);
         this.hat = null;
         this.artifact = null;
         this.bonusStats = new BonusStats();
         if(hatAndCharCode < RACE_LIMIT)
         {
            raceId = hatAndCharCode;
            hatId = 0;
         }
         else
         {
            hatId = STUFF_START_INDEX + hatAndCharCode % RACE_BASE;
            raceId = (hatAndCharCode - STUFF_START_INDEX) / RACE_BASE;
         }
         if(raceId == 0)
         {
            raceId = 2;
         }
         this.charType = Race.getRaceRecordById(raceId);
         this.hat = ItemsStore.getHatById(hatId,this.charType);
         this.bonusStats.update(this);
      }
      
      private function setIds(id:uint, socialId:String) : void
      {
         this.id = id;
         this.socialId = Boolean(socialId) ? socialId : id.toString();
      }
      
      public function getId() : uint
      {
         return this.id;
      }
      
      public function setId(value:uint) : void
      {
         this.id = value;
      }
      
      public function getSocialId() : String
      {
         return this.socialId;
      }
      
      public function setSocialId(value:String) : void
      {
         this.socialId = value;
      }
      
      public function getIngameMoney() : uint
      {
         return this.ingameMoney.value;
      }
      
      public function setIngameMoney(value:uint) : void
      {
         this.ingameMoney.value = value;
      }
      
      public function addIngameMoney(value:uint) : void
      {
         this.ingameMoney.value += value;
      }
      
      public function getRealMoney() : uint
      {
         return this.realMoney.value;
      }
      
      public function setRealMoney(value:uint) : void
      {
         this.realMoney.value = value;
      }
      
      public function addRealMoney(value:uint) : void
      {
         this.realMoney.value += value;
      }
      
      public function getRating() : uint
      {
         return this.rating.value;
      }
      
      public function setRating(value:uint) : void
      {
         this.rating.value = value;
      }
      
      public function getDailyRating() : uint
      {
         return this.dailyRating.value;
      }
      
      public function getTeam() : Array
      {
         return this.team;
      }
      
      public function setTeam(team:Array) : void
      {
         this.team = team;
      }
      
      public function getTeamLevel() : uint
      {
         var profile:UserProfile = null;
         var level:uint = 0;
         for each(profile in this.team)
         {
            level += profile.getLevel();
         }
         return level;
      }
      
      public function getMaxLevel() : uint
      {
         var profile:UserProfile = null;
         var level:uint = 0;
         for each(profile in this.team)
         {
            if(level < profile.getLevel())
            {
               level = uint(profile.getLevel());
            }
         }
         return level;
      }
      
      public function getHp() : int
      {
         return this.hp.value;
      }
      
      public function getArmor() : int
      {
         return this.armor.value;
      }
      
      public function setArmor(armor:int) : void
      {
         this.armor.value = armor;
         if(this.armor.value < 0)
         {
            this.armor.value = 0;
         }
      }
      
      public function getAttack() : int
      {
         return this.attack.value;
      }
      
      public function setAttack(attack:int) : void
      {
         this.attack.value = attack;
      }
      
      public function getLevel() : int
      {
         return this.level.value;
      }
      
      public function setExperience(experience:int) : void
      {
         this.experience.value = experience;
      }
      
      public function getExperience() : int
      {
         return this.experience.value;
      }
      
      public function setHp(hp:int) : void
      {
         this.hp.value = hp;
      }
      
      public function getReaction() : uint
      {
         return this.reactionRate.value;
      }
      
      public function setReaction(value:uint) : void
      {
         this.reactionRate.value = value;
      }
      
      public function getOnline() : Boolean
      {
         return this.online;
      }
      
      public function setOnline(value:Boolean) : void
      {
         this.online = value;
      }
      
      public function setLevel(level:int) : void
      {
         this.level.value = level;
      }
      
      public function getCharName() : String
      {
         return this.charName;
      }
      
      public function getTeamName() : String
      {
         return this.teamName;
      }
      
      public function setTeamName(teamName:String) : void
      {
         this.teamName = teamName;
      }
      
      public function getCharType() : RaceRecord
      {
         return this.charType;
      }
      
      public function setCharType(charType:RaceRecord) : void
      {
         var item:BackpackItem = null;
         if(this.charType != charType)
         {
            if(Boolean(this.hat))
            {
               this.hat = ItemsStore.getHatById(this.hat.getId(),charType);
            }
            for each(item in this.wardrobe.getDeposit())
            {
               this.wardrobe.replaceItem(item.getId(),ItemsStore.getHatById(item.getRecord().getId(),charType),true);
            }
         }
         this.charType = charType;
         this.bonusStats.update(this);
      }
      
      public function setCharName(value:String) : void
      {
         this.charName = value;
      }
      
      public function getGraveType(race:RaceRecord) : String
      {
         return race.graveType;
      }
      
      public function getSoulType(race:RaceRecord) : String
      {
         return race.soulType;
      }
      
      public function getWeaponBackpack() : WeaponBackpack
      {
         return this.weaponBackpack;
      }
      
      public function setWeaponBackpack(backpack:WeaponBackpack) : void
      {
         this.weaponBackpack = backpack;
      }
      
      public function isInTeam(friend:UserProfile) : Boolean
      {
         var wormProfile:UserProfile = null;
         for each(wormProfile in this.team)
         {
            if(wormProfile.getId() == friend.getId())
            {
               return true;
            }
         }
         return false;
      }
      
      public function canAddInTeam() : Boolean
      {
         return this.team.length < LevelsGrid.getMaxTeamSize(this.getLevel());
      }
      
      public function minus(price:Price, moneyType:MoneyType) : void
      {
         switch(moneyType)
         {
            case MoneyType.REAL:
               this.addRealMoney(-price.realPrice);
               break;
            case MoneyType.INGAME:
               this.addIngameMoney(-price.ingamePrice);
         }
      }
      
      public function plusMoney(value:int, moneyType:MoneyType) : void
      {
         switch(moneyType)
         {
            case MoneyType.REAL:
               this.addRealMoney(value);
               break;
            case MoneyType.INGAME:
               this.addIngameMoney(value);
         }
      }
      
      public function plus(price:Price, moneyType:MoneyType) : void
      {
         switch(moneyType)
         {
            case MoneyType.REAL:
               this.addRealMoney(price.realPrice);
               break;
            case MoneyType.INGAME:
               this.addIngameMoney(price.ingamePrice);
         }
      }
      
      public function getTeamHp() : uint
      {
         var profile:UserProfile = null;
         var teamHp:uint = 0;
         for each(profile in this.team)
         {
            teamHp += profile.getHp();
            teamHp += profile.getBonusStats().hp;
         }
         return teamHp;
      }
      
      public function equals(p:UserProfile) : Boolean
      {
         return p != null && this.id == p.id;
      }
      
      public function getBonusStats() : BonusStats
      {
         return this.bonusStats;
      }
      
      public function getTotalHp() : uint
      {
         return this.bonusStats.hp + this.getHp();
      }
      
      public function getWardrobe() : BaseBackpack
      {
         return this.wardrobe;
      }
      
      public function addPrize(prize:DailyBonusPrize) : void
      {
         if(prize.type.equals(DailyBonusType.FUZY))
         {
            this.ingameMoney.value += prize.count;
         }
         else if(prize.type.equals(DailyBonusType.RUBY))
         {
            this.realMoney.value += prize.count;
         }
         if(prize.type.equals(DailyBonusType.REACTION))
         {
            this.reactionRate.value += prize.count;
         }
      }
      
      public function removePrize(prize:DailyBonusPrize) : void
      {
         if(prize.type.equals(DailyBonusType.FUZY))
         {
            this.ingameMoney.value -= prize.count;
         }
         else if(prize.type.equals(DailyBonusType.RUBY))
         {
            this.realMoney.value -= prize.count;
         }
         if(prize.type.equals(DailyBonusType.REACTION))
         {
            this.reactionRate.value -= prize.count;
         }
      }
      
      public function getReagentsBackpack() : BaseBackpack
      {
         return this.reagentBackpack;
      }
      
      public function getRecipeBackpack() : BaseBackpack
      {
         return this.recipeBackpack;
      }
   }
}

