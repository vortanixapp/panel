package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vortanixapp/panel/pkg/gamecatalog"
	"github.com/vortanixapp/panel/pkg/gameconsole"
)

func (h *Handler) GetServerConsoleProfile(w http.ResponseWriter, r *http.Request) {
	serverID := chi.URLParam(r, "id")
	if _, ok := h.authorizeServerTab(w, r, serverID, "console_attach"); !ok {
		return
	}
	ctx := r.Context()

	var gameID string
	if err := h.dbOf(ctx).QueryRow(ctx,
		`SELECT COALESCE(game_id, '') FROM core.servers WHERE id = $1`, serverID,
	).Scan(&gameID); err != nil {
		writeError(w, http.StatusNotFound, "server not found")
		return
	}

	profile := gameconsole.For(gameID)
	game := ""
	if g, ok := gamecatalog.Resolve(gameID); ok {
		game = g.Name
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"game":     gameID,
		"title":    game,
		"tailored": gameconsole.Known(gameID),
		"profile":  profile,
	})
}
