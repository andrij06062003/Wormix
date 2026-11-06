package app

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	for _, k := range []string{"APP_ENV", "CD_ENV", "WORMIX_DEV", "WORMIX_STARTER_MONEY", "WORMIX_STARTER_REAL_MONEY", "WORMIX_STARTER_LEVEL", "WORMIX_DEV_STARTER_LEVEL", "WORMIX_DEV_STARTER_REAL_MONEY"} {
		t.Setenv(k, "")
	}
	cases := []struct {
		env         map[string]string
		dev         bool
		lvl, rubies int64
	}{
		{map[string]string{"APP_ENV": "production"}, false, 1, 3},
		{map[string]string{"APP_ENV": "dev"}, true, 30, 3000},
		{map[string]string{"APP_ENV": "local"}, true, 30, 3000},
		{map[string]string{"APP_ENV": "production", "CD_ENV": "dev"}, true, 30, 3000},
		{map[string]string{"APP_ENV": "production", "WORMIX_DEV": "TRUE"}, true, 30, 3000},
		{map[string]string{"APP_ENV": "production", "WORMIX_DEV": "yes"}, false, 1, 3},
		{map[string]string{"APP_ENV": "dev", "WORMIX_DEV_STARTER_LEVEL": "12", "WORMIX_DEV_STARTER_REAL_MONEY": "50"}, true, 12, 50},
		{map[string]string{"APP_ENV": "production", "WORMIX_STARTER_LEVEL": "5", "WORMIX_STARTER_REAL_MONEY": "7"}, false, 5, 7},
	}
	for i, c := range cases {
		for _, k := range []string{"APP_ENV", "CD_ENV", "WORMIX_DEV", "WORMIX_STARTER_LEVEL", "WORMIX_STARTER_REAL_MONEY", "WORMIX_DEV_STARTER_LEVEL", "WORMIX_DEV_STARTER_REAL_MONEY"} {
			t.Setenv(k, c.env[k])
		}
		unsetEmpty(t)
		cfg := LoadConfig()
		lvl, rubies := cfg.starter()
		if cfg.IsDev != c.dev || lvl != c.lvl || rubies != c.rubies || cfg.StarterMoney != 450 {
			t.Errorf("case %d: %+v", i, cfg)
		}
	}
}

// unsetEmpty removes variables set to "" (t.Setenv can't unset; env() treats set-but-empty differently).
func unsetEmpty(t *testing.T) {
	for _, k := range []string{"APP_ENV", "CD_ENV", "WORMIX_DEV", "WORMIX_STARTER_MONEY", "WORMIX_STARTER_REAL_MONEY", "WORMIX_STARTER_LEVEL", "WORMIX_DEV_STARTER_LEVEL", "WORMIX_DEV_STARTER_REAL_MONEY"} {
		if v, ok := os.LookupEnv(k); ok && v == "" {
			os.Unsetenv(k)
		}
	}
}
