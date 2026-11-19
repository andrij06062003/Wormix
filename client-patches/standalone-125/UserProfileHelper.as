package ru.pragmatix.wormix.model.user
{
   import flash.external.ExternalInterface;
   import §1#"§.§<"3§;
   import §1#"§.§@o§;
   import §3R§.§&"D§;
   import §3R§.§0"1§;
   import §4#p§.§3o§;
   import §7!o§.Race;
   import §8#e§.§3$G§;
   import §8#e§.§]$E§;
   import §>#L§.§1"O§;
   import §^E§.§^]§;
   import flash.net.URLRequest;
   import flash.net.navigateToURL;
   import ru.pragmatix.flox.gui.controls.§?"W§;
   import ru.pragmatix.flox.social.controller.§=1§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.messages.structures.RatingProfileStructure;
   import ru.pragmatix.wormix.messages.structures.SimpleRatingProfileStructure;
   import ru.pragmatix.wormix.messages.structures.TeamMemberStructure;
   import ru.pragmatix.wormix.messages.structures.UnitStructure;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.messages.structures.WeaponStructure;
   import ru.pragmatix.wormix.social.§#!K§;
   
   public class §7X§
   {
      
      private static const §+!Y§:int = 20;
      
      public static const §&#E§:String = "N/A";
      
      public function §7X§()
      {
         super();
      }
      
      public static function §4$C§(param1:UserProfile, param2:int = 20) : String
      {
         if(param1.getSocialData())
         {
            return §=#^§(§#!K§.§8"d§.§@#G§(param1),param2);
         }
         if(param1.getCharName())
         {
            return param1.getCharName();
         }
         return §&#E§;
      }
      
      public static function §=#^§(param1:String, param2:uint = 20) : String
      {
         if(!param1)
         {
            return §&#E§;
         }
         if(param2 != -1 && param1.length > param2)
         {
            return param1.substring(0,param2) + "...";
         }
         return param1;
      }
      
      public static function §<!@§(param1:UserProfile, param2:uint = 20) : String
      {
         return §=#^§(param1.getSocialData() ? param1.getSocialData().§<!6§() : "",param2);
      }
      
      public static function §7o§(param1:UserProfile, param2:uint = 20) : String
      {
         return §=#^§(param1.getSocialData() ? param1.getSocialData().§7o§() : "",param2);
      }
      
      public static function §;?§(param1:UserProfile) : String
      {
         return §4$C§(param1,-1);
      }
      
      public static function §,$P§(param1:UserProfile) : String
      {
         var _loc4_:UserProfile = null;
         var _loc2_:String = "";
         var _loc3_:int = 0;
         while(_loc3_ < param1.§?"^§().length)
         {
            _loc4_ = param1.§?"^§()[_loc3_];
            if(_loc3_ > 0)
            {
               _loc2_ += ", ";
            }
            _loc2_ += §;?§(_loc4_);
            _loc2_ += "(" + _loc4_.getLevel() + " " + §^!R§.NOTE_LEVEL + ", " + _loc4_.getRace().getName() + ")";
            _loc3_++;
         }
         return _loc2_;
      }
      
      public static function §+$R§(param1:§1"O§) : Boolean
      {
         return §,$'§(Application.userProfile,param1.moneyType,param1.price);
      }
      
      public static function §,$'§(param1:UserProfile, param2:§0"1§, param3:§&"D§) : Boolean
      {
         var _loc4_:Boolean = true;
         switch(param2)
         {
            case §0"1§.§>!M§:
               if(param1.§8"3§() < param3.realPrice)
               {
                  _loc4_ = false;
               }
               break;
            case §0"1§.§%!e§:
               if(param1.§]#K§() < param3.ingamePrice)
               {
                  _loc4_ = false;
               }
         }
         if(!_loc4_)
         {
            new §^]§().show();
         }
         return _loc4_;
      }
      
      // Patch (standalone-125): "Страница пользователя" opened the player's VK page. It now opens the player's
      // profile on the game's page (wwwroot/play.html, window.wormixProfile).
      public static function §#!#§(param1:UserProfile) : void
      {
         try
         {
            if(Boolean(param1) && ExternalInterface.available)
            {
               ExternalInterface.call("wormixProfile",String(param1.getId()));
               return;
            }
         }
         catch(e:Error)
         {
         }
         if(Boolean(param1) && Boolean(param1.getSocialData()))
         {
            if(param1.getSocialData().§8$H§())
            {
               navigateToURL(new URLRequest(param1.getSocialData().§8$H§()),"_blank");
            }
            else
            {
               §?"W§.§4#v§("",§^!R§.FORBIDDEN_TRANSITION_PAGE,§?"W§.§3#r§);
            }
         }
      }
      
      public static function §4!H§(param1:UserProfile) : int
      {
         return param1.getLevel() * 2 - param1.stats.§%#^§.value - param1.stats.§^"#§.value;
      }
      
      public static function §7"h§(param1:Array, param2:Boolean = false) : Array
      {
         var _loc4_:Object = null;
         var _loc5_:UserProfile = null;
         var _loc3_:Array = [];
         for each(_loc4_ in param1)
         {
            _loc5_ = new UserProfile();
            if(_loc4_ is UserProfileStructure)
            {
               _loc5_.§+8§.§3#n§(_loc4_ as UserProfileStructure);
            }
            else if(_loc4_ is RatingProfileStructure)
            {
               _loc5_.§+8§.§,#Q§(_loc4_ as RatingProfileStructure);
            }
            else if(_loc4_ is UnitStructure)
            {
               _loc5_.§+8§.§@!3§(_loc4_ as UnitStructure,true);
            }
            else if(_loc4_ is SimpleRatingProfileStructure)
            {
               _loc5_.§+8§.§;!j§(_loc4_ as SimpleRatingProfileStructure);
            }
            if(param2)
            {
               if(_loc5_.getLevel() != 0)
               {
                  _loc3_.push(_loc5_);
               }
            }
            else
            {
               _loc3_.push(_loc5_);
            }
         }
         return _loc3_;
      }
      
      public static function §2#%§(param1:UserProfile, param2:uint) : Boolean
      {
         var _loc4_:int = 0;
         var _loc5_:§3$G§ = null;
         var _loc6_:UserProfile = null;
         param1.§;c§(param1.§-#s§() + param2);
         var _loc3_:uint = §<"3§.§]#t§(param1.getLevel());
         if(param1.§-#s§() >= _loc3_)
         {
            if(§<"3§.§""l§(param1.getLevel() + 1))
            {
               _loc4_ = int(§<"3§.§4!Y§(param1.getLevel() + 1));
               param1.stats.§8"n§.value = _loc4_;
               param1.§##D§(param1.getLevel() + 1);
               param1.§;c§(param1.§-#s§() - _loc3_);
               param1.addIngameMoney(§@o§.§1#K§);
               §-"d§.§##2§.§=v§();
               for each(_loc5_ in §]$E§.§@!k§())
               {
                  if(!param1.§-#N§().§@$E§(_loc5_.getId()) && _loc5_.§4$M§() <= param1.getLevel())
                  {
                     param1.§-#N§().putItem(_loc5_);
                  }
               }
               for each(_loc6_ in param1.§3!q§())
               {
                  if(_loc6_.§^$<§() >= param1.getLevel())
                  {
                     _loc6_.§+8§.§^!l§(param1.getLevel());
                  }
               }
               return true;
            }
            param1.§;c§(_loc3_);
         }
         return false;
      }
      
      public static function §<$A§(param1:int, param2:int) : String
      {
         return param1.toString() + (param2 ? " (" + (param2 > 0 ? "+" : "") + param2 + ")" : "");
      }
      
      public static function §7Y§(param1:String) : UserProfile
      {
         var _loc2_:UserProfileStructure = new UserProfileStructure();
         var _loc3_:TeamMemberStructure = new TeamMemberStructure();
         _loc2_.socialId = param1;
         _loc2_.id = parseInt(param1);
         _loc2_.money = 100;
         _loc2_.realMoney = 3;
         _loc2_.units = [_loc3_];
         _loc3_.level = 1;
         _loc3_.armor = 1;
         _loc3_.attack = 1;
         _loc3_.ownerId = _loc2_.id;
         _loc3_.socialOwnerId = _loc2_.socialId;
         _loc3_.experience = 0;
         _loc3_.level = 1;
         _loc3_.hatId = Race.§"D§.getId();
         _loc3_.teamMemberType = §3o§.§-$8§;
         _loc2_.weaponRecordList = [new WeaponStructure(1,-1),new WeaponStructure(2,-1),new WeaponStructure(3,-1),new WeaponStructure(20,5),new WeaponStructure(21,8)];
         var _loc4_:UserProfile = new UserProfile();
         _loc4_.§+8§.§3#n§(_loc2_);
         return _loc4_;
      }
      
      public static function §6#'§(param1:UserProfile) : Boolean
      {
         return param1 == Application.userProfile;
      }
      
      public static function §+#,§(param1:Vector.<UserProfile>) : UserProfile
      {
         var _loc2_:UserProfile = null;
         for each(_loc2_ in param1)
         {
            if(!_loc2_.equals(Application.userProfile))
            {
               return _loc2_;
            }
         }
         return null;
      }
      
      public static function §,!k§(param1:UserProfile) : String
      {
         return §-"d§.§=w§.§,#<§() == §=1§.VK ? "@id" + param1.§9$3§() : §4$C§(param1);
      }
   }
}

