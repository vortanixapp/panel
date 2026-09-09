package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

const singleTenantSlug = "default"

func (h *Handler) dbOf(context.Context) *pgxpool.Pool {
	return h.db
}

func (h *Handler) readerOf(context.Context) *pgxpool.Pool {
	return h.reader()
}
