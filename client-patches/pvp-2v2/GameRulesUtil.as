package ru.pragmatix.wormix.game.rules
{
   import flash.geom.Point;
   import ru.pragmatix.flox.utils.Math2;
   import ru.pragmatix.wormix.controller.Singletons;
   import ru.pragmatix.wormix.loader.level.SquadPosition;
   import ru.pragmatix.wormix.model.battle.BattleStartInfo;
   import ru.pragmatix.wormix.model.battle.BattleType;
   import ru.pragmatix.wormix.model.user.UserProfile;
   import ru.pragmatix.wormix.model.user.UserProfileHelper;
   
   public class GameRulesUtil
   {
      
      private static const OPTIMAL_WORM_POSITION:int = 700;
      
      private static const HP_TEAM_DIFFERENCE:int = 120;
      
      public function GameRulesUtil()
      {
         super();
      }
      
      public static function initPositions(info:BattleStartInfo) : WormPositions
      {
         var positions:Array = Singletons.mapController.getRandomWormPositions(info.mapConfig,info.seed);
         var pos:WormPositions = new WormPositions();
         if(info.isTeamBattle())
         {
            setTeamPositions(info,positions,pos);
            return pos;
         }
         if(positions[0] is SquadPosition)
         {
            addPositions(Application.userProfile,positions[info.myTurn ? 0 : 1],pos);
            addPositions(info.enemyProfile,positions[info.myTurn ? 1 : 0],pos);
         }
         else
         {
            setWormsPositions(info,positions,pos);
         }
         return pos;
      }
      
      private static function setWormsPositions(info:BattleStartInfo, positions:Array, result:WormPositions) : void
      {
         var positionsIndex:int = 0;
         var firstTeam:UserProfile = null;
         var secondTeam:UserProfile = null;
         var indexPriPos:int = 0;
         var indexSecPos:int = 0;
         var distance:int = 0;
         var saveMinDistance:int = OPTIMAL_WORM_POSITION;
         var indexMinimalValue:int = 0;
         if(info.myTurn)
         {
            firstTeam = Application.userProfile;
            secondTeam = info.enemyProfile;
         }
         else
         {
            firstTeam = info.enemyProfile;
            secondTeam = Application.userProfile;
         }
         var firstWorms:Array = firstTeam.getTeam();
         var secondWorms:Array = secondTeam.getTeam();
         if(info.battleType != BattleType.PVP_WAGER && info.battleType != BattleType.PVP_WITH_FRIEND)
         {
            secondWorms = sortTeamOnDecHP(secondWorms);
         }
         result.setPositionByProfile(firstWorms[0],firstTeam,positions[0][0]);
         for(var pos:int = 1; pos < positions[0].length; pos++)
         {
            distance = Math.abs(Math2.distance(positions[0][0].x - positions[0][pos].x,positions[0][0].y - positions[0][pos].y) - OPTIMAL_WORM_POSITION);
            if(saveMinDistance > distance)
            {
               saveMinDistance = distance;
               indexMinimalValue = pos;
            }
         }
         result.setPositionByProfile(secondWorms[0],secondTeam,positions[0][indexMinimalValue]);
         positions[0].splice(indexMinimalValue,1);
         positions[0].shift();
         for(var i:int = 1; i < Math.max(firstWorms.length,secondWorms.length); i++)
         {
            if(Boolean(firstWorms[i]) || Boolean(secondWorms[i]))
            {
               if(Boolean(firstWorms[i]) && Boolean(secondWorms[i]))
               {
                  result.setPositionByProfile(firstWorms[i],firstTeam,positions[0][indexPriPos++]);
                  result.setPositionByProfile(secondWorms[i],secondTeam,positions[0][indexPriPos++]);
               }
               else if(Boolean(firstWorms[i]))
               {
                  positionsIndex = firstTeam.getTeamHp() - secondTeam.getTeamHp() > HP_TEAM_DIFFERENCE ? 1 : 0;
                  result.setPositionByProfile(firstWorms[i],firstTeam,positions[positionsIndex][Boolean(positionsIndex) ? indexSecPos++ : indexPriPos++]);
               }
               else
               {
                  positionsIndex = secondTeam.getTeamHp() - firstTeam.getTeamHp() > HP_TEAM_DIFFERENCE ? 1 : 0;
                  result.setPositionByProfile(secondWorms[i],secondTeam,positions[positionsIndex][Boolean(positionsIndex) ? indexSecPos++ : indexPriPos++]);
               }
            }
         }
      }
      
      // Patch (2v2): places the worms of all four players. Every game must get the same result, so it depends
      // only on the map, the seed and the players in the server's turn order. Worms are taken in turns
      // (each player's 1st worm, then each player's 2nd...). On maps with a spawn area per squad each team gets
      // one area; otherwise everyone shares the primary positions, then the secondary ones. When a map runs out
      // of positions, the used ones are reused a bit higher up (the worm falls down).
      private static function setTeamPositions(info:BattleStartInfo, positions:Array, result:WormPositions) : void
      {
         var lists:Array = null;
         var used:Array = null;
         var i:int = 0;
         var w:int = 0;
         var maxWorms:int = 0;
         var profile:UserProfile = null;
         var team:Array = null;
         var li:int = 0;
         var list:Array = null;
         var n:int = 0;
         var p:Point = null;
         var round:int = 0;
         var all:Array = (positions[0] as Array || []).concat(positions[1] as Array || []);
         if(all.length == 0)
         {
            all = [new Point(0,0)];
         }
         if(positions[0] is Array && positions[1] is Array && info.mapConfig.squadsPositions && info.mapConfig.squadsPositions.length > 0)
         {
            lists = [positions[0],positions[1]];
         }
         else
         {
            lists = [all,all];
         }
         used = [0,0];
         for(i = 0; i < info.teamProfiles.length; i++)
         {
            maxWorms = Math.max(maxWorms,UserProfile(info.teamProfiles[i]).getTeam().length);
         }
         for(w = 0; w < maxWorms; w++)
         {
            for(i = 0; i < info.teamProfiles.length; i++)
            {
               profile = UserProfile(info.teamProfiles[i]);
               team = profile.getTeam();
               if(w >= team.length)
               {
                  continue;
               }
               li = lists[0] == lists[1] ? 0 : (int(info.teamIds[i]) == 1 ? 0 : 1);
               list = lists[li];
               if(list.length == 0)
               {
                  list = all;
               }
               n = int(used[li]);
               used[li] = n + 1;
               p = Point(list[n % list.length]);
               round = int(n / list.length);
               if(round > 0)
               {
                  p = new Point(p.x + 25 * round,p.y - 40 * round);
               }
               result.setPositionByProfile(team[w],profile,p);
            }
         }
      }
      
      [deprecated]
      private static function addPositions(profile:UserProfile, positions:Array, result:WormPositions) : void
      {
         var worm:UserProfile = null;
         var i:uint = 0;
         for each(worm in profile.getTeam())
         {
            result.setPositionByProfile(worm,profile,positions[i++]);
         }
      }
      
      private static function sortTeamOnDecHP(team:Array) : Array
      {
         team.sort(UserProfileHelper.sortOnHp);
         return team;
      }
   }
}

