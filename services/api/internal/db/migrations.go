package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
)

// Migration is one Laravel migration: its file name (as stored in the `migrations`
// table) and the exact SQL Laravel 11's SQLite grammar generated for it.
type Migration struct {
	Name string
	SQL  []string
	// Fn runs after SQL inside the migration's transaction (data migrations).
	Fn func(ctx context.Context, tx *sql.Tx) error
}

const createMigrationsTable = `create table "migrations" ("id" integer primary key autoincrement not null, "migration" varchar not null, "batch" integer not null)`

// Migrations mirrors wormix-api/database/migrations (SQL captured with `artisan migrate --pretend`).
var Migrations = []Migration{
	{Name: "0001_01_01_000000_create_users_table", SQL: []string{
		`create table "users" ("id" integer primary key autoincrement not null, "login" varchar not null, "password" varchar not null, "created_at" datetime, "updated_at" datetime)`,
		`create unique index "i_login_index" on "users" ("login")`,
	}},
	{Name: "0001_01_01_000001_create_cache_table", SQL: []string{
		`create table "cache" ("key" varchar not null, "value" text not null, "expiration" integer not null, primary key ("key"))`,
		`create table "cache_locks" ("key" varchar not null, "owner" varchar not null, "expiration" integer not null, primary key ("key"))`,
	}},
	{Name: "0001_01_01_000002_create_jobs_table", SQL: []string{
		`create table "jobs" ("id" integer primary key autoincrement not null, "queue" varchar not null, "payload" text not null, "attempts" integer not null, "reserved_at" integer, "available_at" integer not null, "created_at" integer not null)`,
		`create index "jobs_queue_index" on "jobs" ("queue")`,
		`create table "job_batches" ("id" varchar not null, "name" varchar not null, "total_jobs" integer not null, "pending_jobs" integer not null, "failed_jobs" integer not null, "failed_job_ids" text not null, "options" text, "cancelled_at" integer, "created_at" integer not null, "finished_at" integer, primary key ("id"))`,
		`create table "failed_jobs" ("id" integer primary key autoincrement not null, "uuid" varchar not null, "connection" text not null, "queue" text not null, "payload" text not null, "exception" text not null, "failed_at" datetime not null default CURRENT_TIMESTAMP)`,
		`create unique index "failed_jobs_uuid_unique" on "failed_jobs" ("uuid")`,
	}},
	{Name: "2024_06_14_140232_create_personal_access_tokens_table", SQL: []string{
		`create table "personal_access_tokens" ("id" integer primary key autoincrement not null, "tokenable_type" varchar not null, "tokenable_id" integer not null, "name" varchar not null, "token" varchar not null, "abilities" text, "last_used_at" datetime, "expires_at" datetime, "created_at" datetime, "updated_at" datetime)`,
		`create index "personal_access_tokens_tokenable_type_tokenable_id_index" on "personal_access_tokens" ("tokenable_type", "tokenable_id")`,
		`create unique index "personal_access_tokens_token_unique" on "personal_access_tokens" ("token")`,
	}},
	{Name: "2024_06_14_170713_create_wormix_user_profiles_table", SQL: []string{
		`create table "wormix_user_profiles" ("user_id" integer not null, "money" integer not null default '450', "real_money" integer not null default '3', "rating" integer not null default '0', "reaction_rate" integer not null default '0', "reagents" text not null default '[]', "recipes" text not null default '[]', "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "users"("id") on delete cascade, primary key ("user_id"))`,
	}},
	{Name: "2024_06_14_174450_create_wormix_worms_data_table", SQL: []string{
		`create table "wormix_worms_data" ("owner_id" integer not null, "armor" integer not null default '1', "attack" integer not null default '1', "level" integer not null default '1', "experience" integer not null default '0', "hat" integer not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("owner_id") references "users"("id") on delete cascade, primary key ("owner_id"))`,
	}},
	{Name: "2024_06_14_174830_create_wormix_weapons_table", SQL: []string{
		`create table "wormix_weapons" ("id" integer not null, "ref_id" integer, "name" varchar, "is_starter" tinyint(1) not null default '0', "hide_in_shop" tinyint(1) not null default '0', "price" integer not null default '0', "real_price" integer not null default '0', "infinity" tinyint(1) not null default '0', "one_day" tinyint(1) not null default '0', "required_friends" integer not null default '0', "required_level" integer not null default '0', "required_rating" integer not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("ref_id") references "wormix_weapons"("id") on delete cascade, primary key ("id"))`,
	}},
	{Name: "2024_06_14_191301_create_wormix_users_weapons_table", SQL: []string{
		`create table "wormix_users_weapons" ("id" integer primary key autoincrement not null, "owner_id" integer not null, "weapon_id" integer not null, "count" integer not null default '-1', "expire_at" integer not null default '-1', "created_at" datetime, "updated_at" datetime, foreign key("owner_id") references "users"("id") on delete cascade, foreign key("weapon_id") references "wormix_weapons"("id") on delete cascade)`,
	}},
	{Name: "2024_06_14_191531_create_notifications_table", SQL: []string{
		`create table "notifications" ("id" varchar not null, "type" varchar not null, "notifiable_type" varchar not null, "notifiable_id" integer not null, "data" text not null, "read_at" datetime, "created_at" datetime, "updated_at" datetime, primary key ("id"))`,
		`create index "notifications_notifiable_type_notifiable_id_index" on "notifications" ("notifiable_type", "notifiable_id")`,
	}},
	{Name: "2024_07_05_182532_create_users_vk_data_table", SQL: []string{
		`create table "users_vk_data" ("user_id" integer not null, "first_name" varchar, "last_name" varchar, "nickname" varchar, "photo" varchar, "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "users"("id") on delete cascade, primary key ("user_id"))`,
	}},
	{Name: "2024_08_10_160106_create_wormix_login_sequence_table", SQL: []string{
		`create table "wormix_login_sequence" ("user_id" integer not null, "last_login" date, "bonus_type" integer not null default '0', "bonus_count" integer not null default '0', "login_sequence" integer not null default '1', "gift_accepted" tinyint(1) not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "users"("id") on delete cascade, primary key ("user_id"))`,
	}},
	{Name: "2024_08_10_160801_create_daily_bonuses_table", SQL: []string{
		`create table "daily_bonuses" ("id" integer primary key autoincrement not null, "login_sequence" integer not null default '1', "bonus_type" integer not null default '0', "bonus_value" integer not null default '0', "random_gift" tinyint(1) not null default '0', "rand_min" integer not null default '1', "rand_max" integer not null default '10', "created_at" datetime, "updated_at" datetime)`,
	}},
	{Name: "2024_08_10_161924_create_wormix_users_battle_info", SQL: []string{
		`create table "wormix_users_battle_info" ("user_id" integer not null, "battles_count" integer not null default '10', "awards" text, "current_battle_id" integer not null default '0', "battle_type" integer not null default '0', "mission_id" integer not null default '0', "last_mission_id" integer not null default '-1', "last_boss_fight_time" integer not null default '0', "last_battle_time" integer not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "users"("id") on delete cascade, primary key ("user_id"))`,
	}},
	{Name: "2024_08_11_150410_create_wormix_races_table", SQL: []string{
		`create table "wormix_races" ("id" integer primary key autoincrement not null, "race_id" integer not null, "race_name" varchar not null, "price" integer not null default '0', "real_price" integer not null default '0', "required_level" integer not null default '0', "created_at" datetime, "updated_at" datetime)`,
		`create unique index "r_game_idx" on "wormix_races" ("race_id")`,
	}},
	{Name: "2024_08_12_185739_create_wormix_levels_table", SQL: []string{
		`create table "wormix_levels" ("id" integer primary key autoincrement not null, "required_experience" integer not null default '0', "max_worms_count" integer not null default '1', "awards" text, "created_at" datetime, "updated_at" datetime)`,
	}},
	{Name: "2024_08_12_185925_create_wormix_missions_table", SQL: []string{
		`create table "wormix_missions" ("id" integer primary key autoincrement not null, "mission_id" integer not null, "required_level" integer not null default '1', "awards" text not null, "created_at" datetime, "updated_at" datetime)`,
		`create unique index "mission_idx" on "wormix_missions" ("mission_id")`,
	}},
	{Name: "2024_08_12_185940_create_wormix_reagents_table", SQL: []string{
		`create table "wormix_reagents" ("id" integer primary key autoincrement not null, "reagent_id" integer not null, "name" varchar, "reagent_price" integer not null default '0', "created_at" datetime, "updated_at" datetime)`,
		`create unique index "reagent_idx" on "wormix_reagents" ("reagent_id")`,
	}},
	{Name: "2024_08_12_190000_create_wormix_craft_table", SQL: []string{
		`create table "wormix_craft" ("id" integer not null, "upgrade_id" integer not null, "prev_upgrade_id" integer, "description" varchar, "required_level" integer not null default '1', "level" integer not null, "reagents" text not null, "created_at" datetime, "updated_at" datetime, primary key ("id"))`,
		`create unique index "upgrade_idx" on "wormix_craft" ("upgrade_id")`,
	}},
	{Name: "2024_08_13_224857_create_wormix_house_actions_table", SQL: []string{
		`create table "wormix_house_actions" ("id" integer primary key autoincrement not null, "user_id" integer not null, "to_user_id" integer not null, "action_type" integer not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "users"("id"), foreign key("to_user_id") references "users"("id"))`,
	}},
	{Name: "2024_08_13_225149_create_wormix_users_teams_table", SQL: []string{
		`create table "wormix_users_teams" ("id" integer primary key autoincrement not null, "user_id" integer not null, "teammate_id" integer not null, "order" integer not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "wormix_user_profiles"("user_id") on delete cascade, foreign key("teammate_id") references "wormix_worms_data"("owner_id") on delete cascade)`,
	}},
	{Name: "2026_11_05_120000_create_wormix_users_achievements_table", SQL: []string{
		`create table "wormix_users_achievements" ("id" integer primary key autoincrement not null, "user_id" integer not null, "achievement_id" integer not null, "value" integer not null default '0', "created_at" datetime, "updated_at" datetime, foreign key("user_id") references "users"("id") on delete cascade)`,
		`create unique index "wormix_users_achievements_user_id_achievement_id_unique" on "wormix_users_achievements" ("user_id", "achievement_id")`,
		`alter table "wormix_user_profiles" add column "invested_award_points" integer not null default '0'`,
	}},
	{Name: "2026_11_06_120000_add_wormix_artifacts", Fn: migrateArtifacts},
	{Name: "2026_11_08_120000_add_wormix_crafted_equip", Fn: migrateCraftedEquip},
	{Name: "2026_11_08_180000_grant_owner_account", Fn: migrateGrantOwner},
	{Name: "2026_11_08_200000_add_wormix_hats_125", Fn: migrateHats125},
	// The 1.25.0 shop sells the Ice Shard, hidden in 1.05.0's.
	{Name: "2026_11_08_210000_sell_wormix_ice_shard", SQL: []string{
		`update "wormix_weapons" set "hide_in_shop" = 0 where "id" = 2019`,
	}},
	{Name: "2026_11_08_220000_add_wormix_teams_inactive", Fn: migrateTeamInactive},
	// The 1.25.0 shop sells weapons 1.05.0 doesn't have: without their rows the API refused every purchase of
	// one (exists:wormix_weapons,id) and the client said "error saving data". Prices and levels from its config.
	{Name: "2026_11_09_120000_add_wormix_weapons_125", SQL: []string{
		`insert or ignore into "wormix_weapons" ("id", "name", "is_starter", "hide_in_shop", "price", "real_price", "infinity", "one_day", "required_friends", "required_level", "required_rating", "created_at", "updated_at") values (81, 'Газовая граната', 0, 0, 3000, 30, 1, 0, 0, 13, 0, datetime('now'), datetime('now')), (82, 'Блокиратор', 0, 0, 3600, 36, 1, 0, 0, 14, 0, datetime('now'), datetime('now')), (83, 'Паразит', 0, 0, 3200, 32, 1, 0, 0, 16, 0, datetime('now'), datetime('now')), (94, 'Лазерный луч', 0, 0, 4200, 42, 1, 0, 0, 20, 0, datetime('now'), datetime('now')), (95, 'Каменюк', 0, 0, 3400, 34, 1, 0, 0, 14, 0, datetime('now'), datetime('now')), (96, 'Вакуумная граната', 0, 0, 5200, 52, 1, 0, 0, 20, 0, datetime('now'), datetime('now')), (105, 'Ледяной дождь', 0, 0, 5400, 54, 1, 0, 0, 15, 0, datetime('now'), datetime('now')), (107, 'Заземлитель', 0, 0, 3700, 37, 1, 0, 0, 16, 0, datetime('now'), datetime('now')), (109, 'Гравимина', 0, 0, 2900, 29, 1, 0, 0, 11, 0, datetime('now'), datetime('now'))`,
	}},
	{Name: "2026_11_09_130000_add_wormix_recipes_125", Fn: migrateRecipes125},
	{Name: "2026_11_09_140000_add_wormix_races_125", Fn: migrateRaces125},
	{Name: "2026_11_09_150000_add_wormix_coop_progress", Fn: migrateCoopProgress},
}

func tableExists(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, name string) (bool, error) {
	var n int
	err := q.QueryRowContext(ctx, `select count(*) from sqlite_master where type = 'table' and name = ?`, name).Scan(&n)
	return n > 0, err
}

// Migrate does what `php artisan migrate --force` did: creates the migrations table
// if needed and runs every migration missing from it, each in its own transaction,
// all in one new batch. It reports whether the migrations table was missing (new DB).
func (d *DB) Migrate(ctx context.Context) (created bool, err error) {
	d.wmu.Lock()
	defer d.wmu.Unlock()

	exists, err := tableExists(ctx, d.DB, "migrations")
	if err != nil {
		return false, err
	}
	if !exists {
		if _, err := d.ExecContext(ctx, createMigrationsTable); err != nil {
			return false, fmt.Errorf("create migrations table: %w", err)
		}
		created = true
	}

	ran := map[string]bool{}
	rows, err := d.QueryContext(ctx, `select "migration" from "migrations" order by "batch" asc, "migration" asc`)
	if err != nil {
		return created, err
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			rows.Close()
			return created, err
		}
		ran[name] = true
	}
	rows.Close()

	var batch int
	if err := d.QueryRowContext(ctx, `select coalesce(max("batch"), 0) from "migrations"`).Scan(&batch); err != nil {
		return created, err
	}
	batch++

	pending := make([]Migration, 0)
	for _, m := range Migrations {
		if !ran[m.Name] {
			pending = append(pending, m)
		}
	}
	sort.SliceStable(pending, func(i, j int) bool { return pending[i].Name < pending[j].Name })

	for _, m := range pending {
		tx, err := d.BeginTx(ctx, nil)
		if err != nil {
			return created, err
		}
		for _, s := range m.SQL {
			if _, err := tx.ExecContext(ctx, s); err != nil {
				tx.Rollback()
				return created, fmt.Errorf("migration %s: %w", m.Name, err)
			}
		}
		if m.Fn != nil {
			if err := m.Fn(ctx, tx); err != nil {
				tx.Rollback()
				return created, fmt.Errorf("migration %s: %w", m.Name, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `insert into "migrations" ("migration", "batch") values (?, ?)`, m.Name, batch); err != nil {
			tx.Rollback()
			return created, err
		}
		if err := tx.Commit(); err != nil {
			return created, err
		}
		log.Printf("migrated %s", m.Name)
	}
	return created, nil
}
