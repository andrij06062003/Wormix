package proto

import (
	"encoding/json"

	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

// The arsenal: the layouts of the weapons panel the client saves and its hotkeys, kept by the API and given
// back in EnterAccount. 1.61.7 keeps several layouts and which one is in use; 1.25.0 keeps one.
const (
	CmdSetBackpackConf125       = 103
	CmdSetHotkeys125            = 104
	CmdSetHotkeys161            = 128
	CmdSetBackpackConf161       = 129
	CmdSetBackpackConfResult125 = 10103
	CmdSetBackpackConfResult161 = 10129
)

// Backpack is the arsenal of SetBackpackConf: its layouts and the one in use (1.61.7 counts it from 1).
type Backpack struct {
	Configs [][]int16
	Active  uint8
}

func readShorts(r *wire.Reader) []int16 {
	out := make([]int16, r.U16())
	for i := range out {
		out[i] = int16(r.U16())
	}
	return out
}

func writeShorts(w *wire.Writer, l []int16) {
	w.U16(uint16(len(l)))
	for _, n := range l {
		w.U16(uint16(n))
	}
}

// ReadSetBackpackConf161 reads SetBackpackConf (129): the layouts (each sized), then the one in use.
func ReadSetBackpackConf161(r *wire.Reader) Backpack {
	var b Backpack
	for n := r.U16(); n > 0; n-- {
		inner := wire.NewReader(r.BytesLenient(int(r.U16())))
		b.Configs = append(b.Configs, readShorts(inner))
	}
	b.Active = r.U8()
	return b
}

// ReadSetBackpackConf125 reads SetBackpackConf (103): the one layout.
func ReadSetBackpackConf125(r *wire.Reader) Backpack {
	return Backpack{Configs: [][]int16{readShorts(r)}}
}

// ReadSetHotkeys reads SetHotkeys (128, 1.25.0: 104).
func ReadSetHotkeys(r *wire.Reader) []int16 { return readShorts(r) }

func shortsJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// JSON is the API request body (account/backpack).
func (b Backpack) JSON() []byte {
	configs := b.Configs
	if configs == nil {
		configs = [][]int16{}
	}
	return jsonx.NewObject().Str("Configs", shortsJSON(configs)).Int("Active", int64(b.Active)).Bytes()
}

// HotkeysJSON is the API request body of account/hotkeys.
func HotkeysJSON(keys []int16) []byte {
	if keys == nil {
		keys = []int16{}
	}
	return jsonx.NewObject().Str("Hotkeys", shortsJSON(keys)).Bytes()
}

// writeBackpacks161 writes the layouts as EnterAccount and SetBackpackConfResult have them: each one sized.
func writeBackpacks161(w *wire.Writer, b Backpack) {
	w.U16(uint16(len(b.Configs)))
	for _, c := range b.Configs {
		var inner wire.Writer
		writeShorts(&inner, c)
		w.U16(uint16(len(inner.B)))
		w.Bytes(inner.B)
	}
	w.U8(b.Active)
}

// EmitSetBackpackConfResult161 answers SetBackpackConf (10129): the result (0 saved), then the layouts.
func EmitSetBackpackConfResult161(out Out, result int16, b Backpack) {
	var w wire.Writer
	w.U16(uint16(result))
	writeBackpacks161(&w, b)
	out(append(Header(CmdSetBackpackConfResult161, uint32(len(w.B))), w.B...))
}

// EmitSetBackpackConfResult125 answers SetBackpackConf (10103): the result, then the layout.
func EmitSetBackpackConfResult125(out Out, result int16, b Backpack) {
	var w wire.Writer
	w.U16(uint16(result))
	writeShorts(&w, b.first())
	out(append(Header(CmdSetBackpackConfResult125, uint32(len(w.B))), w.B...))
}

// first is the 1.25.0 client's one layout: the first one kept.
func (b Backpack) first() []int16 {
	if len(b.Configs) == 0 {
		return nil
	}
	return b.Configs[0]
}

// decodeShortLists reads a JSON string of lists of shorts the API keeps (nil for anything else).
func decodeShortLists(x *jsonx.Value) [][]int16 {
	var out [][]int16
	if x != nil && x.Kind == jsonx.String && json.Unmarshal([]byte(x.S), &out) == nil {
		return out
	}
	return nil
}

func decodeShortList(x *jsonx.Value) []int16 {
	var out []int16
	if x != nil && x.Kind == jsonx.String && json.Unmarshal([]byte(x.S), &out) == nil {
		return out
	}
	return nil
}
