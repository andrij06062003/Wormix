package config

import (
	"os"
	"reflect"
	"testing"
)

func TestRepositoryConfig(t *testing.T) {
	data, err := os.ReadFile("../../../../docker/wormix-core.json")
	if err != nil {
		t.Skip("docker/wormix-core.json not found:", err)
	}
	c, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if c.APIURL != "http://api:8000/api/internal/" || c.Address() != "0.0.0.0" {
		t.Fatalf("%+v", c)
	}
	if !reflect.DeepEqual(c.Order, []string{"policy", "main", "pvp", "achievements", "clan"}) {
		t.Fatalf("order %v", c.Order)
	}
	want := map[string]Server{"policy": {843, false}, "main": {49197, true}, "pvp": {49198, true},
		"achievements": {49199, true}, "clan": {49299, false}}
	if !reflect.DeepEqual(c.Servers, want) {
		t.Fatalf("servers %v", c.Servers)
	}
}

func TestConfigErrors(t *testing.T) {
	if _, err := Parse([]byte(`{"servers":{}}`)); err == nil {
		t.Error("missing ApiUrl accepted")
	}
	if _, err := Parse([]byte(`{"apiurl":"  "}`)); err == nil {
		t.Error("blank ApiUrl accepted")
	}
	c, err := Parse([]byte(`{"LOCAL":true,"apiurl":"http://x/"}`)) // keys match case-insensitively; no redis needed
	if err != nil || c.Address() != "127.0.0.1" || c.APIURL != "http://x/" {
		t.Fatalf("%+v %v", c, err)
	}
}
