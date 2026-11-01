package agents

import (
	"strings"

	"github.com/prolarps/urbanist/backend/config"
)

// renovationNames gives each district type a human flavour for the prompt.
var renovationNames = [config.RENNOVATIONS_TOTAL]string{
	config.RENNOVATE_PANEL_FLAT: "панельные девятиэтажки",
	config.RENNOVATE_PARK:       "городской парк",
	config.RENNOVATE_ALLEY:      "узкие переулки",
	config.RENNOVATE_OLD_HOUSE:  "старый фонд",
	config.RENNOVATE_LAKE:       "озёрная набережная",
}

// RenovationName maps a district's renovation id onto its display name.
func RenovationName(id int) string {
	if id < 0 || id >= len(renovationNames) {
		return "спальный район"
	}
	return renovationNames[id]
}

// trim removes the leftover marker line and surrounding whitespace so the
// player never sees the machine-readable verdict token.
func trim(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "Жители молча кивают."
	}
	return s
}
