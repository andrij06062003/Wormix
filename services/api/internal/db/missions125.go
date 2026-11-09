package db

import (
	"context"
	"database/sql"
)

// missions125 are the solo bosses of the 1.25.0 client that 1.05.0 doesn't have (21 Dark Knight .. 27
// Symbiote), with the client's level and awards (first win, repeat). Without a row the API refused to
// start or unlock them (exists:wormix_missions) and the core dropped the session.
var missions125 = []struct {
	id, level int64
	awards    string
}{
	{21, 20, `[{"money":0,"experience":140,"real_money":7,"weapons":[[72,8],[80,8],[74,6],[1067,1]]},{"money":130,"experience":70,"real_money":0,"weapons":[[72,3],[80,3],[74,4],[1067,1]]}]`},
	{22, 20, `[{"money":0,"experience":150,"real_money":7,"weapons":[[72,10],[80,10],[93,4],[1096,1]]},{"money":140,"experience":75,"real_money":0,"weapons":[[72,3],[80,3],[93,2],[1096,1]]}]`},
	{23, 20, `[{"money":0,"experience":150,"real_money":7,"weapons":[[72,10],[80,10],[84,4],[1103,1]]},{"money":140,"experience":75,"real_money":0,"weapons":[[72,3],[80,3],[84,2],[1103,1]]}]`},
	{24, 20, `[{"money":0,"experience":150,"real_money":7,"weapons":[[72,10],[80,10],[75,6],[2018,1]]},{"money":150,"experience":75,"real_money":0,"weapons":[[72,3],[80,3],[75,4],[2018,1]]}]`},
	{25, 24, `[{"money":0,"experience":160,"real_money":8,"weapons":[[72,10],[80,10],[103,6],[1135,1]]},{"money":160,"experience":80,"real_money":0,"weapons":[[72,3],[80,3],[103,3],[1135,1]]}]`},
	{26, 24, `[{"money":0,"experience":160,"real_money":8,"weapons":[[72,10],[80,10],[101,6],[2026,1]]},{"money":160,"experience":80,"real_money":0,"weapons":[[72,3],[80,3],[101,3],[2026,1]]}]`},
	{27, 24, `[{"money":0,"experience":160,"real_money":8,"weapons":[[72,10],[80,10],[104,4],[2027,1]]},{"money":160,"experience":80,"real_money":0,"weapons":[[72,3],[80,3],[104,2],[2027,1]]}]`},
}

// migrateMissions125 is 2026_11_09_160000_add_wormix_missions_125. A new database gets the missions from
// game:init, after missions_awards.json.
func migrateMissions125(ctx context.Context, tx *sql.Tx) error {
	var n int
	if err := tx.QueryRowContext(ctx, `select count(*) from "wormix_missions"`).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return nil
	}
	return applyMissions125(ctx, tx)
}

// applyMissions125 inserts the missing 1.25.0 missions.
func applyMissions125(ctx context.Context, q execQuerier) error {
	now := Now()
	for _, m := range missions125 {
		if _, err := q.ExecContext(ctx, `insert or ignore into "wormix_missions" ("mission_id", "awards", "required_level", "updated_at", "created_at") values (?, ?, ?, ?, ?)`,
			m.id, m.awards, m.level, now, now); err != nil {
			return err
		}
	}
	return nil
}
