package psqlib

import (
	"../psql"
)

func (s PSQLIB) sqlError(when string, err error) error {
	return psql.SQLError(s.log, when, err)
}