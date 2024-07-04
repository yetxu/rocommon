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

func (this *Wrapper) Query(query string, args ...interface{}) *Wrapper {
	if this.drv == nil {
		this.Err = ErrDriverNotReady
		return this
	}

	this.query = query
	this.row, this.Err = this.drv.Query(query, args...)

	//log.Println("rows=", this.row)

	return this
}

func (this *Wrapper) Execute(query string, args ...interface{}) *Wrapper {
	if this.drv == nil {
		this.Err = ErrDriverNotReady
		return this
	}

	this.query = query
	_, this.Err = this.drv.Exec(query, args...)

	return this
}

func (this *Wrapper) Each(cb func(wrapper *Wrapper) bool) *Wrapper {
	if this.Err != nil {
		return this
	}
	if this.drv == nil {
		this.Err = ErrDriverNotReady
		return this
	}

	for this.row.Next() {
		if !cb(this) {
			break
		}

		if this.Err != nil {
			return this
		}
	}

	this.row.Close()
	return this
}

func (this *Wrapper) Scan(dest ...interface{}) error {
	this.Err = this.row.Scan(dest...)
	if this.Err != nil {
		return this.Err
	}
	return nil
}

func NewWrapper(drv *sql.DB) *Wrapper {
	return &Wrapper{
		drv: drv,
	}
}
