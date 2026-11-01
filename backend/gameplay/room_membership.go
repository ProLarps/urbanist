package gameplay

import (
	"github.com/prolarps/urbanist/backend/errcodes"
)

// JoinGuard is the shared pre-check for room entry points: run-mode rooms
// are closed for joins and capacity limits are enforced.
func (r *Room) JoinGuard(m Member) errcodes.Code {
	if r.mode == ModeRun {
		return errcodes.RoomBusy
	}
	if len(r.members) >= r.MaxPlayers {
		return errcodes.RoomFull
	}
	return errcodes.OK
}

// Join adds playerID. Only possible in wait mode, so mid-run rooms reject;
// joining twice is an idempotent success (RTS-lobby semantics).
func (r *Room) Join(m Member) errcodes.Code {
	r.mu.Lock()
	defer r.mu.Unlock()
	if code := r.JoinGuard(m); code != errcodes.OK {
		return code
	}
	if _, ok := r.members[m.ID]; ok {
		return errcodes.OK
	}
	r.members[m.ID] = &roomPlayer{
		Member: m,
		Money:  r.moneyOnJoin(),
		Rating: r.ratingOnJoin(),
		Alive:  true,
	}
	return errcodes.OK
}

// Leave removes the member; returns whether the player was present at all.
// Anyone may leave, the owner included — the room survives (§2.4 v3).
func (r *Room) Leave(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, existed := r.members[playerID]
	delete(r.members, playerID)
	return existed
}

// moneyOnJoin reads the CURRENT global config: the owner creates rooms
// with today's economy, while run-mode rooms use their frozen snapshot.
func (r *Room) moneyOnJoin() float64 { return GetGlobalConfig().StartMoney }
func (r *Room) ratingOnJoin() float64 {
	return GetGlobalConfig().RatingStart
}

// clampRange limits v into [lo, hi] for user-supplied numbers.
func clampRange(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
