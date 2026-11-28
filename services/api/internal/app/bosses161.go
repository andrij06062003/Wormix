package app

import "github.com/andrij06062003/Wormix/services/api/internal/php"

// bosses161 are the 1.61.7 client's solo bosses 1-31 (its boss table, §6&'§ in the decompiled client): the
// level it asks for and its awards, [first win, a win after that]. The 1.61.7 client shows these in its boss
// window, but wormix_missions has the 1.25.0 client's bosses (other levels, other weapons) and no 28-31
// (Blacksmith, Soul Master, Rebel Robot, Devotee): their unlock was refused (the client's "error code 1") and
// a win gave other awards than the ones shown. The core marks a 1.61.7 client's requests ("Client": 161), and
// for them the API takes the bosses from here. The items are [id, count]; ids from 1000 are hats and
// artifacts, in wormix_weapons since 2026_11_18_140000_shop_161.
var bosses161 = []struct {
	id, level int64
	awards    string
}{
	{1, 6, `[{"money":90,"experience":50,"real_money":0,"weapons":[[1003,1],[66,4],[56,2],[71,2]]},{"money":60,"experience":25,"real_money":0,"weapons":[[1003,1],[63,2],[56,1],[71,1]]}]`},
	{2, 6, `[{"money":90,"experience":50,"real_money":0,"weapons":[[1003,1],[68,4],[66,2],[32,2]]},{"money":60,"experience":25,"real_money":0,"weapons":[[1003,1],[68,2],[66,1],[32,1]]}]`},
	{3, 6, `[{"money":90,"experience":50,"real_money":0,"weapons":[[1003,1],[107,2],[62,4],[105,2]]},{"money":60,"experience":25,"real_money":0,"weapons":[[1003,1],[107,1],[62,2],[105,1]]}]`},
	{4, 7, `[{"money":90,"experience":50,"real_money":0,"weapons":[[1003,1],[78,2],[51,2],[65,2]]},{"money":60,"experience":25,"real_money":0,"weapons":[[1003,1],[78,1],[51,1],[65,1]]}]`},
	{5, 8, `[{"money":90,"experience":50,"real_money":0,"weapons":[[1003,1],[116,2],[72,2],[25,2]]},{"money":60,"experience":25,"real_money":0,"weapons":[[1003,1],[116,1],[72,1],[25,1]]}]`},
	{6, 9, `[{"money":0,"experience":60,"real_money":1,"weapons":[[1003,1],[80,2],[76,4],[27,2]]},{"money":70,"experience":25,"real_money":0,"weapons":[[1003,1],[80,1],[76,2],[27,1]]}]`},
	{7, 10, `[{"money":0,"experience":60,"real_money":1,"weapons":[[1003,1],[72,2],[66,2],[28,2]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1003,1],[72,1],[66,1],[28,1]]}]`},
	{8, 11, `[{"money":0,"experience":60,"real_money":1,"weapons":[[1003,1],[56,4],[80,2],[86,4]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1003,1],[56,2],[80,1],[86,2]]}]`},
	{9, 12, `[{"money":0,"experience":60,"real_money":1,"weapons":[[1037,1],[72,2],[80,2],[64,4]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1037,1],[72,1],[80,1],[64,2]]}]`},
	{10, 12, `[{"money":0,"experience":60,"real_money":1,"weapons":[[1036,1],[72,2],[80,2],[131,2]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1036,1],[72,1],[80,1],[131,1]]}]`},
	{11, 12, `[{"money":0,"experience":60,"real_money":2,"weapons":[[1038,1],[72,2],[80,2],[24,2]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1038,1],[72,1],[80,1],[24,1]]}]`},
	{12, 12, `[{"money":0,"experience":60,"real_money":2,"weapons":[[1035,1],[72,2],[80,2],[58,2]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1035,1],[72,1],[80,1],[58,1]]}]`},
	{13, 12, `[{"money":0,"experience":60,"real_money":2,"weapons":[[1039,1],[72,2],[80,2],[43,2]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1039,1],[72,1],[80,1],[43,1]]}]`},
	{14, 12, `[{"money":0,"experience":60,"real_money":2,"weapons":[[1049,1],[72,4],[80,4],[67,2]]},{"money":80,"experience":30,"real_money":0,"weapons":[[1049,1],[72,2],[80,2],[67,1]]}]`},
	{15, 15, `[{"money":0,"experience":80,"real_money":2,"weapons":[[1044,1],[80,4],[72,4],[28,4]]},{"money":90,"experience":40,"real_money":0,"weapons":[[1044,1],[72,2],[80,2],[28,2]]}]`},
	{16, 15, `[{"money":0,"experience":80,"real_money":3,"weapons":[[1045,1],[1046,1],[80,6],[27,6]]},{"money":90,"experience":40,"real_money":0,"weapons":[[1045,1],[1046,1],[72,3],[27,3]]}]`},
	{17, 18, `[{"money":0,"experience":100,"real_money":5,"weapons":[[1043,1],[80,4],[72,4],[25,4]]},{"money":100,"experience":50,"real_money":0,"weapons":[[1043,1],[72,2],[80,2],[25,2]]}]`},
	{18, 18, `[{"money":0,"experience":100,"real_money":5,"weapons":[[1040,1],[80,4],[72,4],[129,2]]},{"money":100,"experience":50,"real_money":0,"weapons":[[1040,1],[72,2],[80,2],[129,1]]}]`},
	{19, 18, `[{"money":0,"experience":120,"real_money":5,"weapons":[[1059,1],[72,4],[80,4],[118,2]]},{"money":120,"experience":60,"real_money":0,"weapons":[[1059,1],[72,2],[80,2],[118,1]]}]`},
	{20, 18, `[{"money":0,"experience":130,"real_money":5,"weapons":[[1060,1],[72,4],[80,4],[36,4]]},{"money":130,"experience":65,"real_money":0,"weapons":[[1060,1],[72,2],[80,2],[36,2]]}]`},
	{21, 20, `[{"money":0,"experience":140,"real_money":7,"weapons":[[1067,1],[72,4],[80,4],[70,4]]},{"money":130,"experience":70,"real_money":0,"weapons":[[1067,1],[72,2],[80,2],[70,2]]}]`},
	{22, 20, `[{"money":0,"experience":150,"real_money":7,"weapons":[[1096,1],[72,4],[80,4],[67,4]]},{"money":140,"experience":75,"real_money":0,"weapons":[[1096,1],[72,2],[80,2],[67,2]]}]`},
	{23, 20, `[{"money":0,"experience":150,"real_money":7,"weapons":[[1103,1],[72,4],[80,4],[131,4]]},{"money":140,"experience":75,"real_money":0,"weapons":[[1103,1],[72,2],[80,2],[131,2]]}]`},
	{24, 20, `[{"money":0,"experience":150,"real_money":7,"weapons":[[2018,1],[72,4],[80,4],[114,4]]},{"money":150,"experience":75,"real_money":0,"weapons":[[2018,1],[72,2],[80,2],[114,2]]}]`},
	{25, 24, `[{"money":0,"experience":160,"real_money":8,"weapons":[[1135,1],[72,6],[80,6],[129,4]]},{"money":160,"experience":80,"real_money":0,"weapons":[[1135,1],[72,3],[80,3],[129,2]]}]`},
	{26, 24, `[{"money":0,"experience":160,"real_money":8,"weapons":[[2026,1],[72,6],[80,6],[115,2]]},{"money":160,"experience":80,"real_money":0,"weapons":[[2026,1],[72,3],[80,3],[115,1]]}]`},
	{27, 24, `[{"money":0,"experience":170,"real_money":8,"weapons":[[2027,1],[72,6],[80,6],[113,4]]},{"money":170,"experience":85,"real_money":0,"weapons":[[2027,1],[72,3],[80,3],[113,2]]}]`},
	{28, 24, `[{"money":0,"experience":170,"real_money":8,"weapons":[[1146,1],[72,6],[80,6],[118,4]]},{"money":170,"experience":85,"real_money":0,"weapons":[[1146,1],[72,3],[80,3],[118,2]]}]`},
	{29, 24, `[{"money":0,"experience":180,"real_money":9,"weapons":[[2039,1],[72,6],[80,6],[84,4]]},{"money":180,"experience":90,"real_money":0,"weapons":[[2039,1],[72,3],[80,3],[84,2]]}]`},
	{30, 24, `[{"money":0,"experience":180,"real_money":9,"weapons":[[1149,1],[72,6],[80,6],[77,4]]},{"money":180,"experience":90,"real_money":0,"weapons":[[1149,1],[72,3],[80,3],[77,2]]}]`},
	{31, 26, `[{"money":0,"experience":190,"real_money":10,"weapons":[[1109,1],[72,6],[80,6],[130,4]]},{"money":190,"experience":95,"real_money":0,"weapons":[[1109,1],[72,3],[80,3],[130,2]]}]`},
}

// client161 reports whether the request is a 1.61.7 client's (the core adds "Client": 161).
func (c *Ctx) client161() bool {
	return php.ToInt(c.JSON("Client")) == 161
}

// mission is the solo boss (or training mission) mid: for a 1.61.7 client its bosses161 row, else
// wormix_missions' (nil when there is none).
func (c *Ctx) mission(mid any) *Model {
	if c.client161() {
		id := php.ToInt(mid)
		for _, b := range bosses161 {
			if b.id == id {
				m := New(defMission)
				m.SetRaw("mission_id", b.id)
				m.SetRaw("required_level", b.level)
				m.SetRaw("awards", b.awards)
				return m
			}
		}
	}
	return c.First(defMission, `where "mission_id" = ?`, mid)
}
