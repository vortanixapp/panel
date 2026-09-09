package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pools struct {
	Write *pgxpool.Pool
	Read  *pgxpool.Pool
}

func Connect(ctx context.Context, writeURL, readURL string) (*Pools, error) {
	writePool, err := pgxpool.New(ctx, writeURL)
	if err != nil {
		return nil, fmt.Errorf("write pool: %w", err)
	}
	if readURL == "" || readURL == writeURL {
		return &Pools{Write: writePool, Read: writePool}, nil
	}
	readPool, err := pgxpool.New(ctx, readURL)
	if err != nil {
		writePool.Close()
		return nil, fmt.Errorf("read pool: %w", err)
	}
	return &Pools{Write: writePool, Read: readPool}, nil
}

func (p *Pools) Close() {
	if p == nil {
		return
	}
	if p.Write != nil {
		p.Write.Close()
	}
	if p.Read != nil && p.Read != p.Write {
		p.Read.Close()
	}
}

func (p *Pools) Reader() *pgxpool.Pool {
	if p == nil || p.Read == nil {
		return p.Write
	}
	return p.Read
}
