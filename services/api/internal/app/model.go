package app

import (
	"database/sql"
	"strings"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/db"
	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// Def describes an Eloquent model class: table, primary key, casts, hidden attributes.
type Def struct {
	Table  string
	Key    string
	Casts  map[string]string // array | json | boolean | datetime | integer
	Hidden map[string]bool
}

var (
	defUser        = &Def{Table: "users", Key: "id", Hidden: map[string]bool{"password": true}}
	defSocial      = &Def{Table: "users_vk_data", Key: "user_id"}
	defProfile     = &Def{Table: "wormix_user_profiles", Key: "user_id", Casts: map[string]string{"reagents": "array", "recipes": "array"}}
	defWorm        = &Def{Table: "wormix_worms_data", Key: "owner_id"}
	defWeapon      = &Def{Table: "wormix_weapons", Key: "id"}
	defUserWeapon  = &Def{Table: "wormix_users_weapons", Key: "id"}
	defBattleInfo  = &Def{Table: "wormix_users_battle_info", Key: "user_id", Casts: map[string]string{"awards": "array"}}
	defLoginSeq    = &Def{Table: "wormix_login_sequence", Key: "user_id", Casts: map[string]string{"last_login": "datetime", "gift_accepted": "boolean"}}
	defDailyBonus  = &Def{Table: "daily_bonuses", Key: "id", Casts: map[string]string{"random_gift": "boolean"}}
	defRace        = &Def{Table: "wormix_races", Key: "id"}
	defLevel       = &Def{Table: "wormix_levels", Key: "id", Casts: map[string]string{"awards": "json"}}
	defMission     = &Def{Table: "wormix_missions", Key: "id", Casts: map[string]string{"awards": "array"}}
	defReagent     = &Def{Table: "wormix_reagents", Key: "id"}
	defCraft       = &Def{Table: "wormix_craft", Key: "id", Casts: map[string]string{"reagents": "array"}}
	defHouseAction = &Def{Table: "wormix_house_actions", Key: "id", Casts: map[string]string{"action_type": "integer"}}
	defTeam        = &Def{Table: "wormix_users_teams", Key: "id"}
	defAchievement = &Def{Table: "wormix_users_achievements", Key: "id"}
	defToken       = &Def{Table: "personal_access_tokens", Key: "id", Casts: map[string]string{"abilities": "json", "last_used_at": "datetime", "expires_at": "datetime"}, Hidden: map[string]bool{"token": true}}
)

type relation struct {
	name string
	val  any // *Model, []*Model or nil
}

// Model is a loaded or new row with Eloquent attribute/dirty semantics.
type Model struct {
	Def       *Def
	cols      []string
	attrs     map[string]any
	orig      map[string]any
	Exists    bool
	relations []relation
}

// New creates an unsaved model.
func New(def *Def) *Model {
	return &Model{Def: def, attrs: map[string]any{}, orig: map[string]any{}}
}

// Get returns the raw attribute value.
func (m *Model) Get(col string) any { return m.attrs[col] }

// Has reports whether the attribute is set.
func (m *Model) Has(col string) bool { _, ok := m.attrs[col]; return ok }

// Set assigns an attribute (json casts are encoded like Eloquent does).
func (m *Model) Set(col string, v any) {
	if c := m.Def.Casts[col]; (c == "array" || c == "json") && v != nil {
		v = php.MustJSON(v)
	}
	if c := m.Def.Casts[col]; c == "datetime" {
		if t, ok := v.(time.Time); ok {
			v = t.UTC().Format("2006-01-02 15:04:05")
		}
	}
	m.SetRaw(col, v)
}

// SetRaw assigns without casting.
func (m *Model) SetRaw(col string, v any) {
	if _, ok := m.attrs[col]; !ok {
		m.cols = append(m.cols, col)
	}
	m.attrs[col] = v
}

// Arr returns a json-cast attribute decoded (json_decode(..., true)); nil stays nil.
func (m *Model) Arr(col string) any {
	v := m.attrs[col]
	if v == nil {
		return nil
	}
	s, ok := v.(string)
	if !ok {
		return v
	}
	d, ok := php.JSONDecode([]byte(s))
	if !ok {
		return nil
	}
	return d
}

// ArrOrEmpty is the decoded attribute as *Arr (PHP code that treats it as an array).
func (m *Model) ArrList(col string) *php.Arr {
	if a, ok := m.Arr(col).(*php.Arr); ok {
		return a
	}
	return nil
}

// Bool is the boolean cast.
func (m *Model) Bool(col string) bool { return php.ToBool(m.attrs[col]) }

// Fill sets fillable attributes from an array (silently discarding others).
func (m *Model) Fill(data any, fillable ...string) {
	a, ok := data.(*php.Arr)
	if !ok {
		php.Fail("Argument #1 ($attributes) must be of type array, %s given", php.TypeName(data))
	}
	for _, k := range a.Keys() {
		ks, _ := k.(string)
		for _, f := range fillable {
			if f == ks {
				m.Set(f, a.Val(k))
			}
		}
	}
}

func (m *Model) isDateAttr(col string) bool {
	return col == "created_at" || col == "updated_at" || m.Def.Casts[col] == "datetime"
}

func (m *Model) equivalent(col string) bool {
	orig, ok := m.orig[col]
	if !ok {
		return false
	}
	cur := m.attrs[col]
	if php.StrictEq(cur, orig) {
		return true
	}
	if cur == nil {
		return false
	}
	if m.isDateAttr(col) {
		a, ea := fromDateTime(cur)
		b, eb := fromDateTime(orig)
		return ea == nil && eb == nil && a == b
	}
	switch m.Def.Casts[col] {
	case "array", "json":
		return php.StrictEq(decodeCast(cur), decodeCast(orig))
	case "boolean":
		return php.ToBool(cur) == php.ToBool(orig)
	case "integer":
		if orig == nil {
			return false
		}
		return php.ToInt(cur) == php.ToInt(orig)
	}
	return php.IsNumeric(cur) && php.IsNumeric(orig) && php.ToString(cur) == php.ToString(orig)
}

func decodeCast(v any) any {
	s, ok := v.(string)
	if !ok {
		return v
	}
	d, ok := php.JSONDecode([]byte(s))
	if !ok {
		return nil
	}
	return d
}

// Dirty returns the changed columns in attribute order.
func (m *Model) Dirty() []string {
	var out []string
	for _, c := range m.cols {
		if !m.equivalent(c) {
			out = append(out, c)
		}
	}
	return out
}

// IsDirty reports whether any of cols (or any column) changed.
func (m *Model) IsDirty(cols ...string) bool {
	d := m.Dirty()
	if len(cols) == 0 {
		return len(d) > 0
	}
	for _, x := range d {
		for _, c := range cols {
			if x == c {
				return true
			}
		}
	}
	return false
}

func (m *Model) syncOriginal() {
	m.orig = make(map[string]any, len(m.attrs))
	for k, v := range m.attrs {
		m.orig[k] = v
	}
}

// Save is Eloquent save(): insert (with timestamps) or update of dirty columns.
// It returns true when an UPDATE was executed (the "updated" event fires).
func (c *Ctx) Save(m *Model) bool {
	now := db.Now()
	if !m.Exists {
		if !m.IsDirty("updated_at") {
			m.SetRaw("updated_at", now)
		}
		if !m.IsDirty("created_at") {
			m.SetRaw("created_at", now)
		}
		cols := make([]string, 0, len(m.cols))
		args := make([]any, 0, len(m.cols))
		for _, col := range m.cols {
			cols = append(cols, `"`+col+`"`)
			args = append(args, db.Bind(m.attrs[col]))
		}
		q := `insert into "` + m.Def.Table + `" (` + strings.Join(cols, ", ") + `) values (` + strings.TrimSuffix(strings.Repeat("?, ", len(cols)), ", ") + `)`
		res := c.exec(q, args...)
		id, err := res.LastInsertId()
		if err != nil {
			panic(err)
		}
		m.SetRaw(m.Def.Key, id)
		m.Exists = true
		m.syncOriginal()
		return false
	}
	if !m.IsDirty() {
		return false
	}
	if !m.IsDirty("updated_at") {
		m.SetRaw("updated_at", now)
	}
	dirty := m.Dirty()
	sets := make([]string, len(dirty))
	args := make([]any, 0, len(dirty)+1)
	for i, col := range dirty {
		sets[i] = `"` + col + `" = ?`
		args = append(args, db.Bind(m.attrs[col]))
	}
	key, ok := m.orig[m.Def.Key]
	if !ok {
		key = m.attrs[m.Def.Key]
	}
	args = append(args, db.Bind(key))
	c.exec(`update "`+m.Def.Table+`" set `+strings.Join(sets, ", ")+` where "`+m.Def.Key+`" = ?`, args...)
	return true
}

// SaveSynced finishes a save from an observer (syncs original afterwards).
func (c *Ctx) SaveAndSync(m *Model) bool {
	u := c.Save(m)
	m.syncOriginal()
	return u
}

// Delete deletes the model row.
func (c *Ctx) Delete(m *Model) {
	c.exec(`delete from "`+m.Def.Table+`" where "`+m.Def.Key+`" = ?`, db.Bind(m.attrs[m.Def.Key]))
}

// SetRelation attaches a loaded relation (for toArray).
func (m *Model) SetRelation(name string, v any) {
	for i := range m.relations {
		if m.relations[i].name == name {
			m.relations[i].val = v
			return
		}
	}
	m.relations = append(m.relations, relation{name, v})
}

// ToArray is Model::toArray().
func (m *Model) ToArray() *php.Arr {
	out := php.NewArr()
	for _, col := range m.cols {
		if m.Def.Hidden[col] {
			continue
		}
		v := m.attrs[col]
		switch {
		case m.isDateAttr(col):
			if v != nil {
				t, err := asDateTime(v)
				if err != nil {
					php.Fail("%v", err)
				}
				v = t.UTC().Format("2006-01-02T15:04:05.000000Z")
			}
		case m.Def.Casts[col] == "array" || m.Def.Casts[col] == "json":
			v = decodeCast(v)
		case m.Def.Casts[col] == "boolean":
			if v != nil {
				v = php.ToBool(v)
			}
		case m.Def.Casts[col] == "integer":
			if v != nil {
				v = php.ToInt(v)
			}
		}
		out.Set(col, v)
	}
	for _, r := range m.relations {
		switch t := r.val.(type) {
		case *Model:
			if t == nil {
				out.Set(r.name, nil)
			} else {
				out.Set(r.name, t.ToArray())
			}
		case []*Model:
			l := php.NewArr()
			for _, x := range t {
				l.Append(x.ToArray())
			}
			out.Set(r.name, l)
		default:
			out.Set(r.name, nil)
		}
	}
	return out
}

func asDateTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case int64:
		return time.Unix(t, 0).UTC(), nil
	case float64:
		return time.Unix(int64(t), 0).UTC(), nil
	case string:
		if n, ok := php.NumericString(t); ok {
			if i, ok := n.(int64); ok {
				return time.Unix(i, 0).UTC(), nil
			}
		}
		if tm, err := time.Parse("2006-1-2", t); err == nil && len(t) <= 10 {
			return tm, nil
		}
		tm, err := time.Parse("2006-01-02 15:04:05", t)
		if err != nil {
			return time.Time{}, &php.Error{Msg: "Unexpected data found."}
		}
		return tm, nil
	}
	return time.Time{}, &php.Error{Msg: "bad date"}
}

func fromDateTime(v any) (string, error) {
	t, err := asDateTime(v)
	if err != nil {
		return "", err
	}
	return t.Format("2006-01-02 15:04:05"), nil
}

// ---- queries ----

func scanModels(def *Def, rows *sql.Rows) []*Model {
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		panic(err)
	}
	var out []*Model
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			panic(err)
		}
		m := New(def)
		for i, c := range cols {
			v := vals[i]
			switch t := v.(type) {
			case []byte:
				v = string(t)
			case int:
				v = int64(t)
			case time.Time:
				v = t.UTC().Format("2006-01-02 15:04:05")
			}
			m.SetRaw(c, v)
		}
		m.Exists = true
		m.syncOriginal()
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		panic(err)
	}
	return out
}

// All runs `select * from table <rest>`.
func (c *Ctx) All(def *Def, rest string, args ...any) []*Model {
	return scanModels(def, c.query(`select * from "`+def.Table+`" `+rest, args...))
}

// First returns the first row of `select * from table <rest>` or nil.
func (c *Ctx) First(def *Def, rest string, args ...any) *Model {
	ms := c.All(def, rest, args...)
	if len(ms) == 0 {
		return nil
	}
	return ms[0]
}

// Find loads by primary key.
func (c *Ctx) Find(def *Def, id any) *Model {
	return c.First(def, `where "`+def.Key+`" = ? limit 1`, db.Bind(id))
}

// Count runs a count(*) query.
func (c *Ctx) Count(q string, args ...any) int64 {
	var n int64
	if err := c.tx.QueryRowContext(c.ctx, q, bindAll(args)...).Scan(&n); err != nil {
		panic(err)
	}
	return n
}

// Ints returns the first column of each row.
func (c *Ctx) Values(q string, args ...any) []any {
	rows := c.query(q, args...)
	defer rows.Close()
	var out []any
	for rows.Next() {
		var v any
		if err := rows.Scan(&v); err != nil {
			panic(err)
		}
		if b, ok := v.([]byte); ok {
			v = string(b)
		}
		out = append(out, v)
	}
	return out
}

func bindAll(args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		out[i] = db.Bind(a)
	}
	return out
}

func (c *Ctx) query(q string, args ...any) *sql.Rows {
	rows, err := c.tx.QueryContext(c.ctx, q, bindAll(args)...)
	if err != nil {
		panic(err)
	}
	return rows
}

func (c *Ctx) exec(q string, args ...any) sql.Result {
	res, err := c.tx.ExecContext(c.ctx, q, bindAll(args)...)
	if err != nil {
		panic(err)
	}
	return res
}

// placeholders returns "?, ?, ?".
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}
