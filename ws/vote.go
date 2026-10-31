package ws

import (
	"errors"
	"sort"
	"sync"
	"time"
)

// Proposal is one of the plot-defined renovation options the players choose
// between (new residential complex, blocks of flats, park renovation, ...).
type Proposal struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Cost        int    `json:"cost"`    // budget price, arbitrary units
	Benefit     int    `json:"benefit"` // expected payoff for the district
}

// defaultProposals are the campaign options for the old-district renovation.
var defaultProposals = []Proposal{
	{
		ID:          "residential",
		Title:       "New residential complex",
		Description: "Modern apartment cluster with courtyards on the vacant lots.",
		Cost:        480,
		Benefit:     610,
	},
	{
		ID:          "flats",
		Title:       "New blocks of flats",
		Description: "Budget social housing blocks; quick build, modest payoff.",
		Cost:        320,
		Benefit:     400,
	},
	{
		ID:          "park",
		Title:       "Renovate the park",
		Description: "Restore the historic park: paths, trees, lighting, benches.",
		Cost:        150,
		Benefit:     260,
	},
}

// voteDecidePeriod is how long players have to gather information and vote.
// Short for now so the loop is easy to test; tune for real sessions.
const voteDecidePeriod = 3 * time.Minute

// Vote is the room's decision phase: open -> voting -> decided.
type Vote struct {
	room *Room

	mu       sync.Mutex
	round    int
	open     bool
	deadline time.Time
	ballots  map[string]string // clientId -> proposal id
}

func NewVote(room *Room) *Vote {
	return &Vote{room: room, ballots: map[string]string{}}
}

// OpenVote starts a decision round and announces it to the room.
func (v *Vote) OpenVote(proposals []Proposal, timeout time.Duration) {
	if len(proposals) == 0 {
		proposals = defaultProposals
	}
	if timeout <= 0 {
		timeout = voteDecidePeriod
	}

	v.mu.Lock()
	v.round++
	roundDeadline := time.Now().Add(timeout)
	v.open = true
	v.deadline = roundDeadline
	v.ballots = map[string]string{}
	v.mu.Unlock()

	v.room.broadcast(Envelope{
		Type: TypeVoteOpen,
		Data: mustJSON(VoteOpen{Proposals: proposals, Deadline: roundDeadline.Unix()}),
	})
	// The room keeps serving chat/chunks while the round waits.
	go v.decideAt(roundDeadline)
}

// cast records (or re-records) a player's vote. Changing your mind is
// allowed until the deadline: the new ballot simply overwrites the old one.
func (v *Vote) cast(c *Client, proposalID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.open {
		return errors.New("no vote is open")
	}
	if proposalByID(defaultProposals, proposalID) == nil {
		return errors.New("unknown proposal")
	}

	v.ballots[c.ID] = proposalID
	v.room.broadcast(Envelope{Type: TypeVoteTally, Data: mustJSON(VoteTally{Votes: v.countsLocked(), Total: len(v.ballots)})})
	return nil
}

// decideAt resolves the round once its deadline passes, unless the same
// deadline was already cleared by a newer round.
func (v *Vote) decideAt(roundDeadline time.Time) {
	t := time.NewTimer(time.Until(roundDeadline))
	defer t.Stop()
	<-t.C

	v.mu.Lock()
	defer v.mu.Unlock()
	if v.open && v.deadline.Equal(roundDeadline) {
		v.finishLocked()
	}
}

// finishLocked resolves the round and broadcasts the winner; mu must be held.
func (v *Vote) finishLocked() {
	if !v.open {
		return
	}
	v.open = false

	winnerID, tally := v.winnerLocked()
	v.room.broadcast(Envelope{Type: TypeVoteResult, Data: mustJSON(VoteResult{Winner: winnerID, Tally: tally})})

	// The winning proposal reshapes the district; chunk updates go out to all.
	if winner := proposalByID(defaultProposals, winnerID); winner != nil {
		coords := proposalChunks(*winner)
		v.room.world.ApplyProposal(*winner, coords)
		v.room.broadcastChunkUpdate(coords, "decision: "+winner.Title)
	}
}

func (v *Vote) syncWith(c *Client) {
	v.mu.Lock()
	open, deadline := v.open, v.deadline
	v.mu.Unlock()

	if open {
		c.sendEnvelope(Envelope{
			Type: TypeVoteOpen,
			Data: mustJSON(VoteOpen{Proposals: defaultProposals, Deadline: deadline.Unix()}),
		})
	}
}

// countsLocked returns votes per proposal; mu must be held.
func (v *Vote) countsLocked() map[string]int {
	counts := map[string]int{}
	for _, pid := range v.ballots {
		counts[pid]++
	}
	return counts
}

// winnerLocked picks the plurality winner; ties go to the most profitable
// proposal per the plot ("the most profitable will be chosen") and then to
// lower cost. If nobody voted, the default stands.
func (v *Vote) winnerLocked() (string, map[string]int) {
	counts := v.countsLocked()

	sorted := make([]Proposal, len(defaultProposals))
	copy(sorted, defaultProposals)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if counts[a.ID] != counts[b.ID] {
			return counts[a.ID] > counts[b.ID]
		}
		if a.Benefit != b.Benefit {
			return a.Benefit > b.Benefit
		}
		return a.Cost < b.Cost
	})

	if len(sorted) == 0 {
		return "", counts
	}
	return sorted[0].ID, counts
}

func proposalByID(props []Proposal, id string) *Proposal {
	for i := range props {
		if props[i].ID == id {
			return &props[i]
		}
	}
	return nil
}

// proposalChunks is the set of chunks repainted by a proposal. For now every
// renovation covers the central district; per-proposal plots come later.
func proposalChunks(p Proposal) [][2]int {
	return [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}
}
