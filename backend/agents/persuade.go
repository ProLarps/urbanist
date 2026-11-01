package agents

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
)

// MaxShift caps one persuasion attempt (§8): a single chat message may
// move the citizens' will at most by this much — anti-exploit for spam.
const MaxShift = 0.10

// Reply is what the hub sends back after one chat persuasion attempt:
// the in-character answer plus the (already clamped) will shift to apply.
type Reply struct {
	Text      string
	WillShift float64
}

// Prompt context describing the chunk the citizens live in.
type Context struct {
	RennovationName string // "панелька", "парк", ... – flavour only
	Will            float64
}

// systemPrompt fixes the persona AND the machine-readable verdict format.
// The trailing [WILL:+0.05] token is parsed out and never shown to players.
func systemPrompt(ctx Context) string {
	return fmt.Sprintf(
		"Ты — собирательный голос жителей квартала «%s» в игре-градостроителе. "+
			"Текущая воля жителей к застройщику: %.2f (от 0.0 — вражда, до 1.0 — полная поддержка). "+
			"Игрок пишет тебе, чтобы убедить поддержать его план реновации. "+
			"Отвечай коротко (1-3 предложения) на русском, от лица жителей, эмоционально. "+
			"Повторяющиеся одинаковые аргументы убеждают всё меньше. "+
			"В конце ответа ОБЯЗАТЕЛЬНО добавь служебную метку вида [WILL:+0.05], "+
			"где число от -0.10 до +0.10 — насколько реплика игрока сдвинула вашу волю.",
		ctx.RennovationName, ctx.Will)
}

// willToken finds the LAST [WILL:...] marker; models sometimes echo it twice.
var willToken = regexp.MustCompile(`(?i)\[will:\s*([+-]?[0-9]*\.?[0-9]+)\s*\]`)

// Persuade asks Ollama to react to one player message. On ANY failure
// (disabled client, offline daemon, timeout, missing token) it returns a
// safe fallback so the game keeps running without an LLM.
func (c *Client) Persuade(ctx context.Context, pc Context, playerName, text string) Reply {
	fallback := Reply{Text: "Жители переглядываются и молчат (офлайн-режим).", WillShift: 0}
	if !c.enabled || text == "" {
		return fallback
	}
	user := fmt.Sprintf("Игрок «%s» пишет жителям: %s", playerName, text)
	answer, err := c.Chat(ctx, systemPrompt(pc), user)
	if err != nil || answer == "" {
		return fallback
	}
	shift := parseShift(answer)
	clean := willToken.ReplaceAllString(answer, "")
	return Reply{Text: trim(clean), WillShift: clampShift(shift)}
}

// parseShift extracts the numeric part of the last WILL token; absent → 0.
func parseShift(answer string) float64 {
	m := willToken.FindAllStringSubmatch(answer, -1)
	if len(m) == 0 {
		return 0
	}
	v, err := strconv.ParseFloat(m[len(m)-1][1], 64)
	if err != nil {
		return 0
	}
	return v
}

// clampShift keeps one verdict inside [-MaxShift, +MaxShift].
func clampShift(v float64) float64 {
	if v > MaxShift {
		return MaxShift
	}
	if v < -MaxShift {
		return -MaxShift
	}
	return v
}
