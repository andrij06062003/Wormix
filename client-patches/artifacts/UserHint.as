package ru.pragmatix.wormix.gui.avatarmenu
{
   import flash.display.DisplayObjectContainer;
   import flash.display.MovieClip;
   import flash.text.TextField;
   import ru.pragmatix.flox.gui.core.UITexts;
   import ru.pragmatix.wormix.gui.Messages;
   import ru.pragmatix.wormix.model.items.HatRecord;
   import ru.pragmatix.wormix.model.items.ItemsStore;
   import ru.pragmatix.wormix.model.race.Race;
   import ru.pragmatix.wormix.model.race.RaceRecord;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   import ru.pragmatix.wormix.objects.graphics.CharGraphics;
   import ru.pragmatix.wormix.settings.RatingSettings;
   
   public class UserHint
   {
      
      private static var isInit:Boolean = false;
      
      private static const DEFAULT_LABEL:String = "N/A";
      
      public function UserHint()
      {
         super();
      }
      
      public static function setUserData(p:UserProfile, panel:MovieClip) : void
      {
         if(Boolean(panel.userName))
         {
            if(Boolean(p.getTeamName()))
            {
               TextField(panel.userName).text = p.getTeamName();
            }
            else
            {
               TextField(panel.userName).text = DEFAULT_LABEL;
            }
         }
         if(Boolean(panel.userName_short))
         {
            if(Boolean(p))
            {
               TextField(panel.userName_short).text = UserProfileHelper.getShortName(p);
            }
            else
            {
               TextField(panel.userName_short).text = DEFAULT_LABEL;
            }
         }
         if(Boolean(panel.level))
         {
            if(Boolean(p.getLevel()))
            {
               TextField(panel.level).text = p.getLevel().toString();
            }
            else
            {
               TextField(panel.level).text = DEFAULT_LABEL;
            }
         }
         if(Boolean(panel.hp))
         {
            if(Boolean(p.getHp()))
            {
               TextField(panel.hp).text = UserProfileHelper.getValueIncludeStats(p.getHp(),p.getBonusStats().hp);
            }
            else
            {
               TextField(panel.hp).text = DEFAULT_LABEL;
            }
            UITexts.setLabelTextTF(panel.hpLabel,Messages.HP);
         }
         if(Boolean(panel.teamHp))
         {
            if(Boolean(p.getTeamHp()))
            {
               TextField(panel.teamHp).text = p.getTeamHp().toString();
            }
            else
            {
               TextField(panel.teamHp).text = DEFAULT_LABEL;
            }
            TextField(panel.teamHpLabel).text = Messages.TEAM_HP_LABEL;
         }
         if(Boolean(panel.teamSize))
         {
            if(Boolean(p.getTeam()))
            {
               TextField(panel.teamSize).text = p.getTeam().length.toString();
            }
            else
            {
               TextField(panel.teamSize).text = DEFAULT_LABEL;
            }
            TextField(panel.teamLabel).text = Messages.TEAM_LABEL;
            UITexts.setLabelTextTF(panel.teamLabel,Messages.TEAM_LABEL);
         }
         if(Boolean(panel.rating))
         {
            TextField(panel.rating).text = RatingSettings.getRating(p);
            UITexts.setLabelTextTF(panel.ratingLabel,Messages.RATING);
         }
         if(Boolean(panel.armor))
         {
            TextField(panel.armor).text = UserProfileHelper.getValueIncludeStats(p.getArmor(),p.getBonusStats().armor);
            UITexts.setLabelTextTF(panel.armorLabel,Messages.ARMOR);
         }
         if(Boolean(panel.attack))
         {
            TextField(panel.attack).text = UserProfileHelper.getValueIncludeStats(p.getAttack(),p.getBonusStats().attack);
            UITexts.setLabelTextTF(panel.attackLabel,Messages.ATTACK);
         }
         if(Boolean(panel.money))
         {
            TextField(panel.money).text = p.getIngameMoney().toString();
         }
         if(Boolean(panel.realMoney))
         {
            TextField(panel.realMoney).text = p.getRealMoney().toString();
         }
         if(Boolean(panel.reactionRate))
         {
            TextField(panel.reactionRate).text = p.getReaction().toString();
         }
         if(Boolean(panel.race))
         {
            TextField(panel.race).text = p.getCharType().getName();
            UITexts.setLabelTextTF(panel.raceLabel,Messages.RACE);
         }
      }
      
      public static function getCharIcon(charIcon:CharGraphics, p:UserProfile, container:DisplayObjectContainer, race:RaceRecord = null, hat:HatRecord = null) : CharGraphics
      {
         var raceRecord:RaceRecord = null;
         var charGr:CharGraphics = null;
         var clip:MovieClip = null;
         if(Boolean(p) && Boolean(container))
         {
            if(Boolean(charIcon) && container.getChildIndex(charIcon.getClip()) != -1)
            {
               container.removeChild(charIcon.getClip());
            }
            raceRecord = Boolean(race) ? race : p.getCharType();
            charGr = new CharGraphics(raceRecord);
            clip = charGr.getClip();
            clip.scaleX = clip.scaleY = container.width / 40;
            // hat may be an artifact (the shop shows what we're buying): it goes in the hand and
            // the worm keeps its own hat.
            if(Boolean(hat) && hat.isArtifact)
            {
               charGr.setArtifact(hat);
               hat = null;
            }
            else
            {
               charGr.setArtifact(p.getArtifact());
            }
            charGr.setHat(Boolean(hat) ? hat : (Boolean(p.getHat()) ? ItemsStore.getHatById(p.getHat().getId(),raceRecord) : null));
            clip.x = container.width / 2;
            container.addChild(clip);
            clip.y = clip.parent.getBounds(clip.parent).bottom - clip.height - clip.getBounds(clip.parent).top;
            if(charGr.getCharRecord().equals(Race.WORM.getRaceRecord()))
            {
               MovieClip(charGr.getClip().parent.parent).play();
               charGr.stay();
            }
            else
            {
               MovieClip(charGr.getClip().parent.parent).stop();
               charGr.breath(false);
            }
            return charGr;
         }
         return null;
      }
   }
}

