package proto

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
	"github.com/andrij06062003/Wormix/services/core/internal/wire"
)

func TestSetBackpackConf161(t *testing.T) {
	want := Backpack{Configs: [][]int16{{1, 2, -3}, {}}, Active: 2}
	var w wire.Writer
	writeBackpacks161(&w, want)
	if got := ReadSetBackpackConf161(wire.NewReader(w.B)); !reflect.DeepEqual(got.Configs, [][]int16{{1, 2, -3}, {}}) || got.Active != 2 {
		t.Fatalf("read %+v", got)
	}

	var msg []byte
	EmitSetBackpackConfResult161(func(b []byte) { msg = append(msg, b...) }, 0, want)
	cmd, n, err := wire.ReadHeader(bytes.NewReader(msg))
	if err != nil || cmd != CmdSetBackpackConfResult161 || int(n) != len(msg)-wire.HeaderSize {
		t.Fatalf("header %d %d %v", cmd, n, err)
	}
	r := wire.NewReader(msg[wire.HeaderSize:])
	if r.U16() != 0 {
		t.Fatal("result")
	}
	if got := ReadSetBackpackConf161(r); got.Active != 2 || len(got.Configs) != 2 || r.Remaining() != 0 {
		t.Fatalf("answer %+v", got)
	}
}

func TestBackpackJSON(t *testing.T) {
	v, err := jsonx.Parse(Backpack{Configs: [][]int16{{5, -1}}, Active: 1}.JSON())
	if err != nil {
		t.Fatal(err)
	}
	if got := decodeShortLists(v.Obj.Get("Configs")); !reflect.DeepEqual(got, [][]int16{{5, -1}}) {
		t.Fatalf("configs %v", got)
	}
	if string(HotkeysJSON(nil)) != `{"Hotkeys":"[]"}` {
		t.Fatalf("hotkeys %s", HotkeysJSON(nil))
	}
}
