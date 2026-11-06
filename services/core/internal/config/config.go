// Package config reads the server configuration (the JSON format of docker/wormix-core.json).
package config

import (
	"encoding/json"
	"errors"
	"strings"
)

// Server is one entry of "servers".
type Server struct {
	Port    int  `json:"port"`
	Enabled bool `json:"enabled"`
}

// Config is the whole file. The "redis" section is accepted and ignored: sessions are kept in memory.
type Config struct {
	Local   bool              `json:"local"`
	Servers map[string]Server `json:"servers"`
	APIURL  string            `json:"ApiUrl"`
	// Order of the "servers" keys in the file.
	Order []string `json:"-"`
}

// Parse parses and checks a configuration. Keys match case-insensitively, as with Newtonsoft.Json.
func Parse(data []byte) (*Config, error) {
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	if strings.TrimSpace(c.APIURL) == "" {
		return nil, errors.New("ApiUrl is undefined")
	}
	// Remember the order of the servers in the file (start order of the C# server).
	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)
	for k, v := range raw {
		if strings.EqualFold(k, "servers") {
			dec := json.NewDecoder(strings.NewReader(string(v)))
			if t, err := dec.Token(); err == nil && t == json.Delim('{') {
				for dec.More() {
					t, err := dec.Token()
					if err != nil {
						break
					}
					if key, ok := t.(string); ok {
						c.Order = append(c.Order, key)
					}
					var skip json.RawMessage
					if dec.Decode(&skip) != nil {
						break
					}
				}
			}
		}
	}
	return &c, nil
}

// Address is the listen address: 127.0.0.1 with "local", else all interfaces.
func (c *Config) Address() string {
	if c.Local {
		return "127.0.0.1"
	}
	return "0.0.0.0"
}
