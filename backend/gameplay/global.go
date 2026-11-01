// Package gameplay owns rooms, worlds and districts — everything about a
// running game. Two access rules keep it clean:
//
//   - Rooms read their OWN config snapshot (taken when entering run mode),
//     so admin edits of GlobalConfig never affect already-running rooms.
//   - Never mutate the maps inside Config in place: the admin panel always
//     replaces the whole maps, or snapshots of other rooms would observe it.
package gameplay

import (
	"sync"

	"github.com/prolarps/urbanist/backend/config"
)

// NotYetImplemented is the shared toast text for protocol frames whose
// game-loop stage (move, chunks, act, chat) has not been built yet.
const NotYetImplemented = "not implemented at this stage"

// GlobalConfig is the mutable server-wide gameplay configuration.
// The administrator panel edits it; new rooms copy it on start.
var globalLock sync.RWMutex

var GlobalConfig = config.DefaultConfig

// GetGlobalConfig returns a copy of the current global configuration.
func GetGlobalConfig() config.Config {
	globalLock.RLock()
	defer globalLock.RUnlock()
	return GlobalConfig
}

// SetGlobalConfig replaces the global configuration atomically.
func SetGlobalConfig(cfg config.Config) {
	globalLock.Lock()
	defer globalLock.Unlock()
	GlobalConfig = cfg
}
