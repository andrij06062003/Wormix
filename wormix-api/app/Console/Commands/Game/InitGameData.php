<?php

namespace App\Console\Commands\Game;

use App\Models\Wormix\Craft;
use App\Models\Wormix\DailyBonus;
use App\Models\Wormix\Level;
use App\Models\Wormix\Mission;
use App\Models\Wormix\Race;
use App\Models\Wormix\Reagent;
use App\Models\Wormix\Weapon;
use Illuminate\Console\Command;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\File;
use Mockery\Exception;

class InitGameData extends Command
{
    /**
     * The name and signature of the console command.
     *
     * @var string
     */
    protected $signature = 'game:init';

    /**
     * The console command description.
     *
     * @var string
     */
    protected $description = 'Add base data from game to DB';

    private array $messages = [];

    private function parseNames():void
    {
        $this->info('Parsing strings from items_messages_ru.xml');
        $items_messages_path = resource_path('game/items.messages_ru.xml');
        if(!File::exists($items_messages_path)){
            $this->error("Can't find items.messages_ru.xml in resources");
            return;
        }
        $messages_array = simplexml_load_file($items_messages_path);
        $messages_object = [];
        foreach($messages_array->children() as $message){
            $messages_object[(string)$message['name']] = (string)$message['value'];
        }
        $this->messages = $messages_object;
    }

    private function parseWeapons():void
    {
        $this->info('Parsing from weapons.json');
        $weapons_path = resource_path('game/weapons.json');
        if(!File::exists($weapons_path)){
            $this->error("Can't find weapons.json in resources");
            return;
        }
        $weapons_array = json_decode(file_get_contents($weapons_path), true);
        foreach($weapons_array as $weapon){
            if(array_key_exists('name', $weapon)) {
                DB::beginTransaction();
                try{
                    Weapon::insert(
                        [
                            'id' => $weapon['id'],
                            'name' => @$this->messages[$weapon['name']] ?? $weapon['name'],
                            'hide_in_shop' => array_key_exists('hideInShop', $weapon),
                            'is_starter' => in_array((int)$weapon['id'], config('wormix.starter.weapons')),
                            'price' => @$weapon['price'] ?? 0,
                            'real_price' => @$weapon['realprice'] ?? 0,
                            'required_friends' => @$weapon['requiredFriends'] ?? 0,
                            'required_level' => @$weapon['requiredLevel'] ?? 0,
                            'infinity' => array_key_exists('infinite', $weapon)
                        ]
                    );
                    DB::commit();
                    $this->info($weapon['name']." saved!");
                }
                catch (\Exception $ex){
                    $this->error("Error in {$weapon['id']}: {$ex->getMessage()}");
                    DB::rollBack();
                }

            }elseif(array_key_exists('refId', $weapon)){
                try{
                    DB::beginTransaction();
                    Weapon::insert(
                        [
                            'id' => $weapon['id'],
                            'ref_id' =>  $weapon['refId'],
                            'hide_in_shop' => @$weapon['hideInShop'] ?? true,
                            'price' => $weapon['price'],
                            'required_friends' => @$weapon['requiredFriends'] ?? 0,
                            'required_level' => @$weapon['requiredLevel'] ?? 0,
                        ]
                    );
                    $this->info('Ref '.$weapon['refId']." saved!");
                    DB::commit();
                }catch (Exception $ex){
                    $this->error("Error in {$weapon['id']}: {$ex->getMessage()}");
                    DB::rollBack();
                }
            }
        }
    }

    private function parseHats():void
    {
        $this->info('Parsing hats from hats.json');
        $hats_path = resource_path('game/hats.json');
        if(!File::exists($hats_path)){
            $this->error("Can't find hats.json in resources");
            return;
        }
        $hats_array = json_decode(file_get_contents($hats_path), true);
        foreach($hats_array as $hat){
            try{
                DB::beginTransaction();
                Weapon::insert(
                    [
                        'id' => $hat['id'],
                        'name' => @$this->messages[$hat['name']] ?? $hat['name'],
                        'hide_in_shop' => array_key_exists('hideInShop', $hat),
                        'is_starter' => false,
                        'price' => @$hat['price'] ?? 0,
                        'real_price' => @$hat['realprice'] ?? 0,
                        'required_rating' => @$hat['requiredRating'] ?? 0,
                        'required_level' => @$hat['requiredLevel'] ?? 0,
                        'infinity' => is_bool(@$hat['infinite']) && @$hat['infinite'],
                        'one_day' => is_bool(@$hat['oneDay']) && @$hat['infinite']
                    ]
                );
                DB::commit();
                $this->info("Hat ".$hat['name']." saved!");
            }
            catch (\Exception $ex){
                DB::rollBack();
                $this->error("Error in {$hat['id']}: {$ex->getMessage()}");
            }
        }
    }

    private function parseGifts():void
    {
        $this->info('Parsing gifts from gifts.json');
        $gifts_path = resource_path('game/gifts.json');
        if(!File::exists($gifts_path)){
            $this->error("Can't find gifts.json in resources");
            return;
        }
        $gifts_array = json_decode(file_get_contents($gifts_path), true);
        foreach ($gifts_array as $gift) {
            try{
                DB::beginTransaction();
                DailyBonus::insert([
                    'login_sequence' => $gift['sequence'],
                    'bonus_type' => $gift['type'],
                    'bonus_value' => $gift['value'],
                    'random_gift' => $gift['random']
                ]);
                DB::commit();
                $this->info('Gift added for sequence '.$gift['sequence']);
            }catch (\Exception $ex){
                DB::rollBack();
                $this->error("Error in {$gift['sequence']}: {$ex->getMessage()}");
            }
        }
    }

    private function parseRaces():void
    {
        $this->info('Parsing races from races.json');

        $races_path = resource_path('game/races.json');

        if(!File::exists($races_path)){
            $this->error("Can't find races.json in resources");
            return;
        }

        $races_array = json_decode(file_get_contents($races_path), true);

        foreach($races_array as $race){
            try{
                DB::beginTransaction();
                Race::insert([
                    'race_id' => $race['raceId'],
                    'race_name' => $race['configName'],

                    'price' => $race['price'],
                    'real_price' => $race['realPrice'],

                    'required_level' => $race['requiredLevel'],
                ]);
                $this->info('Saved new race '.$race['configName']);
                DB::commit();
            }catch (Exception $ex){
                DB::rollBack();
                $this->error("Error in {$race['raceId']}: {$ex->getMessage()}");
            }
        }
    }

    private function addStartItems():void
    {
        $this->info('Parsing startings weapons from weapons_start.json');

        $start_weapons_path = resource_path('game/weapons_start.json');

        if(!File::exists($start_weapons_path)){
            $this->error("Can't find weapons_start.json in resources");
            return;
        }
        $start_items = json_decode(file_get_contents($start_weapons_path), true);
        if($start_items == null){
            $this->error("Can't parse weapons_start.json");
        }
        try{
            DB::beginTransaction();
            $update_count = Weapon::query()
                ->whereIn('id', $start_items)
                ->update([
                    'is_starter' => 1
                ]);
            DB::commit();
            $this->info("Set [{$update_count}] items ".json_encode($start_items)." as starter");
        }catch (Exception $exception){
            DB::rollBack();
            $this->error("Error {$exception->getMessage()}");
        }
    }

    private function parseLevelAwards():void
    {
        $this->info('Parsing levels awards from level_awards.json');

        $levels_path = resource_path('game/level_awards.json');

        if(!File::exists($levels_path)){
            $this->error("Can't find weapons_start.json in resources");
            return;
        }

        $levels = json_decode(file_get_contents($levels_path), true);
        foreach($levels as $level){
            try{
                DB::beginTransaction();
                Level::insert([
                    'required_experience' => $level['required_experience'],
                    'max_worms_count' => $level['max_worms_count'],
                    'awards' => json_encode($level['reward_weapons']),
                ]);
                DB::commit();
                $this->info("Added new level {$level['level']}");
            }catch (\Exception $ex){
                DB::rollBack();
                $this->error("Error in {$level['level']}: {$ex->getMessage()}");
            }
        }
    }

    private function parseMissions():void
    {
        $this->info('Parsing missions awards from missions_awards.json');

        $missions_path = resource_path('game/missions_awards.json');
        if(!File::exists($missions_path)){
            $this->error("Can't find missions_awards.json in resources");
            return;
        }

        $missions = json_decode(file_get_contents($missions_path), true);
        DB::beginTransaction();
        try{
            foreach($missions as $mission){
                $m = new Mission();
                $m->mission_id = $mission['id'];
                $m->awards = $mission['awards'];
                $m->required_level = $mission['required_level'];
                $m->save();
            }
            DB::commit();
        }catch (Exception $exception){
            $this->error("Error while adding missions: {$exception->getMessage()}");
            DB::rollBack();
        }
    }

    private function parseCraft():void
    {
        $this->info('Parsing craft recipes from recipes.json');
        $recipes_path = resource_path('game/recipes.json');
        if(!File::exists($recipes_path)){
            $this->error("Can't find recipes.json in resources");
            return;
        }

        $recipes = json_decode(file_get_contents($recipes_path), true);
        foreach($recipes as $recipe) {
            try{
                DB::beginTransaction();
                $craft = new Craft();
                $craft->id = $recipe['id'];
                $craft->description = $recipe['description'];
                $craft->upgrade_id = $recipe['upgradeId'];
                $craft->prev_upgrade_id = @$recipe['prevUpgradeId'];
                $craft->reagents = $recipe['reagents'];
                $craft->level = $recipe['level'];
                $craft->required_level = $recipe['requiredLevel'];
                $craft->save();
                DB::commit();
                $this->info("Added recipe {$recipe['description']}");
            }catch (Exception $exception){
                DB::rollBack();
                $this->error("Error in {$recipe['description']}: {$exception->getMessage()}");
            }
        }
    }

    private function parseReagents():void
    {
        $this->info('Parsing reagents config from reagents.json');

        $reagents_path = resource_path('game/reagents.json');

        if(!File::exists($reagents_path)){
            $this->error("Can't find weapons_start.json in resources");
            return;
        }

        $reagents = json_decode(file_get_contents($reagents_path), true);
        foreach($reagents as $reagent){
            try{
                DB::beginTransaction();
                Reagent::insert([
                    'reagent_id' => $reagent['id'],
                    'name' => $reagent['name'],
                    'reagent_price' => $reagent['price'],
                ]);
                DB::commit();
                $this->info("Added new reagent {$reagent['id']}");
            }catch (\Exception $ex){
                DB::rollBack();
                $this->error("Error in reagent {$reagent['id']}: {$ex->getMessage()}");
            }
        }
    }

    /**
     * Execute the console command.
     */
    public function handle() : void
    {
        $this->parseNames();

        $this->parseWeapons();
        $this->parseHats();
        $this->addStartItems();

        $this->parseGifts();

        $this->parseRaces();

        $this->parseLevelAwards();

        $this->parseReagents();

        $this->parseMissions();

        $this->parseCraft();

        $this->info('SETUP IS COMPLETED');
    }
}
