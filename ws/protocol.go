package ws

import "encoding/json"

// Message types sent by the client.
const (
	TypeChatSend  = "chat_send"
	TypeChunkReq  = "chunk_req"
	TypeVoteStart = "vote_start"
	TypeVoteCast  = "vote_cast"
	TypeNpcTalk   = "npc_talk"
	TypeSetName   = "set_name"
)

// Message types sent by the server.
const (
	TypeJoined     = "joined"
	TypeLeft       = "left"
	TypeChatMsg    = "chat_msg"
	TypeChatHist   = "chat_history"
	TypeMemberUpd  = "member_update"
	TypeChunk      = "chunk"
	TypeChunkUpd   = "chunk_update"
	TypeVoteOpen   = "vote_open"
	TypeVoteTally  = "vote_tally"
	TypeVoteResult = "vote_result"
	TypeNpcReply   = "npc_reply"
	TypeError      = "error"
)

// Envelope is every message in both directions: {"type": "...", ...payload}.
type Envelope struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// ChatMsg is payload for chat_send (client) and chat_msg (server).
type ChatMsg struct {
	AuthorID string `json:"authorId,omitempty"` // omitted in client->server
	Name     string `json:"name,omitempty"`     // set by server
	Text     string `json:"text"`
}

// ChatHistory is payload for chat_history.
type ChatHistory struct {
	Messages []ChatMsg `json:"messages"`
}

// Joined / Left notification.
type MemberInfo struct {
	ClientID string `json:"clientId"`
	Name     string `json:"name,omitempty"`
}

// ChunkReq: client asks for chunk at (X, Y).
type ChunkReq struct {
	X, Y int
}

// Tile is a single terrain cell. Kind maps to the front-end tile atlas.
type Tile struct {
	Kind int `json:"kind"`
}

// ChunkData: one chunk of tiles, deterministic from the district seed.
type ChunkData struct {
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Tiles []Tile `json:"tiles"`
}

// ChunkUpdate: broadcast after a decision modifies the district.
type ChunkUpdate struct {
	ChunkData
	Reason string `json:"reason"`
}

// VoteStart: client asks to open a decision round (empty proposals -> defaults).
type VoteStart struct {
	TimeoutSec int `json:"timeoutSec,omitempty"`
}

// VoteOpen: start the information+decision phase.
type VoteOpen struct {
	Proposals []Proposal `json:"proposals"`
	Deadline  int64      `json:"deadline"` // unix seconds
}

// VoteCast: client votes for a proposal (re-cast changes the vote).
type VoteCast struct {
	ProposalID string `json:"proposalId"`
}

// VoteTally: poll snapshot broadcast after each vote change.
type VoteTally struct {
	Votes map[string]int `json:"votes"`
	Total int            `json:"total"`
}

// VoteResult: winner at deadline.
type VoteResult struct {
	Winner string         `json:"winner"`
	Tally  map[string]int `json:"tally"`
	Reason string         `json:",omitempty"`
}

// NpcTalk / NpcReply.
type NpcTalk struct {
	NpcID string `json:"npcId"`
	Text  string `json:"text"`
}

type NpcReply struct {
	NpcID string `json:"npcId"`
	Text  string `json:"text"`
}

// ErrorMsg is sent when a client message cannot be processed.
type ErrorMsg struct {
	Message string `json:"message"`
}
