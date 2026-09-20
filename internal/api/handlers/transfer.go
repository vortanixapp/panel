package handlers

import (
	"context"
	"net/http"
	"time"
)

const transferDeadline = 2 * time.Hour

func extendTransfer(w http.ResponseWriter, r *http.Request) (*http.Request, context.CancelFunc) {
	deadline := time.Now().Add(transferDeadline)
	rc := http.NewResponseController(w)
	_ = rc.SetReadDeadline(deadline)
	_ = rc.SetWriteDeadline(deadline)
	ctx, cancel := context.WithDeadline(context.WithoutCancel(r.Context()), deadline)
	return r.WithContext(ctx), cancel
}
