package ru.pragmatix.wormix.social.common.post
{
   import §[$@§.§]#7§;
   import § !O§.§1#;§;
   import § !O§.§2J§;
   import § !O§.§;#]§;
   import § !O§.§^U§;
   import § !O§.§`p§;
   import § #R§.*;
   import §!!=§.§7"Q§;
   import §!!=§.§?0§;
   import §!!=§.§`§;
   import §!!G§.§'"i§;
   import §"!+§.§>#x§;
   import §"#?§.§1!J§;
   import §"#U§.§2"0§;
   import §"#U§.§2$5§;
   import §#!s§.§@"7§;
   import §#C§.§"§;
   import §#C§.§<L§;
   import §%"D§.§0o§;
   import §%#l§.*;
   import §%^§.§9§;
   import §%c§.*;
   import §&!Y§.§8"9§;
   import §&#q§.§#=§;
   import §'""§.§>$Q§;
   import §'"4§.§9!o§;
   import §'"H§.§2"C§;
   import §+!U§.§7"s§;
   import §+$I§.§<?§;
   import §+0§.§"$L§;
   import §,!8§.§!"z§;
   import §,!8§.§""A§;
   import §,!8§.§-!H§;
   import §,!8§.§1I§;
   import §,!8§.§9k§;
   import §-v§.§ "]§;
   import §-v§.§ §;
   import §-v§.§<""§;
   import §-v§.§?"Q§;
   import §0!8§.§"+§;
   import §0!P§.§1!i§;
   import §0#1§.§7"D§;
   import §1#"§.§"Q§;
   import §1#"§.§%!]§;
   import §1#"§.§-"`§;
   import §1#"§.§8" §;
   import §1#"§.§9#w§;
   import §1#"§.§=!j§;
   import §1$#§.§7x§;
   import §1Q§.§#§;
   import §1Q§.§##5§;
   import §3R§.§0"1§;
   import §3R§.§8"$§;
   import §3R§.§<$H§;
   import §4"+§.§@J§;
   import §4"D§.§ $4§;
   import §4"D§.§];§;
   import §4#p§.§3o§;
   import §4#p§.Team;
   import §4F§.*;
   import §5!`§.§ ##§;
   import §5!`§.CheatDetectedEvent;
   import §5"U§.§-!Z§;
   import §5"U§.§;"M§;
   import §5#V§.*;
   import §6!O§.§2!-§;
   import §6!O§.§3"e§;
   import §6k§.*;
   import §7#F§.§=U§;
   import §7I§.§+#?§;
   import §8#e§.§]$E§;
   import §9!@§.§4#G§;
   import §9!@§.§9"V§;
   import §9!@§.§@$J§;
   import §9!@§.§^9§;
   import §9"`§.§8%§;
   import §9"j§.§9"X§;
   import §9$"§.§6!%§;
   import §9$"§.§;!^§;
   import §;"S§.DamageEvent;
   import §;'§.§8#_§;
   import §;'§.§8]§;
   import §<"f§.§["Q§;
   import §="z§.§,#f§;
   import §>"'§.§]!y§;
   import §>$1§.§ #O§;
   import §?"Z§.*;
   import §?=§.§3$%§;
   import §?=§.§>$T§;
   import §@!"§.§;"K§;
   import §@!9§.§^"E§;
   import §@#M§.*;
   import §@#n§.§[q§;
   import §@$O§.§["M§;
   import §[#M§.§!3§;
   import §[#M§.§'4§;
   import §[#b§.§+#c§;
   import §[$@§.§]§;
   import §[z§.§ t§;
   import §[z§.§5!X§;
   import §[z§.§6#,§;
   import §]"x§.*;
   import §]`§.§ "§;
   import §]s§.§#§;
   import §]s§.§]#-§;
   import §^"$§.§,!7§;
   import §^#c§.§'§;
   import §^1§.§+!`§;
   import §`!g§.§ $#§;
   import §`!g§.§'!8§;
   import §`!g§.§,k§;
   import §`!g§.§="w§;
   import §`!g§.§^$6§;
   import com.greensock.TweenLite;
   import com.greensock.easing.Expo;
   import flash.display.DisplayObject;
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.display.Sprite;
   import flash.events.Event;
   import flash.events.IEventDispatcher;
   import flash.geom.ColorTransform;
   import flash.text.TextField;
   import flash.ui.Keyboard;
   import flash.utils.ByteArray;
   import flash.utils.Dictionary;
   import flash.utils.IDataOutput;
   import flash.utils.getQualifiedClassName;
   import flash.utils.getTimer;
   import ru.pragmatix.flox.gui.controls.§0b§;
   import ru.pragmatix.flox.gui.controls.§?"W§;
   import ru.pragmatix.flox.logger.§1#O§;
   import ru.pragmatix.flox.logger.§[$Q§;
   import ru.pragmatix.flox.serialization.§;"k§;
   import ru.pragmatix.flox.social.controller.§5y§;
   import ru.pragmatix.wormix.controller.§-"d§;
   import ru.pragmatix.wormix.gui.§^!R§;
   import ru.pragmatix.wormix.gui.window.§0#q§;
   import ru.pragmatix.wormix.gui.window.§<!X§;
   import ru.pragmatix.wormix.gui.window.BossBattleOfferDialog;
   import ru.pragmatix.wormix.gui.window.§^3§;
   import ru.pragmatix.wormix.messages.clans.structures.ClanMemberTO;
   import ru.pragmatix.wormix.messages.server.ProfilesResult;
   import ru.pragmatix.wormix.messages.structures.UserProfileStructure;
   import ru.pragmatix.wormix.model.rewards.§-#I§;
   import ru.pragmatix.wormix.model.user.§7X§;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.objects.*;
   import ru.pragmatix.wormix.physics.§6o§;
   import ru.pragmatix.wormix.pvp.messages.ex.*;
   import ru.pragmatix.wormix.serialization.server.*;
   import ru.pragmatix.wormix.social.§#!K§;
   import ru.pragmatix.wormix.utils.§[#;§;
   import ru.pragmatix.wormix.weapons.*;
   import ru.pragmatix.wormix.weapons.interfaces.§ #d§;
   import ru.pragmatix.wormix.weapons.interfaces.§'"Y§;
   import ru.pragmatix.wormix.weapons.interfaces.§5#j§;
   import ru.pragmatix.wormix.weapons.interfaces.§[!=§;
   
   public class §'i§ extends §##5§ implements §"+§
   {
      
      protected static const log:§[$Q§ = new §[$Q§(§'i§);
      
      protected var §"q§:§9"V§;
      
      protected var §`!"§:§@$J§;
      
      protected var § "C§:§^9§;
      
      protected var §-",§:§4#G§;
      
      protected var iconsScheme:§2"C§;
      
      protected var isCanceledRequestPumpReaction:Boolean;
      
      protected var isCanceledNewExpensiveWeapon:Boolean;
      
      protected var isCanceledNewHat:Boolean;
      
      protected var isCanceledNewArtifact:Boolean;
      
      protected var isCanceledNewUpgradedWeapon:Boolean;
      
      public function §'i§(param1:§1#;§, param2:§5y§, param3:§2"C§, param4:String = "", param5:§@$J§ = null, param6:§4#G§ = null, param7:§^9§ = null)
      {
         super(param1,param2,param4);
         this.iconsScheme = param3;
         this.§`!"§ = param5 || new §@$J§();
         this.§ "C§ = param7 || new §^9§();
         this.§-",§ = param6 || new §4#G§();
         this.§"q§ = new §9"V§();
         this.§"q§.§6"+§();
         if(§9$2§)
         {
            this.§`!"§.§6"+§();
         }
         if(§@!y§)
         {
            this.§ "C§.§6"+§();
         }
         if(§!#%§)
         {
            this.§-",§.§6"+§();
         }
      }
      
      public function §9"I§(param1:§]#7§) : void
      {
         if(param1.configName != "worm")
         {
            this.§@"%§(this.§ "C§.§@$7§,this.§ "C§.§="+§,this.§ "C§.§9"I§,{"race":param1.getName()},this.iconsScheme.§=I§(param1));
         }
      }
      
      public function rubyFoundInHouse(param1:UserProfile) : void
      {
         var _loc2_:Object = this.§&_§(param1);
         this.§@"%§(this.§ "C§.§!# §,this.§ "C§.§="+§,this.§ "C§.rubyFoundInHouse,_loc2_,this.iconsScheme.rubyFoundInHouse);
      }
      
      public function activityPumpFriendsReaction(param1:UserProfile, param2:uint) : void
      {
      }
      
      public function requestPumpReaction(param1:UserProfile, param2:Boolean) : void
      {
         var params:Object = null;
         var profile:UserProfile = param1;
         var needToAsk:Boolean = param2;
         if(profile)
         {
            params = this.§&_§(profile);
            if(needToAsk)
            {
               if(!this.isCanceledRequestPumpReaction)
               {
                  this.§#!+§(profile,"",this.§-",§.§3t§,this.§-",§.requestPumpReaction,params,this.iconsScheme.requestPumpReaction,null,function():void
                  {
                     isCanceledRequestPumpReaction = true;
                  });
               }
            }
            else
            {
               §-"d§.§%"r§.§^H§();
               notify(profile,this.§-",§.requestPumpReaction,params,this.iconsScheme.requestPumpReaction);
            }
         }
         else
         {
            §?"W§.§4#v§(§^!R§.TITLE_ATTENTION,this.§-",§.§-#$§);
         }
      }
      
      public function § !N§(param1:uint) : void
      {
         §-"d§.§%"r§.§^H§();
         §8!O§(this.§ "C§.§ !N§,{"level":param1},this.iconsScheme.§%"6§(Application.userProfile.getRace()));
      }
      
      public function §["6§(param1:§9k§) : void
      {
         var _loc3_:DisplayObject = null;
         var _loc2_:String = this.§ "C§.§!!j§[int(Math.random() * this.§ "C§.§!!j§.length)];
         if(param1 is §1I§)
         {
            _loc3_ = this.§;$+§(param1,§ $#§.§0!I§,§ $#§.§4!3§);
         }
         else if(param1 is §!"z§)
         {
            _loc3_ = this.§;$+§(param1,§ $#§.§&$9§,§ $#§.§'"g§);
         }
         else if(param1 is §""A§)
         {
            _loc3_ = this.§;$+§(param1,§ $#§.§<$3§,§ $#§.§4!3§);
         }
         if(_loc3_)
         {
            this.§>"u§(this.§ "C§.§-I§,this.§ "C§.§="+§,_loc2_,_loc3_,{});
         }
         else
         {
            this.§@"%§(this.§ "C§.§-I§,this.§ "C§.§="+§,param1.getName() + "! " + _loc2_,{},this.iconsScheme.bonusIcon);
         }
      }
      
      public function §3$7§(param1:§ $4§, param2:int, param3:MovieClip) : void
      {
         var _loc4_:uint = 0;
         var _loc5_:Object = null;
         _loc4_ = this.§ "C§.§6#3§.length;
         _loc5_ = {
            "achievement_name":param1.levels[param2].name,
            "user_name":§7X§.§4$C§(Application.userProfile)
         };
         var _loc6_:String = §;#]§.format(this.§ "C§.§6#3§[int(Math.random() * _loc4_)],_loc5_);
         §-"d§.§%"r§.§^H§();
         §@!y§.§+#-§(_loc6_,§^!R§.GAME_NAME,param3);
      }
      
      protected function §;$+§(param1:§9k§, param2:DisplayObject = null, param3:DisplayObject = null) : DisplayObject
      {
         return § $#§.§]"R§(§^$6§.getMenuIcon(param1),§ $#§.§6!Y§,param2,param3);
      }
      
      public function §9$ §(param1:§9k§) : void
      {
         var _loc2_:String = null;
         if(param1)
         {
            _loc2_ = §+#?§.§,!M§(param1);
            if(param1 is §!"z§)
            {
               this.§1"E§(_loc2_,param1 as §!"z§);
            }
            else if(param1 is §1I§)
            {
               if(param1.§"N§() >= §9#w§.§;!"§ || param1.getRealPrice() >= §9#w§.§]![§)
               {
                  this.§@a§(_loc2_,param1 as §1I§);
               }
               else
               {
                  this.§@#6§(_loc2_,param1 as §1I§);
               }
            }
            else if(param1 is §""A§)
            {
               this.§1#N§(_loc2_,param1 as §""A§);
            }
         }
      }
      
      protected function §@#6§(param1:String, param2:§1I§) : void
      {
      }
      
      protected function §@a§(param1:String, param2:§1I§) : void
      {
         var weaponName:String = param1;
         var item:§1I§ = param2;
         if(!this.isCanceledNewExpensiveWeapon)
         {
            this.§>"u§(this.§ "C§.§^!7§,this.§ "C§.§="+§,this.§ "C§.§@#6§,this.§;$+§(item,§ $#§.§0!I§,§ $#§.§4!3§),{"weapon":weaponName},null,function():void
            {
               isCanceledNewExpensiveWeapon = true;
            });
         }
      }
      
      protected function §1"E§(param1:String, param2:§!"z§) : void
      {
         var hatName:String = param1;
         var item:§!"z§ = param2;
         if(!this.isCanceledNewHat)
         {
            this.§>"u§(this.§ "C§.§`!c§,this.§ "C§.§="+§,this.§ "C§.§1"E§,this.§;$+§(item,§ $#§.§&$9§,§ $#§.§'"g§),{"hat":hatName},null,function():void
            {
               isCanceledNewHat = true;
            });
         }
      }
      
      protected function §1#N§(param1:String, param2:§""A§) : void
      {
         var artifactName:String = param1;
         var item:§""A§ = param2;
         if(!this.isCanceledNewArtifact)
         {
            this.§>"u§(this.§ "C§.§2$C§,this.§ "C§.§="+§,this.§ "C§.§1#N§,this.§;$+§(item,§ $#§.§<$3§,§ $#§.§4!3§),{"artifact":artifactName},null,function():void
            {
               isCanceledNewArtifact = true;
            });
         }
      }
      
      public function §;"V§(param1:§9k§) : void
      {
         var _loc2_:String = null;
         if(param1)
         {
            _loc2_ = §+#?§.§,!M§(param1);
            if(param1 is §!"z§)
            {
               this.§4'§(_loc2_,param1 as §!"z§);
            }
            else if(param1 is §""A§)
            {
               this.§[!Q§(_loc2_,param1 as §""A§);
            }
            else if(param1 is §1I§ && param1.getId() > §>$Q§.§1J§)
            {
               this.§##§(_loc2_,param1 as §1I§);
            }
         }
      }
      
      protected function §4'§(param1:String, param2:§!"z§) : void
      {
         §-"d§.§%"r§.§^H§();
         §@!y§.§+#-§(§;#]§.format(this.§ "C§.§'"-§,{"item":param1}),§^!R§.GAME_NAME,this.§;$+§(param2,§ $#§.§&$9§,§ $#§.§'"g§));
      }
      
      protected function §[!Q§(param1:String, param2:§""A§) : void
      {
         §-"d§.§%"r§.§^H§();
         §@!y§.§+#-§(§;#]§.format(this.§ "C§.§'"-§,{"item":param1}),§^!R§.GAME_NAME,this.§;$+§(param2,§ $#§.§<$3§,§ $#§.§4!3§));
      }
      
      protected function §##§(param1:String, param2:§1I§) : void
      {
         var weaponName:String = param1;
         var item:§1I§ = param2;
         if(!this.isCanceledNewUpgradedWeapon)
         {
            this.§>"u§(this.§ "C§.§^!7§,this.§ "C§.§="+§,this.§ "C§.§'"-§,this.§;$+§(item,§ $#§.§0!I§,§ $#§.§4!3§),{"item":weaponName},null,function():void
            {
               isCanceledNewUpgradedWeapon = true;
            });
         }
      }
      
      protected function newItems(param1:Array, param2:Array) : void
      {
      }
      
      public function §'$<§(param1:Array) : void
      {
         var _loc3_:§9k§ = null;
         var _loc2_:Array = [];
         for each(_loc3_ in param1)
         {
            _loc2_.push(§+#?§.§,!M§(_loc3_));
         }
         this.newItems(_loc2_,param1);
      }
      
      public function newTeamMate(param1:UserProfile, param2:Boolean = true) : void
      {
         var _loc3_:Object = this.§=!9§(param1);
         if(param2)
         {
            this.§#!+§(param1,this.§-",§.§""n§,this.§-",§.§%$@§,this.§-",§.newTeamMate,_loc3_,this.iconsScheme.newTeamMateYou);
         }
         else
         {
            this.§@"%§(§;#]§.format(this.§`!"§.newTeamMate,_loc3_),this.§ "C§.§="+§,this.§`!"§.newTeamMate,_loc3_,this.iconsScheme.newTeamMateYou);
         }
      }
      
      public function kickTeamMate(param1:UserProfile, param2:Boolean = true) : void
      {
         var _loc3_:Object = this.§=!9§(param1);
         if(param2)
         {
            this.§#!+§(param1,this.§-",§.§ #j§,this.§-",§.§0"d§,this.§-",§.kickTeamMate,_loc3_,this.iconsScheme.kickTeamMateYou);
         }
         else
         {
            this.§@"%§(§;#]§.format(this.§`!"§.kickTeamMate,_loc3_),this.§ "C§.§="+§,this.§`!"§.kickTeamMate,_loc3_,this.iconsScheme.kickTeamMateYou);
         }
      }
      
      public function §]-§(param1:§ "]§, param2:§<""§, param3:Function = null) : void
      {
         var _loc5_:§-"`§ = null;
         var _loc6_:§-"`§ = null;
         var _loc4_:UserProfile = §7X§.§+#,§(param1.§3!<§());
         if(param1.battleType == § #5§.§!#§)
         {
            this.§&]§(param1.§>R§()[0],param2.result);
         }
         else if(param1.battleType == § #5§.§0"@§)
         {
            this.§-#q§(_loc4_,param1.§>R§(),param2.result,param2.rating);
         }
         else if(param1.battleType == § #5§.§9""§)
         {
            this.§0#i§(param2.result,param2.§9#G§.name);
         }
         else if(param1.battleType == § #5§.§%#!§)
         {
            _loc5_ = §<?§.§9##§(param1.missionIds[0]);
            _loc6_ = param1.missionIds.length > 1 ? §<?§.§9##§(param1.missionIds[1]) : null;
            if(!param1.§2"k§())
            {
               if(param1.§!#p§())
               {
                  this.§@7§(_loc5_,_loc6_,param2.result,_loc4_);
               }
               else if(§<?§.§0z§(param1.missionIds[0]))
               {
                  this.§@#Z§(_loc5_,param2.result,_loc4_);
               }
               else
               {
                  this.§5![§(_loc5_.getName());
               }
            }
         }
         if(param3 != null)
         {
            param3();
         }
      }
      
      public function §0#i§(param1:§?"Q§, param2:String) : void
      {
         var _loc3_:String = param1 == §?"Q§.§["9§ ? this.§`!"§.missionComplete : this.§`!"§.§#!0§;
         var _loc4_:Object = {"name_mission":param2};
         §-"d§.§%"r§.§^H§();
         if(§9$2§)
         {
            log.info(this + ".postBattleMission: activityPoster","VK");
            postActivity(_loc3_,_loc4_,this.iconsScheme.missionComplete);
         }
         else
         {
            log.info(this + ".postBattleMission: postOnWall","VK");
            §8!O§(_loc3_,_loc4_,this.iconsScheme.missionComplete);
         }
      }
      
      protected function §"!N§() : void
      {
         this.§@"%§(this.§`!"§.missionComplete,this.§ "C§.§="+§,this.§`!"§.missionComplete,null,this.iconsScheme.missionComplete);
      }
      
      public function §5![§(param1:String) : void
      {
         var _loc2_:Object = {"name_mission":param1};
         §-"d§.§%"r§.§^H§();
         if(§9$2§)
         {
            postActivity(this.§ "C§.§<"C§,_loc2_,this.iconsScheme.tutorialIcon);
         }
         else
         {
            §8!O§(this.§ "C§.§<"C§,_loc2_,this.iconsScheme.tutorialIcon);
         }
      }
      
      protected function §5$@§(param1:Vector.<uint>) : DisplayObject
      {
         var _loc2_:BossBattleOfferDialog = §^3§.create(§ #5§.§%#!§) as BossBattleOfferDialog;
         _loc2_.§=#3§(param1);
         var _loc3_:MovieClip = new MovieClip();
         _loc3_.addChild(_loc2_.§6#!§);
         _loc2_.§6#!§.x = 0;
         _loc2_.§6#!§.y = 20;
         return § $#§.§;$2§(_loc3_);
      }
      
      public function §@7§(param1:§-"`§, param2:§-"`§, param3:§?"Q§, param4:UserProfile = null) : void
      {
         var _loc5_:String = null;
         var _loc6_:String = null;
         if(param3.equals(§?"Q§.§["9§))
         {
            _loc6_ = param4 ? this.§`!"§.§"!#§ : this.§`!"§.§@!<§;
            _loc5_ = param4 ? this.iconsScheme.winSuperbossWithFriend : this.iconsScheme.winSuperboss;
         }
         else if(param3.equals(§?"Q§.§-"P§))
         {
            _loc6_ = this.§ "C§.§;#^§;
            _loc5_ = param4 ? this.iconsScheme.looseSuperbossWithFriend : this.iconsScheme.looseSuperboss;
         }
         else
         {
            _loc6_ = param4 ? this.§ "C§.§"h§ : this.§ "C§.§ "m§;
            _loc5_ = param4 ? this.iconsScheme.looseSuperbossWithFriend : this.iconsScheme.looseSuperboss;
         }
         var _loc7_:String = §7X§.§,!k§(Application.userProfile);
         var _loc8_:String = param4 ? §7X§.§,!k§(param4) : "";
         var _loc9_:Object = {
            "userName":_loc7_,
            "friendName":_loc8_,
            "bossName":param2.getName(),
            "bossName2":param1.getName()
         };
         §-"d§.§%"r§.§^H§();
         §8!O§(_loc6_,_loc9_,_loc5_);
      }
      
      public function §@#Z§(param1:§-"`§, param2:§?"Q§, param3:UserProfile = null) : void
      {
         var _loc4_:String = null;
         var _loc5_:String = null;
         if(param2.equals(§?"Q§.§["9§))
         {
            _loc4_ = param3 ? this.§`!"§.§0!V§ : this.§ "C§.§!$N§(param1.getId());
            _loc5_ = this.iconsScheme.§ <§(param1.getId());
         }
         else if(param2.equals(§?"Q§.§-"P§))
         {
            _loc4_ = this.§ "C§.§;#^§;
            _loc5_ = param3 ? this.iconsScheme.looseSuperbossWithFriend : this.iconsScheme.looseSuperboss;
         }
         else
         {
            _loc4_ = param3 ? this.§ "C§.§1#R§ : this.§ "C§.§[!k§;
            _loc5_ = param3 ? this.iconsScheme.looseSuperbossWithFriend : this.iconsScheme.looseSuperboss;
         }
         var _loc6_:String = §7X§.§,!k§(Application.userProfile);
         var _loc7_:String = param3 ? §7X§.§,!k§(param3) : "";
         §-"d§.§%"r§.§^H§();
         §8!O§(_loc4_,{
            "userName":_loc6_,
            "friendName":_loc7_,
            "bossName":param1.getName()
         },_loc5_);
      }
      
      public function §-#q§(param1:UserProfile, param2:Vector.<UserProfile>, param3:§?"Q§, param4:int) : void
      {
         var _loc11_:String = null;
         var _loc12_:String = null;
         var _loc13_:Object = null;
         var _loc5_:Boolean = param3 == §?"Q§.§["9§;
         var _loc6_:Boolean = param3 == §?"Q§.§-"P§;
         var _loc7_:int = int(Application.userProfile.getRating());
         var _loc8_:String = param1 ? §7X§.§,!k§(param1) : "";
         var _loc9_:String = param2 ? §7X§.§,!k§(param2[0]) : "";
         var _loc10_:String = Boolean(param2) && param2.length > 1 ? §7X§.§,!k§(param2[1]) : "";
         if(!param1)
         {
            if(param2.length > 1)
            {
               _loc12_ = _loc5_ ? this.§ "C§.§7#V§ : (_loc6_ ? this.§ "C§.§;#^§ : this.§ "C§.§<!3§);
               _loc11_ = _loc5_ ? this.iconsScheme.winTwoOpponents : this.iconsScheme.looseTwoOpponents;
               _loc13_ = {
                  "enemyName":_loc9_,
                  "enemyName2":_loc10_,
                  "rating":_loc7_
               };
            }
            else
            {
               _loc12_ = _loc5_ ? this.§ "C§.§7#V§ : (_loc6_ ? this.§ "C§.§;#^§ : this.§ "C§.§-!s§);
               _loc11_ = _loc5_ ? this.iconsScheme.winInPvp : this.iconsScheme.remindAboutLose;
               _loc13_ = {
                  "enemyName":_loc9_,
                  "rating":_loc7_
               };
            }
         }
         else
         {
            _loc12_ = _loc5_ ? this.§ "C§.§<$8§ : (_loc6_ ? this.§ "C§.§;#^§ : this.§ "C§.§<!3§);
            _loc11_ = _loc5_ ? this.iconsScheme.winWithFriend : this.iconsScheme.looseWithFriend;
            _loc13_ = {
               "allyName":_loc8_,
               "enemyName":_loc9_,
               "enemyName2":_loc10_
            };
         }
         _loc12_ = §9#0§.§7N§(_loc12_);
         §-"d§.§%"r§.§^H§();
         §8!O§(_loc12_,_loc13_,_loc11_);
      }
      
      public function §&]§(param1:UserProfile, param2:§?"Q§) : void
      {
         var _loc3_:Boolean = param2 == §?"Q§.§["9§;
         var _loc4_:Boolean = param2 == §?"Q§.§-"P§;
         var _loc5_:String = _loc3_ ? this.§ "C§.§`"`§ : (_loc4_ ? this.§ "C§.§;#^§ : this.§ "C§.§-!s§);
         _loc5_ = §9#0§.§7N§(_loc5_);
         var _loc6_:String = _loc3_ ? this.iconsScheme.winInPvp : this.iconsScheme.remindAboutLose;
         §-"d§.§%"r§.§^H§();
         §8!O§(_loc5_,{"enemyName":§7X§.§,!k§(param1)},_loc6_);
      }
      
      public function §4$S§(param1:§ "]§, param2:§<""§, param3:Function) : void
      {
      }
      
      public function callToPvp(param1:UserProfile, param2:UserProfile) : void
      {
         var _loc3_:String = this.iconsScheme.callToPvp[§-"d§.§1y§.§7$M§(0,this.iconsScheme.callToPvp.length - 1)];
         if(§!#%§)
         {
            §-"d§.§%"r§.§^H§();
            notify(param2,this.§2"3§(param1,param2),null,_loc3_);
         }
      }
      
      protected function §2"3§(param1:UserProfile, param2:UserProfile) : String
      {
         var _loc3_:Object = {
            "rating":param1.getRating(),
            "name":§7X§.§;?§(param2)
         };
         return §;#]§.format(param1.getRating() ? this.§"q§.callToPvp : this.§"q§.§%!z§,_loc3_);
      }
      
      public function share4DaySequence(param1:§"Q§) : void
      {
      }
      
      public function share5DaySequence(param1:§"Q§) : void
      {
      }
      
      public function enter4DaySequence(param1:§"Q§) : void
      {
         this.§@"%§(this.§ "C§.enter4DaySequenceTitle,this.§ "C§.§="+§,this.§ "C§.enter4DaySequence,{"reward":§;#]§.format(this.§>#t§(param1),{"count":param1.count})},this.iconsScheme.§8!G§(param1));
      }
      
      public function enter5DaySequence(param1:§"Q§) : void
      {
         this.§@"%§(this.§ "C§.enter5DaySequenceTitle,this.§ "C§.§="+§,this.§ "C§.enter5DaySequence,{"reward":§;#]§.format(this.§>#t§(param1),{"count":param1.count})},this.iconsScheme.§8!G§(param1));
      }
      
      protected function §>#t§(param1:§"Q§) : String
      {
         var _loc2_:String = null;
         if(param1.type == §8" §.§;#J§)
         {
            _loc2_ = this.§ "C§.enter5DaySequence_2mission;
         }
         else if(param1.type == §8" §.REACTION)
         {
            _loc2_ = this.§ "C§.enter5DaySequence_reaction;
         }
         else if(param1.type == §8" §.§2!?§)
         {
            _loc2_ = this.§ "C§.enter5DaySequence_fuzzes;
         }
         else if(param1.type == §8" §.RUBY)
         {
            _loc2_ = this.§ "C§.enter5DaySequence_rubies;
         }
         return _loc2_;
      }
      
      protected function §1§(param1:§"Q§) : String
      {
         return this.iconsScheme.§8!G§(param1);
      }
      
      public function §>",§(param1:String) : void
      {
         postActivity(this.§`!"§.§>",§,null,this.iconsScheme.postStatus,param1);
      }
      
      public function §%!3§(param1:String, param2:Boolean) : void
      {
         §-"d§.§%"r§.§^H§();
         §8!O§(param1,{},this.iconsScheme.§;§(param2));
      }
      
      public function §^!Q§(param1:UserProfile, param2:Function, param3:String = "") : void
      {
         var _loc4_:String = this.iconsScheme.missionComplete;
         if(this.§@#g§())
         {
            if(param2 == null)
            {
               this.§#!+§(param1,this.§-",§.§2d§,param3 + " " + this.§-",§.§@#o§,this.§-",§.returnFriend,null,_loc4_);
            }
            else
            {
               §-"d§.§%"r§.§^H§();
               §!#%§.§2##§(param1,this.§-",§.returnFriend,param2,_loc4_);
            }
         }
      }
      
      public function §2W§() : Boolean
      {
         return false;
      }
      
      public function §@#g§() : Boolean
      {
         return §!#%§ != null;
      }
      
      public function §]$J§() : Boolean
      {
         return §9$2§ != null;
      }
      
      public function §-$>§(param1:§ "]§, param2:§<""§, param3:Function) : void
      {
         var start:§ "]§ = param1;
         var result:§<""§ = param2;
         var callback:Function = param3;
         var text:Array = this.§2$ §(start,result);
         var postStatusDialog:§0#q§ = new §0#q§();
         postStatusDialog.setText(text,this.postStatus);
         postStatusDialog.addEventListener(§^"E§.WINDOW_CLOSE,function(param1:Event):void
         {
            callback();
         });
         postStatusDialog.show();
      }
      
      protected function postStatus(param1:String) : void
      {
         if(§@!y§)
         {
            §@!y§.§^#n§(param1,"Test Wormix",§""X§ + this.iconsScheme.postStatus);
         }
         else if(§9$2§)
         {
            §9$2§.§!!C§(param1,§""X§ + this.iconsScheme.postStatus,§^!R§.PLAY_CAPTION);
         }
      }
      
      private function §2$ §(param1:§ "]§, param2:§<""§) : Array
      {
         var _loc4_:UserProfile = null;
         var _loc9_:String = null;
         var _loc3_:Array = [];
         _loc4_ = Application.userProfile;
         var _loc5_:Object = {
            "level":_loc4_.getLevel(),
            "rating":_loc4_.getRating(),
            "opponent":§7X§.§;?§(param1.§>R§()[0])
         };
         if(param2.result.equals(§?"Q§.§["9§))
         {
            _loc3_.push(§;#]§.format(this.§"q§.§1O§,_loc5_));
         }
         if(param1.battleType.§&"0§())
         {
            _loc3_.push(this.§2"3§(_loc4_,param1.§>R§()[0]));
         }
         var _loc6_:Array = [this.§"q§.status0,this.§"q§.status1,this.§"q§.status2,this.§"q§.status3];
         var _loc7_:uint = 3 - _loc3_.length;
         var _loc8_:uint = 0;
         while(_loc8_ < _loc7_)
         {
            _loc9_ = _loc6_[§-"d§.§1y§.§7$M§(0,_loc6_.length - 1)];
            _loc3_.push(_loc9_);
            §`p§.remove(_loc6_,_loc9_);
            _loc8_++;
         }
         return _loc3_;
      }
      
      protected function §@"%§(param1:String, param2:String, param3:String, param4:Object = null, param5:String = null, param6:Function = null, param7:Function = null) : void
      {
         var msgBox:§?"W§ = null;
         var questionTitle:String = param1;
         var question:String = param2;
         var msg:String = param3;
         var params:Object = param4;
         var icon:String = param5;
         var callbackClose:Function = param6;
         var callbackCancel:Function = param7;
         if(§@!y§)
         {
            msgBox = §?"W§.§5#E§(questionTitle,question,§?"W§.§!$M§,function():void
            {
               §-"d§.§%"r§.§^H§();
               §8!O§(msg,params,icon);
               if(callbackClose != null)
               {
                  callbackClose();
               }
            });
            §,!7§.§<!s§(msgBox.getChildByName("msg"),msgBox.getChildByName("heightLine"));
            msgBox.addEventListener(§^"E§.WINDOW_CANCEL,function(param1:Event):void
            {
               if(callbackCancel != null)
               {
                  callbackCancel();
               }
            });
            msgBox.show();
         }
         else if(callbackClose != null)
         {
            // Nothing to ask: go on (the level-up dialog after a friend battle waits for this).
            callbackClose();
         }
      }
      
      protected function §>"u§(param1:String, param2:String, param3:String, param4:DisplayObject, param5:Object = null, param6:Function = null, param7:Function = null) : void
      {
         var msgBox:§?"W§ = null;
         var questionTitle:String = param1;
         var question:String = param2;
         var msg:String = param3;
         var icon:DisplayObject = param4;
         var params:Object = param5;
         var callbackClose:Function = param6;
         var callbackCancel:Function = param7;
         if(§@!y§)
         {
            msgBox = §?"W§.§5#E§(questionTitle,question,§?"W§.§!$M§,function():void
            {
               §-"d§.§%"r§.§^H§();
               §@!y§.§+#-§(§;#]§.format(msg,params),§^!R§.GAME_NAME,icon);
               if(callbackClose != null)
               {
                  callbackClose();
               }
            });
            §,!7§.§<!s§(msgBox.getChildByName("msg"),msgBox.getChildByName("heightLine"));
            msgBox.addEventListener(§^"E§.WINDOW_CANCEL,function(param1:Event):void
            {
               if(callbackCancel != null)
               {
                  callbackCancel();
               }
            });
            msgBox.show();
         }
      }
      
      protected function §#!+§(param1:UserProfile, param2:String, param3:String, param4:String, param5:Object = null, param6:String = null, param7:Function = null, param8:Function = null) : void
      {
         var msgBox:§?"W§ = null;
         var profile:UserProfile = param1;
         var questionTitle:String = param2;
         var question:String = param3;
         var msg:String = param4;
         var params:Object = param5;
         var icon:String = param6;
         var callbackClose:Function = param7;
         var callbackCancel:Function = param8;
         if(this.§@#g§())
         {
            msgBox = §?"W§.§5#E§(questionTitle,question,§?"W§.§!$M§,function():void
            {
               var _loc1_:* = undefined;
               §-"d§.§%"r§.§^H§();
               if(!profile.getSocialData())
               {
                  _loc1_ = new §2"0§();
                  _loc1_.setSocialId(profile.§9$3§());
                  profile.§9$?§(_loc1_);
               }
               notify(profile,msg,params,icon);
            });
            §,!7§.§<!s§(msgBox.getChildByName("msg"),msgBox.getChildByName("heightLine"));
            msgBox.addEventListener(§^"E§.WINDOW_CLOSE,function(param1:Event):void
            {
               if(callbackClose != null)
               {
                  callbackClose();
               }
            });
            msgBox.addEventListener(§^"E§.WINDOW_CANCEL,function(param1:Event):void
            {
               if(callbackCancel != null)
               {
                  callbackCancel();
               }
            });
            msgBox.show();
         }
      }
      
      public function §+$9§(param1:UserProfile, param2:String) : void
      {
         this.§#!+§(param1,§^!R§.§0!M§("FRIEND_INVITE_SENT"),this.§-",§.§0$,§,this.§-",§.§%#Z§,{
            "friendName":param1.getCharName(),
            "clanName":param2
         });
      }
      
      public function §-"'§() : void
      {
         this.§%"A§(§;#]§.format(this.§ "C§.§2'§,this.§"$2§()),this.§ "C§.§="+§,this.§ "C§.§9"T§,this.§"$2§(),this.iconsScheme.joinToClan);
      }
      
      public function §6$-§() : void
      {
         this.§%"A§(§;#]§.format(this.§ "C§.§5"%§,this.§"$2§()),this.§ "C§.§="+§,this.§ "C§.createClan,this.§"$2§(),this.iconsScheme.createClan);
      }
      
      public function §=#i§() : void
      {
         this.§%"A§(§;#]§.format(this.§ "C§.§%$H§,this.§"$2§()),this.§ "C§.§="+§,this.§ "C§.§-h§,this.§"$2§(),this.iconsScheme.extendClan);
      }
      
      public function §2#A§() : void
      {
         this.§%"A§(§;#]§.format(this.§ "C§.§+#4§,this.§"$2§()),this.§ "C§.§="+§,this.§ "C§.§5!I§,this.§"$2§(),this.iconsScheme.openClan);
      }
      
      public function §]#p§(param1:String, param2:String) : void
      {
         §8!O§(param1,this.§"$2§,param2);
      }
      
      public function §%"A§(param1:String, param2:String, param3:String, param4:Object, param5:*) : void
      {
         if(param5 is String)
         {
            this.§@"%§(param1,param2,param3,param4,param5);
         }
         else if(param5 is DisplayObject)
         {
            this.§>"u§(param1,param2,param3,param5,param4);
         }
      }
      
      protected function §,!"§(param1:UserProfile = null, param2:int = 0) : DisplayObject
      {
         var _loc3_:UserProfile = param1 == null ? Application.userProfile : param1;
         var _loc4_:Sprite = new Sprite();
         §[#;§.§=y§(_loc4_,§;"K§.§2F§(_loc3_.clanMember ? _loc3_.clanMember.clanId : param2).emblem);
         §[#;§.§?$!§(_loc4_);
         return § $#§.§;$2§(_loc4_);
      }
      
      protected function §@#2§(param1:UserProfile) : DisplayObject
      {
         return § $#§.§;$2§(§[#;§.§[$M§(param1.clanMember.rank).getContainer(),0,0,false,1122833);
      }
      
      protected function §"$2§(param1:UserProfile = null) : Object
      {
         var _loc2_:UserProfile = param1 == null ? Application.userProfile : param1;
         return {"clanName":_loc2_.clanMember.clanName};
      }
      
      protected function §=!9§(param1:UserProfile = null) : Object
      {
         var _loc2_:UserProfile = param1 == null ? Application.userProfile : param1;
         return {"name":_loc2_.getCharName()};
      }
      
      protected function §&_§(param1:UserProfile = null) : Object
      {
         var _loc2_:UserProfile = param1 == null ? Application.userProfile : param1;
         return {"name":§7X§.§;?§(_loc2_)};
      }
      
      protected function §%$,§(param1:§-#I§) : String
      {
         var _loc3_:Dictionary = null;
         var _loc6_:Dictionary = null;
         var _loc9_:String = null;
         var _loc10_:int = 0;
         var _loc11_:String = null;
         var _loc12_:int = 0;
         var _loc13_:Boolean = false;
         var _loc14_:§9k§ = null;
         var _loc15_:§9"X§ = null;
         var _loc2_:String = param1.§+#;§ + "\n";
         if(param1.money > 0)
         {
            _loc2_ = _loc2_ + §^!R§.COIN_QTIP.toLowerCase() + " (" + param1.money + "), ";
         }
         if(param1.realMoney > 0)
         {
            _loc2_ = _loc2_ + §^!R§.RUBY_QTIP.toLowerCase() + " (" + param1.realMoney + "), ";
         }
         if(param1.reaction > 0)
         {
            _loc2_ = _loc2_ + §^!R§.REACTION_QTIP.toLowerCase() + " (" + param1.reaction + "), ";
         }
         _loc3_ = param1.reagents;
         var _loc4_:int = int(§'!8§.§ !T§(_loc3_));
         var _loc5_:int = 0;
         if(Boolean(_loc3_) && _loc4_ > 0)
         {
            if(_loc4_ > 1)
            {
               _loc2_ = _loc2_ + §^!R§.REAGENTS_QTIP.toLowerCase() + " (";
            }
            for(_loc9_ in _loc3_)
            {
               _loc10_ = int(_loc9_);
               _loc2_ = _loc2_ + §]$E§.§#"z§(_loc10_).getName().toLowerCase() + " (" + _loc3_[_loc9_] + ")" + (_loc5_ < _loc4_ - 1 ? ", " : "");
               _loc5_++;
            }
            if(_loc4_ > 1)
            {
               _loc2_ += ")";
            }
         }
         _loc6_ = param1.items;
         var _loc7_:int = int(§'!8§.§ !T§(_loc6_));
         var _loc8_:int = 0;
         if(Boolean(_loc6_) && _loc7_ > 0)
         {
            if(_loc4_ > 0)
            {
               _loc2_ += ", ";
            }
            if(_loc7_ > 1)
            {
               _loc2_ = _loc2_ + §^!R§.ARMORY_QTIP.toLowerCase() + " (";
            }
            for(_loc11_ in _loc6_)
            {
               _loc12_ = int(_loc11_);
               _loc13_ = true;
               if(_loc12_ >= 2000)
               {
                  _loc14_ = §2$!§.§+!5§(_loc12_);
                  _loc2_ = _loc2_ + _loc14_.getName().toLowerCase() + (_loc8_ < _loc7_ - 1 ? ", " : "");
               }
               else if(_loc12_ >= 1000)
               {
                  _loc14_ = §2$!§.§""2§(_loc12_,Application.userProfile.getRace());
                  _loc2_ = _loc2_ + _loc14_.getName().toLowerCase() + (_loc8_ < _loc7_ - 1 ? ", " : "");
               }
               else
               {
                  _loc14_ = §2$!§.§-m§(_loc12_);
                  _loc15_ = Application.userProfile.§]!&§().§@$E§(_loc12_);
                  if(!_loc15_)
                  {
                     _loc15_ = §>#x§.§-$,§(_loc12_);
                  }
                  if(Boolean(_loc15_) && _loc15_.isInfinite())
                  {
                     _loc13_ = false;
                  }
                  else
                  {
                     _loc2_ = _loc2_ + _loc14_.getName().toLowerCase() + " (" + _loc6_[_loc11_] + ")" + (_loc8_ < _loc7_ - 1 ? ", " : "");
                  }
               }
               _loc8_++;
            }
            if(_loc7_ > 1)
            {
               _loc2_ += ")";
            }
         }
         return _loc2_;
      }
      
      public function §5"p§(param1:§-#I§) : void
      {
         §-"d§.§%"r§.§^H§();
         §8!O§(this.§%$,§(param1),{},this.iconsScheme.bonusIcon);
      }
      
      public function §@>§(param1:ClanMemberTO) : void
      {
         var _loc2_:Object = this.§"$2§();
         _loc2_.userName = §[#;§.§,!k§(param1);
         var _loc3_:String = param1.rank == §6!%§.§2[§ ? this.§ "C§.§"$"§ : this.§ "C§.§^"]§;
         this.§%"A§(§;#]§.format(this.§ "C§.§@p§,{"userName":param1.name}),this.§ "C§.§="+§,_loc3_,_loc2_,this.iconsScheme.promoteRank);
      }
      
      public function §""+§(param1:ClanMemberTO) : void
      {
      }
      
      public function §0$I§(param1:ClanMemberTO) : void
      {
      }
      
      public function §[#j§(param1:int, param2:String) : void
      {
         var _loc3_:Boolean = false;
         var _loc4_:Object = null;
         _loc3_ = param1 == §6!%§.§2[§;
         _loc4_ = {"clanName":param2};
         var _loc5_:String = §;#]§.format(_loc3_ ? this.§ "C§.§1$<§ : this.§ "C§.§!#H§,_loc4_);
         var _loc6_:String = §;#]§.format(_loc3_ ? this.§ "C§.§6$+§ : this.§ "C§.§7"F§,_loc4_);
         this.§%"A§(_loc5_,this.§ "C§.§="+§,_loc6_,{},this.iconsScheme.leaveClan);
      }
      
      public function §3?§() : void
      {
         this.§%"A§(this.§ "C§.§8!K§,this.§ "C§.§="+§,this.§ "C§.§%"^§,this.§"$2§(),this.iconsScheme.medalsExchange);
      }
      
      public function §,#;§() : §2"C§
      {
         return this.iconsScheme;
      }
      
      public function §8!=§() : String
      {
         return §""X§;
      }
   }
}

