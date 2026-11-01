package gameplay_test

import (
	"testing"

	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// TestGlobalConfigCopy proves Get/SetGlobalConfig round-trip scalar values;
// rooms snapshot the config on start (see snapshot_test.go).
func TestGlobalConfigCopy(t *testing.T) {
	defer gameplay.SetGlobalConfig(config.DefaultConfig)

	before := config.DefaultConfig.ChunkSize
	cfg := gameplay.GetGlobalConfig()
	cfg.ChunkSize = before + 8
	gameplay.SetGlobalConfig(cfg)

	if got := gameplay.GetGlobalConfig().ChunkSize; got != before+8 {
		t.Fatalf("global config not applied: %d", got)
	}
}
