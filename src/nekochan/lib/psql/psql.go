package psql

// not-too-generic PSQL connector
// can be used by more concrete forum packages

import (
	. "../logx"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"time"
)

type Config struct {
	ConnStr         string
	ConnMaxLifetime float64
	MaxIdleConns    int32
	MaxOpenConns    int32
	Logger          LoggerX
}

var DefaultConfig = Config{
	ConnStr:         "",
	ConnMaxLifetime: 0.0,
	MaxIdleConns:    0,
	MaxOpenConns:    0,
}

type PSQL struct {
	DB  *sqlx.DB
	log Logger
	id  string
}

func OpenPSQL(cfg Config) (PSQL, error) {
	db, err := sqlx.Open("postgres", cfg.ConnStr)
	if err != nil {
		return PSQL{}, err
	}

	if cfg.ConnMaxLifetime > 0.0 {
		db.SetConnMaxLifetime(time.Duration(float64(time.Second) *
			cfg.ConnMaxLifetime))
	}

	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(int(cfg.MaxIdleConns))
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(int(cfg.MaxOpenConns))
	}

	p := PSQL{DB: db}
	p.id = fmt.Sprintf("psqlib.%p", p.DB)
	p.log = NewLogToX(cfg.Logger, p.id)

	return p, nil
}

func (p PSQL) Close() error {
	return p.DB.Close()
}

func (p PSQL) ID() string {
	return p.id
}