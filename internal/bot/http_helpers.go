package bot

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes JSON response and logs error if any
func (b *Bot) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		b.logger.Error("Failed to encode JSON response", err)
	}
}

// writeJSONSuccess writes JSON success response
func (b *Bot) writeJSONSuccess(w http.ResponseWriter, data interface{}) {
	b.writeJSON(w, map[string]interface{}{
		"success": true,
		"data":    data,
	})
}
