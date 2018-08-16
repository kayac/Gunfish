package config

import (
	"os"
	"testing"
)

func TestLoadTomlConfigFile(t *testing.T) {
	if err := os.Setenv("TEST_GUNFISH_HOOK_CMD", "cat | grep test"); err != nil {
		t.Error(err)
	}

	c, err := LoadConfig("../test/gunfish_test.toml")
	if err != nil {
		t.Error(err)
	}

	if g, w := c.Provider.ErrorHook, "cat | grep test"; g != w {
		t.Errorf("not match error hook: got %s want %s", g, w)
	}
}
