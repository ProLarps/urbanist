package gameplay

import (
	"testing"
	"time"

	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/errcodes"
)

// newRunRoom builds a started room with one member and a frozen config.
func onlineAll(string) bool { return true }

func newRunRoom(t *testing.T, id string) (*Room, *roomPlayer) {
	t.Helper()
	SetGlobalConfig(config.DefaultConfig)
	m := NewManager()
	r, code := m.CreateRoom(Member{ID: id, Name: "Test"}, "r", 2)
	if code != errcodes.OK {
		t.Fatal(code)
	}
	if code := r.Start(onlineAll); code != errcodes.OK {
		t.Fatal(code)
	}
	p := r.members[id]
	if p == nil || len(p.granted) != 1 { // §8: first grant = 1 chunk
		t.Fatalf("seed loop failed: %+v", p)
	}
	return r, p
}

// heldChunkPos returns the single held chunk of the player.
func heldChunkPos(p *roomPlayer) chunkPos {
	for c := range p.granted {
		return c
	}
	panic("no granted chunk")
}

func TestPickAndRenovateFullRound(t *testing.T) {
	r, p := newRunRoom(t, "p1")
	c := heldChunkPos(p)

	evs, code := r.Pick(p.ID, c.cx, c.cz)
	if code != errcodes.OK || evs[0].Kind != "assigned" {
		t.Fatalf("pick: %v %+v", code, evs)
	}
	d := r.world.chunks[c]
	needs := needsCells(d)
	if len(needs) == 0 {
		t.Fatal("generated district has no renovation cells")
	}
	moneyBefore := p.Money
	if _, code := r.Renovate(p.ID, c.cx, c.cz, needs[0], "rebuild"); code != errcodes.OK {
		t.Fatalf("renovate: %v", code)
	}
	wantCost, _ := projectCost("rebuild")
	if p.Money != moneyBefore-wantCost {
		t.Fatalf("cost not charged: %f", p.Money)
	}
	// Renovating outside the assigned chunk must be rejected (§8 bans).
	if _, code := r.Renovate(p.ID, 9, 9, 0, "repair"); code != errcodes.Forbidden {
		t.Fatalf("expected Forbidden for foreign chunk, got %v", code)
	}
	// Not enough money must be rejected (§8 instantaneous toast path).
	p.Money = 1
	if _, code := r.Renovate(p.ID, c.cx, c.cz, needs[len(needs)-1], "demolish"); code != errcodes.BadRequest {
		t.Fatalf("expected BadRequest on bankrupt renovate, got %v", code)
	}
	// Finish every remaining needs-cell -> round closes with a fresh grant.
	p.Money = r.cfg.StartMoney
	for _, cell := range needs[1:] {
		if _, code := r.Renovate(p.ID, c.cx, c.cz, cell, "repair"); code != errcodes.OK {
			t.Fatalf("renovate rest: %v", code)
		}
	}
	if p.cycle != 1 || p.assigned != nil {
		t.Fatalf("round did not close: cycle=%d assigned=%v", p.cycle, p.assigned)
	}
	if len(p.granted) == 0 {
		t.Fatal("no next grant after finishing (§8 step 3)")
	}
	if drained := r.DrainEvents(); len(drained) == 0 {
		t.Fatal("grant event missing from outbox")
	}
}

func TestTimeoutPenalty(t *testing.T) {
	r, p := newRunRoom(t, "p1")
	c := heldChunkPos(p)
	if _, code := r.Pick(p.ID, c.cx, c.cz); code != errcodes.OK {
		t.Fatal(code)
	}
	money, rating := p.Money, p.Rating
	r.Tick(nowFunc().Add(time.Duration(r.cfg.RoundSeconds+2) * time.Second))
	if p.Money >= money || p.Rating >= rating {
		t.Fatalf("timeout punished nothing: %f/%f -> %f/%f",
			money, rating, p.Money, p.Rating)
	}
	if p.assigned != nil || p.cycle != 1 {
		t.Fatal("timeout must release the assignment and advance the cycle")
	}
	if _, ok := p.granted[c]; !ok {
		t.Fatal("timed-out chunk stays pending and ages (§8)")
	}
}

func TestSkipAfterFourCycles(t *testing.T) {
	r, p := newRunRoom(t, "p1")
	c := heldChunkPos(p)
	money, rating := p.Money, p.Rating
	limit := r.cfg.MaxCyclesHeld
	for i := 0; i < limit+1; i++ {
		r.skipExpired(p) // each idle cycle ages the held chunk by one
	}
	if _, ok := p.granted[c]; ok {
		t.Fatal("chunk should have been skipped away")
	}
	if p.skipped != 1 || p.Rating >= rating || p.Money >= money {
		t.Fatalf("skip penalty wrong: skipped=%d %f/%f", p.skipped, p.Money, p.Rating)
	}
}

func TestDefeatConditions(t *testing.T) {
	r, p := newRunRoom(t, "p1")
	p.Money = 1 // below the cheapest scenario cost
	evs := r.events(p)
	if p.Alive || lastKind(evs) != "gameOver" {
		t.Fatal("bankruptcy must fire gameOver/bankrupt")
	}
	// Fresh wait-mode room to test the rating path without side effects.
	r2, p2 := newRunRoom(t, "p2")
	p2.Money, p2.Rating = 400, 0.1
	evs = r2.events(p2)
	if p2.Alive || lastKind(evs) != "gameOver" {
		t.Fatal("low rating must fire gameOver/low_rating")
	}
}

func lastKind(evs []GameEvent) string {
	if len(evs) == 0 {
		return ""
	}
	return evs[len(evs)-1].Kind
}

func TestDeterministicGeneration(t *testing.T) {
	cfg := config.DefaultConfig
	w1, w2 := NewWorld(42, cfg), NewWorld(42, cfg)
	for cz := range config.ChunksPerAxis {
		for cx := range config.ChunksPerAxis {
			a, _ := w1.Chunk(cx, cz)
			b, _ := w2.Chunk(cx, cz)
			if a.Rennovation != b.Rennovation || a.Size != b.Size {
				t.Fatalf("seeded generation diverges at (%d,%d)", cx, cz)
			}
		}
	}
}
