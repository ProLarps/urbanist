package ws

// Message type constants, docs/API.md §5 (client -> server).
const (
	MsgAuth        = "auth"
	MsgMove        = "move"
	MsgNeedChunks  = "needChunks"
	MsgChunkPick   = "chunkPick"
	MsgAct         = "act"
	MsgChat        = "chat"
	MsgChatEnd     = "chatEnd"
	MsgChatHistory = "chatHistory"
	MsgSubscribe   = "subscribe"
	MsgPing        = "ping"
)

// Server -> client, docs/API.md §6.
const (
	MsgAuthOk      = "authOk"
	MsgAuthFail    = "authFail"
	MsgGrant       = "grant"
	MsgAssigned    = "assigned"
	MsgChunks      = "chunks"
	MsgChunkDelta  = "chunkDelta"
	MsgChunkRevoke = "chunkRevoke"
	MsgPlayers     = "players"
	MsgPlayerUpd   = "playerUpdate"
	MsgPlayerLeave = "playerLeave"
	MsgNpcUpd      = "npcUpdate"
	MsgAgentSay    = "agentSay"
	MsgAgentEnd    = "agentEnd"
	MsgChatHist    = "chatHistory"
	MsgStats       = "stats"
	MsgToast       = "toast"
	MsgGameOver    = "gameOver"
	MsgJournal     = "journal"
	MsgPong        = "pong"
)

// Renovation scenarios (docs/API.md §8).
const (
	ScenarioRepair     = "repair"
	ScenarioRebuild    = "rebuild"
	ScenarioRevitalize = "revitalize"
	ScenarioDemolish   = "demolish"
)

// scenario describes one renovation variant of a building.
type scenario struct {
	CostMult   float64 // cost multiplier
	ProfitRate float64 // one-time profit as fraction of cost
	Ticks      int     // construction duration at 10 Hz
	WillShift  float64 // base citizens' will shift for the chunk's opinion
}

var scenarios = map[string]scenario{
	ScenarioRepair:     {CostMult: 0.4, ProfitRate: 0.40, Ticks: 40, WillShift: 0.05},
	ScenarioRebuild:    {CostMult: 1.0, ProfitRate: 0.90, Ticks: 90, WillShift: -0.05},
	ScenarioRevitalize: {CostMult: 1.5, ProfitRate: 1.30, Ticks: 120, WillShift: 0.10},
	ScenarioDemolish:   {CostMult: 2.0, ProfitRate: 1.70, Ticks: 150, WillShift: -0.10},
}

// ClientMsg is the flat {"t": ...} frame. Optional fields are pointers so
// one struct can decode every message type without double-guessing.
type ClientMsg struct {
	T string `json:"t"`

	Token  *string `json:"token,omitempty"`
	Name   *string `json:"name,omitempty"`
	Avatar *int    `json:"avatar,omitempty"`

	X    *float64 `json:"x,omitempty"`
	Z    *float64 `json:"z,omitempty"`
	Dir  *int     `json:"dir,omitempty"`
	Anim *string  `json:"anim,omitempty"`

	CX *int `json:"cx,omitempty"`
	CZ *int `json:"cz,omitempty"`

	LotID    *int    `json:"lotId,omitempty"`
	Action   *string `json:"action,omitempty"`
	Scenario *string `json:"scenario,omitempty"`

	AgentID *string `json:"agentId,omitempty"`
	Text    *string `json:"text,omitempty"`

	// NOTE: ping's payload field is also named "t" per spec and numeric, so
	// ping frames are decoded via a map, not this struct.
	PingT *float64 `json:"pingT,omitempty"`

	Channels []string `json:"channels,omitempty"`
}

// Stats is the HUD payload (docs/API.md §6).
type Stats struct {
	Money   float64      `json:"money"`
	Rating  float64      `json:"rating"`
	Cycle   int          `json:"cycle"`
	Pending []PendingRow `json:"pending"`
	Skipped int          `json:"skipped"`
}

type PendingRow struct {
	CX         int    `json:"cx"`
	CZ         int    `json:"cz"`
	CyclesHeld int    `json:"cyclesHeld"`
	Deadline   int64  `json:"deadline,omitempty"` // unix, only for the assigned one
	Status     string `json:"status"`
}

// Toast kinds.
const (
	ToastInfo  = "info"
	ToastWarn  = "warn"
	ToastError = "error"
	ToastOk    = "ok"
)

// GameOver reasons.
const (
	LoseBankrupt = "bankrupt"
	LoseRating   = "low_rating"
)
