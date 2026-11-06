package app

import (
	"os"
	"strings"

	"github.com/andrij06062003/Wormix/services/api/internal/php"
)

// Config is the env-driven part of config/wormix.php (dev-environment starters).
type Config struct {
	IsDev            bool  // APP_ENV dev|development|local, CD_ENV=dev or WORMIX_DEV true/1
	StarterMoney     int64 // WORMIX_STARTER_MONEY (450)
	StarterRealMoney int64 // WORMIX_STARTER_REAL_MONEY (3, dev 3000)
	StarterLevel     int64 // WORMIX_STARTER_LEVEL (1, dev 30)
	DevLevel         int64 // WORMIX_DEV_STARTER_LEVEL (30)
	DevRealMoney     int64 // WORMIX_DEV_STARTER_REAL_MONEY (3000)
}

// laravelEnv is Laravel's env(): "true"/"(true)" -> true, "false" -> false, "null" -> nil, "empty" -> "".
func laravelEnv(key string, def any) any {
	v, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	switch strings.ToLower(v) {
	case "true", "(true)":
		return true
	case "false", "(false)":
		return false
	case "empty", "(empty)":
		return ""
	case "null", "(null)":
		return nil
	}
	if len(v) > 1 && (v[0] == '"' && v[len(v)-1] == '"' || v[0] == '\'' && v[len(v)-1] == '\'') {
		return v[1 : len(v)-1]
	}
	return v
}

// LoadConfig reads the environment like config/wormix.php and UserObserver::isDevEnvironment.
func LoadConfig() Config {
	env := func(k string) any { return laravelEnv(k, nil) }
	appEnv := laravelEnv("APP_ENV", "production")
	wd := env("WORMIX_DEV")
	isDev := php.StrictEq(appEnv, "dev") || php.StrictEq(appEnv, "development") || php.StrictEq(appEnv, "local") ||
		php.StrictEq(env("CD_ENV"), "dev") || php.StrictEq(wd, true) || php.StrictEq(wd, "true") || php.StrictEq(wd, "1")
	pick := func(dev, prod int64) int64 {
		if isDev {
			return dev
		}
		return prod
	}
	num := func(k string, d int64) int64 { return php.ToInt(laravelEnv(k, d)) }
	return Config{
		IsDev:            isDev,
		StarterMoney:     num("WORMIX_STARTER_MONEY", 450),
		StarterRealMoney: num("WORMIX_STARTER_REAL_MONEY", pick(3000, 3)),
		StarterLevel:     num("WORMIX_STARTER_LEVEL", pick(30, 1)),
		DevLevel:         num("WORMIX_DEV_STARTER_LEVEL", 30),
		DevRealMoney:     num("WORMIX_DEV_STARTER_REAL_MONEY", 3000),
	}
}

// starter returns the level and rubies a new or wiped account starts with.
func (cfg Config) starter() (level, realMoney int64) {
	if cfg.IsDev {
		return cfg.DevLevel, cfg.DevRealMoney
	}
	return cfg.StarterLevel, cfg.StarterRealMoney
}
