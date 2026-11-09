package app

import (
	"strconv"
	"strings"
	"time"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// The players' chat: one room for everybody and private messages between two players. The game's page
// (wwwroot/play.html) shows it over the game; the game opens it from the main menu's "bird with a letter"
// button. The page polls GET api/chat?after=<last id> every few seconds.
const (
	chatMaxLen       = 300 // letters in a message
	chatRoomHistory  = 50  // room messages a newly opened chat gets
	chatPrivHistory  = 200 // private messages a newly opened chat gets
	chatPollLimit    = 200 // messages in one poll
	chatBurst        = 5   // messages a player may send in chatBurstWindow
	chatBurstWindow  = 10 * time.Second
	chatTimeLayout   = time.DateTime
	chatMsgColumns   = `"id", "from_user_id", "to_user_id", "text", "created_at"`
	chatVisibleWhere = `("to_user_id" is null or "to_user_id" = ? or "from_user_id" = ?)`
)

type chatMsg struct {
	id, from, to int64 // to: 0 for the room
	text, at     string
}

func (c *Ctx) chatMessages(q string, args ...any) []chatMsg {
	rows := c.query(q, args...)
	defer rows.Close()
	var out []chatMsg
	for rows.Next() {
		var m chatMsg
		var to any
		if err := rows.Scan(&m.id, &m.from, &to, &m.text, &m.at); err != nil {
			panic(err)
		}
		if to != nil {
			m.to = php.ToInt(to)
		}
		out = append(out, m)
	}
	return out
}

// chatList is GET api/chat: the messages the player may see (the room and their own private ones) after the
// id `after`, or the latest ones without it, and the names and photos of everybody who wrote them.
func (c *Ctx) chatList() Resp {
	me := php.ToInt(c.user.Get("id"))
	after, _ := strconv.ParseInt(c.r.URL.Query().Get("after"), 10, 64)
	var msgs []chatMsg
	if after > 0 {
		msgs = c.chatMessages(`select `+chatMsgColumns+` from "chat_messages" where "id" > ? and `+chatVisibleWhere+` order by "id" limit ?`,
			after, me, me, chatPollLimit)
	} else {
		room := c.chatMessages(`select `+chatMsgColumns+` from "chat_messages" where "to_user_id" is null order by "id" desc limit ?`, chatRoomHistory)
		priv := c.chatMessages(`select `+chatMsgColumns+` from "chat_messages" where "to_user_id" is not null and ("to_user_id" = ? or "from_user_id" = ?) order by "id" desc limit ?`,
			me, me, chatPrivHistory)
		msgs = mergeByID(room, priv)
	}
	list := php.NewArr()
	users := map[int64]bool{}
	for _, m := range msgs {
		users[m.from] = true
		if m.to != 0 {
			users[m.to] = true
		}
		var to any
		if m.to != 0 {
			to = m.to
		}
		list.Append(php.Assoc("id", m.id, "from", m.from, "to", to, "text", m.text, "at", chatTime(m.at)))
	}
	return jsonResp(200, php.Assoc("me", me, "messages", list, "users", c.chatUsers(users)))
}

// mergeByID merges two lists sorted by id descending into one sorted ascending.
func mergeByID(a, b []chatMsg) []chatMsg {
	out := make([]chatMsg, 0, len(a)+len(b))
	i, j := len(a)-1, len(b)-1
	for i >= 0 || j >= 0 {
		if j < 0 || (i >= 0 && a[i].id < b[j].id) {
			out = append(out, a[i])
			i--
		} else {
			out = append(out, b[j])
			j--
		}
	}
	return out
}

func chatTime(at string) string {
	t, err := time.Parse(chatTimeLayout, at)
	if err != nil {
		return at
	}
	return t.UTC().Format(time.RFC3339)
}

// chatUsers: id -> {name, photo} of the given players (the name the game shows, else the login).
func (c *Ctx) chatUsers(ids map[int64]bool) *php.Arr {
	out := php.NewArr()
	if len(ids) == 0 {
		return out
	}
	args := make([]any, 0, len(ids))
	marks := make([]string, 0, len(ids))
	for id := range ids {
		args = append(args, id)
		marks = append(marks, "?")
	}
	rows := c.query(`select "users"."id", "users"."login", "users_vk_data"."first_name", "users_vk_data"."last_name", "users_vk_data"."photo"
		from "users" left join "users_vk_data" on "users_vk_data"."user_id" = "users"."id" where "users"."id" in (`+strings.Join(marks, ", ")+`)`, args...)
	defer rows.Close()
	for rows.Next() {
		var id int64
		var login, first, last, photo any
		if err := rows.Scan(&id, &login, &first, &last, &photo); err != nil {
			panic(err)
		}
		name := strings.TrimSpace(php.ToString(first) + " " + php.ToString(last))
		if name == "" {
			name = php.ToString(login)
		}
		var p any
		if photo != nil {
			p = "/api/photos/users/" + php.ToString(photo)
		}
		out.Set(strconv.FormatInt(id, 10), php.Assoc("name", name, "photo", p))
	}
	return out
}

// chatSend is POST api/chat {text, to}: a message to the room, or to one player with `to`.
func (c *Ctx) chatSend() Resp {
	c.Validate([]R{
		{"text", "required|string|max:" + strconv.Itoa(chatMaxLen)},
		{"to", "nullable|integer|exists:users,id"},
	})
	me := php.ToInt(c.user.Get("id"))
	text := strings.Join(strings.Fields(php.ToString(c.JSON("text"))), " ")
	if text == "" {
		return jsonResp(422, php.Assoc("message", "The message is empty", "error", "empty"))
	}
	var to any
	if t := c.JSON("to"); t != nil {
		if php.ToInt(t) == me {
			return jsonResp(422, php.Assoc("message", "That's you", "error", "self"))
		}
		to = php.ToInt(t)
	}
	now := time.Now().UTC()
	if c.Count(`select count(*) from "chat_messages" where "from_user_id" = ? and "created_at" > ?`,
		me, now.Add(-chatBurstWindow).Format(chatTimeLayout)) >= chatBurst {
		return jsonResp(429, php.Assoc("message", "Too many messages, wait a little", "error", "too_fast"))
	}
	res := c.exec(`insert into "chat_messages" ("from_user_id", "to_user_id", "text", "created_at") values (?, ?, ?, ?)`,
		me, to, text, now.Format(chatTimeLayout))
	id, _ := res.LastInsertId()
	return jsonResp(200, php.Assoc("id", id, "from", me, "to", to, "text", text, "at", now.Format(time.RFC3339)))
}
