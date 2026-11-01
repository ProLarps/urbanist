package gameplay_test

import (
	"testing"

	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/errcodes"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// everyoneOnline is a startCheck that accepts any member id.
func everyoneOnline(string) bool { return true }

// TestRunningRoomKeepsItsSnapshot is the core admin-panel guarantee
// (docs/API.md §2.7): rooms already in run mode keep the config they froze
// at Start; only rooms started afterwards see new GlobalConfig values.
func TestRunningRoomKeepsItsSnapshot(t *testing.T) {
	gameplay.SetGlobalConfig(config.DefaultConfig)
	defer gameplay.SetGlobalConfig(config.DefaultConfig)

	m := gameplay.NewManager()
	owner := gameplay.Member{ID: "p_1", Name: "Аня"}
	room, code := m.CreateRoom(owner, "alpha", 2)
	if code != errcodes.OK {
		t.Fatalf("create: %v", code)
	}
	if code := room.Start(everyoneOnline); code != errcodes.OK {
		t.Fatalf("start: %v", code)
	}

	chunkBefore, ok := room.Chunk(0, 0)
	if !ok {
		t.Fatal("run-mode room must serve chunks")
	}

	cfg := gameplay.GetGlobalConfig()
	cfg.ChunkSize = 32 // admin change AFTER the game started
	gameplay.SetGlobalConfig(cfg)

	chunkAfter, _ := room.Chunk(0, 0)
	if chunkAfter.Size != chunkBefore.Size {
		t.Fatalf("running room was mutated by an admin edit: %d -> %d",
			chunkBefore.Size, chunkAfter.Size)
	}

	fresh, code := m.CreateRoom(gameplay.Member{ID: "p_2", Name: "Боб"}, "beta", 2)
	if code != errcodes.OK {
		t.Fatalf("create fresh: %v", code)
	}
	if code := fresh.Start(everyoneOnline); code != errcodes.OK {
		t.Fatalf("fresh start: %v", code)
	}
	newChunk, _ := fresh.Chunk(0, 0)
	if newChunk.Size != 32 {
		t.Fatalf("new room ignored admin config: %d", newChunk.Size)
	}
}
