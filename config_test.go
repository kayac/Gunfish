package gunfish

import (
	"testing"
)

func TestLoadTomlConfigFile(t *testing.T) {
	_, err := LoadConfig("./test/gunfish_test.toml")

	if err != nil {
		t.Error(err)
	}
}
