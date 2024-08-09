package mysql

import (
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/yetxu/rocommon/util"
)

func NewSqlTxConn() (conn *sqlx.DB, tx *sql.Tx, err error) {
	conn = GMysqlPool.GetConn()

	tx, err = conn.Begin()

	return
}

func CloseSqlTxConn(conn *sqlx.DB, tx *sql.Tx, err *error) {
	GMysqlPool.UnGetConn(conn)

	SqlTxProc(tx, err)
}

func SqlTxProc(sqltx *sql.Tx, perr *error) {
	var err error
	if nil != *perr {
		err = sqltx.Rollback()
		if nil != err {
			*perr = util.NewError("sqltx.Rollback err %v *perr %v", err, *perr)
		}
	} else {
		err = sqltx.Commit()
		if nil != err {
			*perr = util.NewError("sqltx.Commit err %v", err)
		}
	}
}

func SqlTxExec(sqltx *sql.Tx, sqlstr string) (err error) {
	_, err = sqltx.Exec(sqlstr)
	if nil != err {
		err = util.NewError("%v sql[%s]", err, sqlstr)
		return
	}
	return
}

func SqlTxExecf(sqltx *sql.Tx, format string, a ...interface{}) (err error) {
	sqlstr := fmt.Sprintf(format, a...)
	_, err = sqltx.Exec(sqlstr)
	if nil != err {
		err = util.NewError("%v sql[%s]", err, sqlstr)
		return
	}
	return
}
