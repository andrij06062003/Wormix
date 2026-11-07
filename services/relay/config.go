package main

// Configuration that behaves like ASP.NET Core's WebApplication.CreateBuilder defaults, so the same
// settings keep working: keys are case-insensitive and nested keys use ':' ("AllowedPorts:0").
// Sources, later ones win:
//
//	DOTNET_* and ASPNETCORE_* environment variables (prefix removed)
//	appsettings.json, then appsettings.{Environment}.json, in the working directory
//	environment variables ("__" in a name stands for ':', e.g. AllowedPorts__0=49197)
//	command line: --Key value, --Key=value, /Key value, /Key=value, Key=value

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Config is a flat, case-insensitive key/value view of all configuration sources.
type Config struct {
	values map[string]string // lower-cased key -> value
}

func newConfig() *Config { return &Config{values: map[string]string{}} }

func (c *Config) set(key, value string) { c.values[strings.ToLower(key)] = value }

// Get returns the value of key and whether it is set.
func (c *Config) Get(key string) (string, bool) {
	v, ok := c.values[strings.ToLower(key)]
	return v, ok
}

// String returns the value of key, or def when it isn't set.
func (c *Config) String(key, def string) string {
	if v, ok := c.Get(key); ok {
		return v
	}
	return def
}

// Bool parses key like .NET's bool.Parse (case-insensitive "true"/"false", surrounding spaces allowed).
func (c *Config) Bool(key string, def bool) (bool, error) {
	v, ok := c.Get(key)
	if !ok {
		return def, nil
	}
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	return false, fmt.Errorf("failed to convert configuration value %q at %q to bool", v, key)
}

// Ints binds section key as an int array, the way ConfigurationBinder binds int[]: the children
// "key:0", "key:1", ... in key order. ok is false when the section doesn't exist.
func (c *Config) Ints(key string) (vals []int, ok bool, err error) {
	prefix := strings.ToLower(key) + ":"
	children := map[string]bool{}
	for k := range c.values {
		if rest, found := strings.CutPrefix(k, prefix); found {
			child, _, _ := strings.Cut(rest, ":")
			children[child] = true
		}
	}
	if len(children) == 0 {
		// Not ASP.NET behavior but handy: a plain "49197,49198" list.
		v, set := c.Get(key)
		if !set || strings.TrimSpace(v) == "" {
			return nil, false, nil
		}
		for _, f := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			n, err := parseDotnetInt(f)
			if err != nil {
				return nil, true, fmt.Errorf("failed to convert configuration value %q at %q to int", f, key)
			}
			vals = append(vals, n)
		}
		return vals, true, nil
	}
	names := make([]string, 0, len(children))
	for k := range children {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool { return configKeyLess(names[i], names[j]) })
	for _, name := range names {
		v, set := c.values[prefix+name]
		if !set {
			continue // only has sub-keys
		}
		n, err := parseDotnetInt(v)
		if err != nil {
			return nil, true, fmt.Errorf("failed to convert configuration value %q at %q to int", v, key+":"+name)
		}
		vals = append(vals, n)
	}
	return vals, true, nil
}

// configKeyLess orders keys like ConfigurationKeyComparer: numbers numerically, before other keys.
func configKeyLess(a, b string) bool {
	ai, aerr := strconv.Atoi(a)
	bi, berr := strconv.Atoi(b)
	switch {
	case aerr == nil && berr == nil:
		return ai < bi
	case aerr == nil:
		return true
	case berr == nil:
		return false
	}
	return a < b
}

// parseDotnetInt parses an Int32 like int.Parse with NumberStyles.Integer.
func parseDotnetInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "+")
	n, err := strconv.ParseInt(s, 10, 32)
	return int(n), err
}

// LoadConfig reads all sources. dir is where appsettings*.json are looked for.
func LoadConfig(args, environ []string, dir string) (*Config, error) {
	c := newConfig()
	env := make([][2]string, 0, len(environ))
	for _, kv := range environ {
		k, v, ok := strings.Cut(kv, "=")
		if ok && k != "" {
			env = append(env, [2]string{strings.ReplaceAll(k, "__", ":"), v})
		}
	}
	for _, prefix := range []string{"DOTNET_", "ASPNETCORE_"} {
		for _, kv := range env {
			if len(kv[0]) > len(prefix) && strings.EqualFold(kv[0][:len(prefix)], prefix) {
				c.set(kv[0][len(prefix):], kv[1])
			}
		}
	}
	// The environment name decides which appsettings.{Environment}.json is read; it can only
	// come from the prefixed variables or the command line at this point, like in ASP.NET.
	pre := newConfig()
	for k, v := range c.values {
		pre.values[k] = v
	}
	if err := pre.addArgs(args); err != nil {
		return nil, err
	}
	envName := pre.String("environment", "Production")
	for _, name := range []string{"appsettings.json", "appsettings." + envName + ".json"} {
		if err := c.addJSONFile(filepath.Join(dir, name)); err != nil {
			return nil, err
		}
	}
	for _, kv := range env {
		c.set(kv[0], kv[1])
	}
	if err := c.addArgs(args); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) addJSONFile(path string) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // optional, like in ASP.NET
	}
	if err != nil {
		return err
	}
	var root any
	dec := json.NewDecoder(strings.NewReader(stripJSONComments(string(data))))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	obj, ok := root.(map[string]any)
	if !ok {
		return fmt.Errorf("%s: top-level JSON element must be an object", path)
	}
	c.flatten("", obj)
	return nil
}

func (c *Config) flatten(prefix string, v any) {
	join := func(k string) string {
		if prefix == "" {
			return k
		}
		return prefix + ":" + k
	}
	switch t := v.(type) {
	case map[string]any:
		if len(t) == 0 && prefix != "" {
			c.set(prefix, "")
		}
		for k, child := range t {
			c.flatten(join(k), child)
		}
	case []any:
		if len(t) == 0 && prefix != "" {
			c.set(prefix, "")
		}
		for i, child := range t {
			c.flatten(join(strconv.Itoa(i)), child)
		}
	case string:
		c.set(prefix, t)
	case json.Number:
		c.set(prefix, t.String())
	case bool:
		if t {
			c.set(prefix, "True")
		} else {
			c.set(prefix, "False")
		}
	case nil:
		c.set(prefix, "")
	}
}

// stripJSONComments removes // and /* */ comments (allowed in appsettings.json) outside strings,
// and trailing commas before } or ].
func stripJSONComments(s string) string {
	var b strings.Builder
	inStr, esc := false, false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inStr {
			b.WriteByte(ch)
			switch {
			case esc:
				esc = false
			case ch == '\\':
				esc = true
			case ch == '"':
				inStr = false
			}
			continue
		}
		switch {
		case ch == '"':
			inStr = true
			b.WriteByte(ch)
		case ch == '/' && i+1 < len(s) && s[i+1] == '/':
			for i < len(s) && s[i] != '\n' {
				i++
			}
			b.WriteByte('\n')
		case ch == '/' && i+1 < len(s) && s[i+1] == '*':
			end := strings.Index(s[i+2:], "*/")
			if end < 0 {
				i = len(s)
			} else {
				i += end + 3
			}
			b.WriteByte(' ')
		case ch == ',':
			j := i + 1
			for j < len(s) && strings.IndexByte(" \t\r\n", s[j]) >= 0 {
				j++
			}
			if j < len(s) && (s[j] == '}' || s[j] == ']') {
				continue
			}
			b.WriteByte(ch)
		default:
			b.WriteByte(ch)
		}
	}
	return b.String()
}

// addArgs follows CommandLineConfigurationProvider (without switch mappings).
func (c *Config) addArgs(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		start := 0
		switch {
		case strings.HasPrefix(arg, "--"):
			start = 2
		case strings.HasPrefix(arg, "-"):
			continue // single-dash keys need switch mappings in ASP.NET; ignored there too
		case strings.HasPrefix(arg, "/"):
			start = 1
		}
		key, value, hasEq := strings.Cut(arg[start:], "=")
		if !hasEq {
			if start == 0 {
				continue // a bare word is not a setting
			}
			if i+1 >= len(args) {
				continue
			}
			i++
			value = args[i]
		}
		if key == "" {
			return fmt.Errorf("invalid command line argument %q", arg)
		}
		c.set(key, value)
	}
	return nil
}
