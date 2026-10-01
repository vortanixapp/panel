package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/i18n"
	"github.com/vortanixapp/panel/pkg/notify"
)

const refreshGraceSeconds = 30

type refreshAction int

const (
	refreshRotate refreshAction = iota
	refreshRepeat
	refreshReuse
)

type refreshState struct {
	current  string
	previous string
	graceOK  bool
}

func decideRefresh(sessionID, presented string, st refreshState) refreshAction {
	switch {
	case st.current == "" && presented == sessionID:
		return refreshRotate
	case st.current != "" && presented == st.current:
		return refreshRotate
	case st.current != "" && st.previous != "" && presented == st.previous && st.graceOK:
		return refreshRepeat
	}
	return refreshReuse
}

type refreshOutcome int

const (
	refreshOutcomeRotated refreshOutcome = iota
	refreshOutcomeRepeated
	refreshOutcomeReused
	refreshOutcomeGone
)

func (h *Handler) rotateRefresh(ctx context.Context, userID, sessionID, presented string) (string, refreshOutcome, error) {
	tx, err := h.dbOf(ctx).Begin(ctx)
	if err != nil {
		return "", refreshOutcomeGone, err
	}
	defer tx.Rollback(ctx)

	var current, previous *string
	var st refreshState
	err = tx.QueryRow(ctx, `
		SELECT refresh_jti, prev_refresh_jti,
		       COALESCE(rotated_at > now() - make_interval(secs => $3::float8), false)
		FROM core.user_sessions
		WHERE id = $1 AND user_id = $2
		FOR UPDATE
	`, sessionID, userID, float64(refreshGraceSeconds)).Scan(&current, &previous, &st.graceOK)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", refreshOutcomeGone, nil
	}
	if err != nil {
		return "", refreshOutcomeGone, err
	}
	if current != nil {
		st.current = *current
	}
	if previous != nil {
		st.previous = *previous
	}

	switch decideRefresh(sessionID, presented, st) {
	case refreshRotate:
		next := randomToken(16)
		if _, err := tx.Exec(ctx, `
			UPDATE core.user_sessions
			SET prev_refresh_jti = $2, refresh_jti = $3, rotated_at = now()
			WHERE id = $1
		`, sessionID, presented, next); err != nil {
			return "", refreshOutcomeGone, err
		}
		if err := tx.Commit(ctx); err != nil {
			return "", refreshOutcomeGone, err
		}
		return next, refreshOutcomeRotated, nil
	case refreshRepeat:
		if err := tx.Commit(ctx); err != nil {
			return "", refreshOutcomeGone, err
		}
		return st.current, refreshOutcomeRepeated, nil
	}

	if _, err := tx.Exec(ctx, `DELETE FROM core.user_sessions WHERE id = $1`, sessionID); err != nil {
		return "", refreshOutcomeGone, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", refreshOutcomeGone, err
	}
	return "", refreshOutcomeReused, nil
}

func (h *Handler) onRefreshReuse(ctx context.Context, r *http.Request, userID, sessionID string) {
	var email string
	_ = h.dbOf(ctx).QueryRow(ctx, `SELECT email FROM core.users WHERE id = $1`, userID).Scan(&email)
	ip := clientIP(r)
	audit(ctx, h.dbOf(ctx), userID, "auth.refresh_reuse", "session:"+sessionID, map[string]any{"ip": ip})
	h.recordLoginAttempt(ctx, r, userID, email, "повторное использование токена обновления, сеанс закрыт", false)

	where := i18n.Raw(ip)
	if ip == "" {
		where = i18n.Key("notify.new_login.unknown_ip")
	}
	h.notifyUser(ctx, userID, notify.Event{
		Kind:      notify.KindSessionRevoked,
		Title:     i18n.Key("notify.session_revoked.title"),
		Body:      i18n.Key("notify.session_revoked.body", i18n.Params{"ip": where}),
		Action:    h.panelAction("notify.action.sessions", "/settings?tab=sessions"),
		Meta:      map[string]any{"ip": ip},
		DedupeKey: "session.revoked:" + sessionID,
	})
}
