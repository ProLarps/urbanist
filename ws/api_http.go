package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// Server owns config, the store and all HTTP routes (§2); it also creates
// the WS hub per room.
type Server struct {
	cfg   Config
	store *Store
	auth  *Authenticator

	rooms *Hub // per-room game hubs
}

func NewServer(cfg Config) *Server {
	return &Server{cfg: cfg, rooms: NewHub(cfg)}
}

// Run opens the store, wires routes and serves. It blocks.
func (s *Server) Run(addr string, dbPath string) error {
	store, err := OpenStore(dbPath)
	if err != nil {
		return err
	}
	s.store = store

	auth, err := NewAuthenticator(s.cfg.SecretDir+"/secret.key", s.cfg.TokenTTL)
	if err != nil {
		return err
	}
	s.auth = auth

	s.rooms.attach(s) // let the WS hub reach the store/config

	// The default room auto-exists so v1 clients sending room="default" work.
	if r, _ := s.store.GetRoom("default"); r == nil {
		if _, err := s.store.CreateRoomWithCode("default", "", "Default room", s.cfg.multiMaxPlayers()); err != nil {
			return err
		}
	}

	mux := http.NewServeMux()
	s.routes(mux)

	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	err = srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (cfg Config) multiMaxPlayers() int { return 8 }

// bearer extracts/parses the Authorization: Bearer token on API routes.
func (s *Server) bearer(r *http.Request) (*tokenPayload, bool) {
	h := r.Header.Get("Authorization")
	tok, ok := strings.CutPrefix(h, "Bearer ")
	if !ok {
		return nil, false
	}
	payload, err := s.auth.Verify(tok)
	return payload, err == nil
}

func (s *Server) routes(mux *http.ServeMux) {
	// Spec §2.1
	mux.HandleFunc("POST /api/auth", s.handleAuth)
	// Spec §2.2
	mux.HandleFunc("GET /api/health", s.handleHealth)
	// Spec §2.3
	mux.HandleFunc("GET /api/players", s.handlePlayers)
	mux.HandleFunc("GET /api/players/{id}", s.handlePlayerProfile)
	mux.HandleFunc("GET /api/leaderboard", s.handleLeaderboard)

	// Rooms CRUD, spec §2.4
	mux.HandleFunc("POST /api/rooms", s.handleRoomCreate)
	mux.HandleFunc("GET /api/rooms", s.handleRoomList)
	mux.HandleFunc("GET /api/rooms/{code}", s.handleRoomGet)
	mux.HandleFunc("GET /api/rooms/{code}/scoreboard", s.handleRoomScoreboard)
	mux.HandleFunc("PATCH /api/rooms/{code}", s.handleRoomPatch)
	mux.HandleFunc("DELETE /api/rooms/{code}", s.handleRoomDelete)

	// WS, spec §3
	mux.HandleFunc("GET /ws", s.rooms.ServeWS)

	// Static client: spec §2.4
	mux.Handle("GET /", http.FileServer(http.Dir("static")))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func apiError(w http.ResponseWriter, status int, code, detail string) {
	body := map[string]any{"error": code}
	if detail != "" {
		body["detail"] = detail
	}
	writeJSON(w, status, body)
}

const scorePrecision = 2

// ---- §2.1 POST /api/auth ----------------------------------------------

type authRequest struct {
	Name   string `json:"name"`
	Avatar *int   `json:"avatar"`
	Room   string `json:"room"`
}

func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	var body authRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusBadRequest, "bad_request", "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if len(name) < 1 || len(name) > 24 {
		apiError(w, http.StatusBadRequest, "bad_request", "имя должно быть от 1 до 24 символов")
		return
	}
	avatar := 0
	if body.Avatar != nil {
		avatar = clampInt(*body.Avatar, 0, 7)
	}
	room := body.Room
	if room == "" {
		room = "default"
	}
	roomRow, err := s.store.GetRoom(room)
	if err != nil || roomRow == nil {
		apiError(w, http.StatusNotFound, "room_not_found", "")
		return
	}

	player, err := s.store.CreatePlayer(name, avatar, paletteByAvatar(avatar))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	token, err := s.auth.Issue(player.ID, player.Name, room)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token":    token,
		"playerId": player.ID,
		"name":     player.Name,
		"color":    player.Color,
		"world":    s.worldInfo(),
		"protocol": 2,
	})
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// paletteByAvatar: 8 fixed colors for players (§2.1).
func paletteByAvatar(n int) string {
	palette := [8]string{
		"#4f9dff", "#ffb64f", "#8bd450", "#ff6f6f",
		"#b28ad6", "#ff8bd1", "#4fd1c9", "#c9b881",
	}
	return palette[n%len(palette)]
}

func (s *Server) worldInfo() map[string]any {
	total := s.cfg.ChunksPerAxis * s.cfg.ChunksPerAxis * s.cfg.LotsPerChunk * s.cfg.LotsPerChunk
	return map[string]any{
		"chunks":       s.cfg.ChunksPerAxis * s.cfg.ChunksPerAxis,
		"lotsPerChunk": s.cfg.LotsPerChunk * s.cfg.LotsPerChunk,
		"totalLots":    total,
		"viewChunks":   s.cfg.ViewChunks * s.cfg.ViewChunks,
	}
}

// ---- §2.2 GET /api/health ----------------------------------------------

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	players, _ := s.store.CountRoomPlayers("") // live sessions across rooms
	total := s.cfg.ChunksPerAxis * s.cfg.ChunksPerAxis * s.cfg.LotsPerChunk * s.cfg.LotsPerChunk
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"uptime":    time.Since(startTime).Seconds(),
		"players":   players,
		"chunks":    s.cfg.ChunksPerAxis * s.cfg.ChunksPerAxis,
		"buildings": total,
		"tps":       s.cfg.TickHz,
	})
}

var startTime = time.Now()

// ---- §2.3 players/profiles/leaderboard ----------------------------------

func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := s.store.ListPlayers()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(players))
	for _, p := range players {
		out = append(out, map[string]any{
			"id": p.ID, "name": p.Name, "color": p.Color, "lastSeen": p.LastSeen,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePlayerProfile(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetPlayer(r.PathValue("id"))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if p == nil {
		apiError(w, http.StatusNotFound, "not_found", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": p.ID, "name": p.Name, "avatar": p.Avatar, "color": p.Color,
		"bestScore": p.BestScore, "createdAt": p.CreatedAt, "lastSeen": p.LastSeen,
	})
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.Leaderboard()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if rows == nil {
		rows = []BestScoreRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

// ---- §2.4 rooms CRUD ------------------------------------------------------

func (s *Server) handleRoomCreate(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.bearer(r)
	if !ok {
		apiError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	name := strings.TrimSpace(body.Name)
	if name == "" {
		name = "Room " + payload.Name
	}
	room, err := s.store.CreateRoom(payload.PID, name, s.cfg.multiMaxPlayers())
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"code": room.Code, "name": room.Name, "ownerId": room.OwnerID,
	})
}

func (s *Server) handleRoomList(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.bearer(r); !ok {
		apiError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
		return
	}
	rooms, err := s.store.ListRooms()
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(rooms))
	for _, room := range rooms {
		count, _ := s.store.CountRoomPlayers(room.Code)
		out = append(out, map[string]any{
			"code": room.Code, "name": room.Name, "players": count,
			"maxPlayers": room.MaxPlayers, "createdAt": room.CreatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleRoomGet(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.bearer(r); !ok {
		apiError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
		return
	}
	room, err := s.store.GetRoom(r.PathValue("code"))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if room == nil {
		apiError(w, http.StatusNotFound, "not_found", "room")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"code": room.Code, "name": room.Name, "ownerId": room.OwnerID,
		"maxPlayers": room.MaxPlayers, "players": []map[string]any{},
	})
}

func (s *Server) handleRoomScoreboard(w http.ResponseWriter, r *http.Request) {
	room, err := s.store.GetRoom(r.PathValue("code"))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if room == nil {
		apiError(w, http.StatusNotFound, "not_found", "room")
		return
	}
	rows, err := s.store.RoomScoreboard(room.Code)
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if rows == nil {
		rows = []ScoreRow{}
	}
	writeJSON(w, http.StatusOK, rows)
}

func (s *Server) handleRoomPatch(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.bearer(r)
	if !ok {
		apiError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
		return
	}
	room, err := s.store.GetRoom(r.PathValue("code"))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if room == nil {
		apiError(w, http.StatusNotFound, "not_found", "room")
		return
	}
	if room.OwnerID != payload.PID {
		apiError(w, http.StatusForbidden, "forbidden", "only the room owner can modify it")
		return
	}
	var body struct {
		Name       *string `json:"name"`
		MaxPlayers *int    `json:"maxPlayers"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	var mp *int
	if body.MaxPlayers != nil {
		v := clampInt(*body.MaxPlayers, 1, 64)
		mp = &v
	}
	if err := s.store.UpdateRoom(room.Code, body.Name, mp); err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleRoomDelete(w http.ResponseWriter, r *http.Request) {
	payload, ok := s.bearer(r)
	if !ok {
		apiError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
		return
	}
	room, err := s.store.GetRoom(r.PathValue("code"))
	if err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	if room == nil {
		apiError(w, http.StatusNotFound, "not_found", "room")
		return
	}
	if room.OwnerID != payload.PID {
		apiError(w, http.StatusForbidden, "forbidden", "only the room owner can delete it")
		return
	}
	if err := s.store.DeleteRoom(room.Code); err != nil {
		apiError(w, http.StatusInternalServerError, "internal", err.Error())
		return
	}
	s.rooms.CloseRoom(room.Code) // closes active sockets with 4404
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ---- helpers --------------------------------------------------------------

type ScoreRow struct {
	PlayerID string  `json:"playerId"`
	Name     string  `json:"name"`
	Money    float64 `json:"money"`
	Rating   float64 `json:"rating"`
	Score    float64 `json:"score"`
}

func (s *Store) RoomScoreboard(roomID string) ([]ScoreRow, error) {
	rows, err := s.db.Query(`SELECT rp.player_id, p.name, rp.money, rp.rating
		FROM room_players rp JOIN players p ON p.id = rp.player_id
		WHERE rp.room_id=? AND rp.alive=1 ORDER BY rp.money * rp.rating DESC`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScoreRow
	for rows.Next() {
		var r ScoreRow
		var money, rating float64
		if err := rows.Scan(&r.PlayerID, &r.Name, &money, &rating); err != nil {
			return nil, err
		}
		r.Money = money
		r.Rating = rating
		r.Score = roundTo(money*rating, scorePrecision)
		out = append(out, r)
	}
	return out, rows.Err()
}

func roundTo(v float64, digits int) float64 {
	p := 1.0
	for range digits {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
}
