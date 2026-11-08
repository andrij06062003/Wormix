package main

// Achievements server for the game's "wormix_achieve" connection.
//
// Unlike the main and PvP connections, achievements use HTTP: the game POSTs binary commands to
// http://<host>:49199/<prefix>/ (play.html rewrites that URL to this host) with the session id from
// the login reply in the X-PRAGMATIX-CORP-SID header, and reads binary commands back from the reply.
// This decodes them and calls wormix-api's /api/internal/achievements/ endpoints with JSON.
//
// Command format: 7-byte header (flags, command id, length; see wormix-core BinaryCommandHeader),
// then the body. Most bodies end with MD5(body + SecretKey) so the game can spot tampering.

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
)

const (
	// SecuritySettings.SECRET_KEY: the 1.05.0 client's salt, then the 1.25.0 one's.
	achSecretKey     = "ygfA69i3Uz6bDO8"
	achSecretKey125  = "Uzg4IbZnpskiY73hqnjkPsnRY03H"
	achSessionHeader = "X-PRAGMATIX-CORP-SID"
)

// The prefix comes from the preloader ("wormix_achieve") or the client default ("wormix_achievements").
var achievementPrefixes = []string{"/wormix_achieve", "/wormix_achievements"}

// Command ids.
const (
	// Client -> server
	cmdPing                 = 16
	cmdAchieveLogin         = 3001
	cmdGetAchievements      = 3002
	cmdIncreaseAchievements = 3003
	cmdChooseBonusItem      = 3005
	cmdBuyResetBonusItems   = 3010 // 1.25.0
	// Server -> client
	cmdPong                      = 10017
	cmdAchieveLoginSuccess       = 13001
	cmdAchieveLoginError         = 13002
	cmdGetAchievementsResult     = 13003
	cmdIncreaseAchievementResult = 13004
	cmdChooseBonusItemResult     = 13006
	cmdBuyResetBonusItemsResult  = 13010
)

// Error kinds, each answered differently (see serveAchievements).
var (
	errUnauthorized = errors.New("unauthorized") // API said 401: the game logs in again
	errBadData      = errors.New("bad request")  // undecodable request
	errAPIRequest   = errors.New("API request")  // API unreachable or answered an error
)

func (s *server) serveAchievements(w http.ResponseWriter, r *http.Request) {
	input, ok, err := readBody(w, r)
	if !ok {
		if err != nil {
			s.log.Warn("Can't read achievements request: " + err.Error())
		}
		return
	}
	var session *string
	if vs := r.Header.Values(achSessionHeader); len(vs) > 0 {
		joined := strings.Join(vs, ",")
		session = &joined
	}

	var output bytes.Buffer
	err = func() error {
		pos := 0
		for len(input)-pos >= 7 {
			id, length, err := readHeader(input[pos:])
			if err != nil {
				return err
			}
			pos += 7
			if len(input)-pos < length {
				return fmt.Errorf("%w: command is cut short", errBadData)
			}
			body := input[pos : pos+length]
			pos += length
			if err := s.handleAchievement(r.Context(), id, body, session, &output); err != nil {
				return err
			}
		}
		return nil
	}()
	switch {
	case err == nil:
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(output.Len()))
		w.WriteHeader(http.StatusOK)
		w.Write(output.Bytes())
	case errors.Is(err, errUnauthorized):
		// The game looks for "401" in a non-binary reply, then logs in again.
		writeText(w, "401 Unauthorized")
	case errors.Is(err, errBadData):
		s.log.Warn("Bad achievements request: " + err.Error())
		writeText(w, "400 Bad Request")
	case errors.Is(err, errAPIRequest):
		s.log.Warn("Can't reach API for achievements: " + err.Error())
		w.WriteHeader(http.StatusBadGateway)
	default:
		// An unhandled exception in the C# version.
		s.log.Error("Achievements request failed: " + err.Error())
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func writeText(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(text)))
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, text)
}

func (s *server) handleAchievement(ctx context.Context, id int, body []byte, session *string, output *bytes.Buffer) error {
	switch id {
	case cmdPing:
		writeCommand(output, cmdPong, nil, "")

	case cmdAchieveLogin:
		rd := reader{data: body}
		rd.utf() // application id
		rd.utf() // social network
		userID := rd.utf()
		authKey := rd.utf()
		if rd.err != nil {
			return rd.err
		}
		// The 1.25.0 client adds sendAchievements (bool) and signs the replies with its own salt.
		key := achSecretKey
		if len(rd.data) > 0 {
			key = achSecretKey125
		}
		res, err := s.achCall(ctx, "login", struct {
			ID      string `json:"id"`
			AuthKey string `json:"auth_key"`
		}{userID, authKey})
		if err != nil {
			return err
		}
		var wr writer
		if res.boolean("ok") {
			wr.utf(res.str("session_id"))
			wr.u32(res.uint32("login_time"))
			if res.err == nil {
				writeCommand(output, cmdAchieveLoginSuccess, wr.Bytes(), key)
			}
		} else {
			wr.i32(res.int32("code"))
			if res.err == nil {
				writeCommand(output, cmdAchieveLoginError, wr.Bytes(), "")
			}
		}
		return res.err

	case cmdGetAchievements:
		key, err := checkSecure(body)
		if err != nil {
			return err
		}
		rd := reader{data: body[:len(body)-16]}
		profileID := rd.utf() // 1.25.0: then investedAwardPoints (byte)
		if rd.err != nil {
			return rd.err
		}
		res, err := s.achCall(ctx, "get", struct {
			SessionID *string `json:"session_id"`
			ProfileID string  `json:"profile_id"`
		}{session, profileID})
		if err != nil {
			return err
		}
		var wr writer
		wr.utf(res.str("profile_id"))
		wr.byteArray(res.ints("indexes"))
		wr.u16Array(res.ints("values"))
		if key == achSecretKey125 {
			wr.byteArray(nil) // boolAchievements
		}
		wr.u8(res.int32("invested"))
		if res.err == nil {
			writeCommand(output, cmdGetAchievementsResult, wr.Bytes(), key)
		}
		return res.err

	case cmdIncreaseAchievements:
		key, err := checkSecure(body)
		if err != nil {
			return err
		}
		rd := reader{data: body[:len(body)-16]}
		bodySession := rd.utf()
		indexes := make([]int, rd.u16())
		for i := range indexes {
			indexes[i] = rd.u8()
		}
		rises := make([]int, rd.u16())
		for i := range rises {
			rises[i] = rd.i16()
		}
		if key == achSecretKey125 {
			// 1.25.0: boolAchievements (ints), not kept
			for range rd.u16() {
				rd.i32()
			}
		}
		timeScale := rd.u32()
		if rd.err != nil {
			return rd.err
		}
		sid := session
		if bodySession != "" {
			sid = &bodySession
		}
		res, err := s.achCall(ctx, "increase", struct {
			SessionID *string `json:"session_id"`
			Indexes   []int   `json:"indexes"`
			Rises     []int   `json:"rises"`
			TimeScale uint32  `json:"time_scale"`
		}{sid, indexes, rises, timeScale})
		if err != nil {
			return err
		}
		var wr writer
		wr.byteArray(res.ints("indexes"))
		wr.u16Array(res.ints("values"))
		wr.byteArray(res.ints("threshold_indexes"))
		wr.u16Array(res.ints("threshold_old_values"))
		if key == achSecretKey125 {
			wr.u16(0) // boolAchievements (ints)
		}
		wr.u32(res.uint32("time_sequence"))
		wr.u8(res.int32("invested"))
		if res.err == nil {
			writeCommand(output, cmdIncreaseAchievementResult, wr.Bytes(), key)
		}
		return res.err

	case cmdChooseBonusItem:
		key, err := checkSecure(body)
		if err != nil {
			return err
		}
		rd := reader{data: body[:len(body)-16]}
		itemID := rd.i32()
		if rd.err != nil {
			return rd.err
		}
		res, err := s.achCall(ctx, "choose_bonus", struct {
			SessionID *string `json:"session_id"`
			ItemID    int     `json:"item_id"`
		}{session, itemID})
		if err != nil {
			return err
		}
		var wr writer
		wr.i16(res.int32("result"))
		wr.i32(res.int32("item_id"))
		if res.err == nil {
			writeCommand(output, cmdChooseBonusItemResult, wr.Bytes(), key)
		}
		return res.err

	case cmdBuyResetBonusItems:
		// 1.25.0: pay to choose the bonus items again. Not on the server: refused (result 1).
		key, err := checkSecure(body)
		if err != nil {
			return err
		}
		rd := reader{data: body[:len(body)-16]}
		requestNum := rd.i32()
		if rd.err != nil {
			return rd.err
		}
		var wr writer
		wr.i16(1)
		wr.i32(requestNum)
		writeCommand(output, cmdBuyResetBonusItemsResult, wr.Bytes(), key)

	default:
		return fmt.Errorf("%w: unknown achievements command %d", errBadData, id)
	}
	return nil
}

// achCall POSTs payload to /api/internal/achievements/{action} and returns the JSON reply.
func (s *server) achCall(ctx context.Context, action string, payload any) (*jsonObj, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.apiBase+"/api/internal/achievements/"+action, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-TCP-SESSION", "achievements")
	req.Header["User-Agent"] = []string{""}
	resp, err := s.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, err // timeout or cancelled: not an HttpRequestException in .NET
		}
		return nil, fmt.Errorf("%w: %v", errAPIRequest, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errUnauthorized
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%w: response status code does not indicate success: %d", errAPIRequest, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("API reply is not JSON: %w", err)
	}
	if dec.More() {
		return nil, errors.New("API reply has data after the JSON value")
	}
	return &jsonObj{v: v}, nil
}

// jsonObj reads properties like JsonElement.GetProperty(...).GetXxx(): the first failure is kept
// in err (an unhandled exception, so a 500, in the C# version) and later reads return zero values.
type jsonObj struct {
	v   any
	err error
}

func (o *jsonObj) prop(name string) any {
	if o.err != nil {
		return nil
	}
	m, ok := o.v.(map[string]any)
	if !ok {
		o.err = fmt.Errorf("API reply is not a JSON object")
		return nil
	}
	v, ok := m[name]
	if !ok {
		o.err = fmt.Errorf("API reply has no %q", name)
		return nil
	}
	return v
}

func (o *jsonObj) fail(name, want string) { o.err = fmt.Errorf("API reply: %q is not %s", name, want) }

func (o *jsonObj) boolean(name string) bool {
	v := o.prop(name)
	b, ok := v.(bool)
	if !ok && o.err == nil {
		o.fail(name, "a boolean")
	}
	return b
}

func (o *jsonObj) str(name string) string {
	v := o.prop(name)
	s, ok := v.(string)
	if !ok && o.err == nil {
		o.fail(name, "a string")
	}
	return s
}

func numberAs(v any, bits int, unsigned bool) (int64, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, false
	}
	if unsigned {
		u, err := strconv.ParseUint(n.String(), 10, bits)
		return int64(u), err == nil
	}
	i, err := strconv.ParseInt(n.String(), 10, bits)
	return i, err == nil
}

func (o *jsonObj) int32(name string) int {
	v := o.prop(name)
	n, ok := numberAs(v, 32, false)
	if !ok && o.err == nil {
		o.fail(name, "an Int32")
	}
	return int(n)
}

func (o *jsonObj) uint32(name string) uint32 {
	v := o.prop(name)
	n, ok := numberAs(v, 32, true)
	if !ok && o.err == nil {
		o.fail(name, "a UInt32")
	}
	return uint32(n)
}

// ints reads an array of Int32.
func (o *jsonObj) ints(name string) []int {
	v := o.prop(name)
	arr, ok := v.([]any)
	if !ok {
		if o.err == nil {
			o.fail(name, "an array")
		}
		return nil
	}
	out := make([]int, len(arr))
	for i, e := range arr {
		n, ok := numberAs(e, 32, false)
		if !ok {
			o.fail(name, "an array of Int32")
			return nil
		}
		out[i] = int(n)
	}
	return out
}

// --- binary format ---

// readHeader decodes a 7-byte command header and checks its sync bits.
func readHeader(data []byte) (id, length int, err error) {
	flags := uint32(data[0])
	rawID := uint32(binary.BigEndian.Uint16(data[1:]))
	rawLength := binary.BigEndian.Uint32(data[3:])
	valid := flags&87 == 87 &&
		(flags&128 != 0) != (rawID&0x8000 != 0) &&
		(flags&32 != 0) != (rawLength&0x40000000 != 0) &&
		(flags&8 != 0) != (rawLength&0x80000000 != 0)
	if !valid {
		return 0, 0, fmt.Errorf("%w: invalid command header", errBadData)
	}
	return int(rawID & 0x7FFF), int(rawLength & 0x3FFFFFFF), nil
}

// writeCommand writes a command; with a key (the client's salt) its body is signed.
func writeCommand(output *bytes.Buffer, id int, body []byte, key string) {
	if key != "" {
		h := achHash(body, key)
		body = append(body[:len(body):len(body)], h[:]...)
	}
	// Each sync bit is carried either in the flags byte or in the id/length field, picked at random.
	flags, rawID, rawLength := uint32(87), uint32(id), uint32(len(body))
	if rand.IntN(2) == 0 {
		flags |= 128
	} else {
		rawID |= 0x8000
	}
	if rand.IntN(2) == 0 {
		flags |= 32
	} else {
		rawLength |= 0x40000000
	}
	if rand.IntN(2) == 0 {
		flags |= 8
	} else {
		rawLength |= 0x80000000
	}
	var header [7]byte
	header[0] = byte(flags)
	binary.BigEndian.PutUint16(header[1:], uint16(rawID))
	binary.BigEndian.PutUint32(header[3:], rawLength)
	output.Write(header[:])
	output.Write(body)
}

func achHash(data []byte, key string) [16]byte {
	return md5.Sum(append(data[:len(data):len(data)], key...))
}

// checkSecure checks a signed command's hash and returns the salt it was signed with: the 1.05.0 or
// the 1.25.0 client's (its replies are signed with the same).
func checkSecure(body []byte) (string, error) {
	if len(body) < 16 {
		return "", fmt.Errorf("%w: bad command hash", errBadData)
	}
	for _, key := range []string{achSecretKey, achSecretKey125} {
		h := achHash(body[:len(body)-16], key)
		if bytes.Equal(h[:], body[len(body)-16:]) {
			return key, nil
		}
	}
	return "", fmt.Errorf("%w: bad command hash", errBadData)
}

// reader reads big-endian values; the first short read sets err and later reads return 0.
type reader struct {
	data []byte
	err  error
}

func (r *reader) take(n int) []byte {
	if r.err != nil {
		return nil
	}
	if len(r.data) < n {
		r.err = fmt.Errorf("%w: unable to read beyond the end of the stream", errBadData)
		r.data = nil
		return nil
	}
	b := r.data[:n]
	r.data = r.data[n:]
	return b
}

func (r *reader) u8() int {
	if b := r.take(1); b != nil {
		return int(b[0])
	}
	return 0
}

func (r *reader) u16() int {
	if b := r.take(2); b != nil {
		return int(binary.BigEndian.Uint16(b))
	}
	return 0
}

func (r *reader) i16() int {
	if b := r.take(2); b != nil {
		return int(int16(binary.BigEndian.Uint16(b)))
	}
	return 0
}

func (r *reader) i32() int {
	if b := r.take(4); b != nil {
		return int(int32(binary.BigEndian.Uint32(b)))
	}
	return 0
}

func (r *reader) u32() uint32 {
	if b := r.take(4); b != nil {
		return binary.BigEndian.Uint32(b)
	}
	return 0
}

// utf reads a 2-byte length then that many UTF-8 bytes (invalid bytes become U+FFFD, like .NET).
func (r *reader) utf() string {
	n := r.u16()
	return strings.ToValidUTF8(string(r.take(n)), "�")
}

type writer struct{ bytes.Buffer }

func (w *writer) u8(v int)     { w.WriteByte(byte(v)) }
func (w *writer) i16(v int)    { w.Write(binary.BigEndian.AppendUint16(nil, uint16(int16(v)))) }
func (w *writer) u16(v int)    { w.Write(binary.BigEndian.AppendUint16(nil, uint16(v))) }
func (w *writer) i32(v int)    { w.Write(binary.BigEndian.AppendUint32(nil, uint32(int32(v)))) }
func (w *writer) u32(v uint32) { w.Write(binary.BigEndian.AppendUint32(nil, v)) }

func (w *writer) utf(v string) {
	w.u16(len(v))
	w.WriteString(v)
}

func (w *writer) byteArray(values []int) {
	w.u16(len(values))
	for _, v := range values {
		w.u8(v)
	}
}

func (w *writer) u16Array(values []int) {
	w.u16(len(values))
	for _, v := range values {
		w.u16(v)
	}
}
