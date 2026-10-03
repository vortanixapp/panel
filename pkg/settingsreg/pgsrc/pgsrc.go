package pgsrc

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/vortanixapp/panel/pkg/settingsreg"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func Loader(db DB) settingsreg.Loader {
	return func(ctx context.Context, keys []string) (map[string]string, error) {
		rows, err := db.Query(ctx, `SELECT key, value FROM core.tenant_settings WHERE key = ANY($1)`, keys)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := map[string]string{}
		for rows.Next() {
			var key string
			var value []byte
			if rows.Scan(&key, &value) == nil {
				out[key] = settingsreg.RawToString(value)
			}
		}
		return out, rows.Err()
	}
}
