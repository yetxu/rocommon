package mysql

import (
	"database/sql"
	"errors"
)

type Wrapper struct {
	drv   *sql.DB
	row   *sql.Rows
	query string

	Err error
}

var ErrDriverNotReady = errors.New("driver not ready")

func (a *Wrapper) Query(query string, args ...interface{}) *Wrapper {
	if a.drv == nil {
		a.Err = ErrDriverNotReady
		return a
	}

	a.query = query
	a.row, a.Err = a.drv.Query(query, args...)

	//log.Println("rows=", a.row)

	return a
}

func (a *Wrapper) Execute(query string, args ...interface{}) *Wrapper {
	if a.drv == nil {
		a.Err = ErrDriverNotReady
		return a
	}

	a.query = query
	_, a.Err = a.drv.Exec(query, args...)

	return a
}

func (a *Wrapper) Each(cb func(wrapper *Wrapper) bool) *Wrapper {
	if a.Err != nil {
		return a
	}
	if a.drv == nil {
		a.Err = ErrDriverNotReady
		return a
	}

	for a.row.Next() {
		if !cb(a) {
			break
		}

		if a.Err != nil {
			return a
		}
	}

	a.row.Close()
	return a
}

func (a *Wrapper) Scan(dest ...interface{}) error {
	a.Err = a.row.Scan(dest...)
	if a.Err != nil {
		return a.Err
	}
	return nil
}

func NewWrapper(drv *sql.DB) *Wrapper {
	return &Wrapper{
		drv: drv,
	}
}
