package mysql

import (
	"database/sql"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"
)

type MysqlConnector struct {
	MySQLParameter
	socket.NetServerNodeProperty
	db      *sql.DB
	dbMutex sync.RWMutex

	reconDur time.Duration
}

func (mysqltor *MysqlConnector) dbConn() *sql.DB {
	mysqltor.dbMutex.RLock()
	defer mysqltor.dbMutex.RUnlock()

	return mysqltor.db
}

func (mysqltor *MysqlConnector) IsReady() bool {
	return mysqltor.dbConn() != nil
}

func (mysqltor *MysqlConnector) Operate(cb func(client interface{}) interface{}) interface{} {
	return cb(mysqltor.dbConn())
}

func (mysqltor *MysqlConnector) TypeOfName() string {
	return "mysqlConnector"
}

func (mysqltor *MysqlConnector) SetReconnectDuration(v time.Duration) {
	mysqltor.reconDur = v
}

func (mysqltor *MysqlConnector) tryConnect() {
	_, err := mysql.ParseDSN(mysqltor.GetAddr())
	if err != nil {
		util.ErrorF("invalid mysql dns=%v err=%v", mysqltor.GetAddr(), err)
		return
	}
	//util.InfoF("connect to mysql name=%v addr=%v dbname=%v", a.GetName(), cfg.Addr, cfg.DBName)

	db, err := sql.Open("mysql", mysqltor.GetAddr())
	if err != nil {
		util.ErrorF("open mysql database err=%v", err)
		return
	}

	err = db.Ping()
	if err != nil {
		util.ErrorF("ping err=%v", err)
		return
	}

	db.SetMaxIdleConns(mysqltor.PoolConnCount)
	//db.SetMaxIdleConns(int(a.PoolConnCount))
	dbx := sqlx.NewDb(db, "mysql")
	dbx.MapperFunc(util.LowerCaseWithUnderscores)

	mysqltor.dbMutex.Lock()
	mysqltor.db = db

	pool := &MysqlPool{db, dbx}
	if GMysqlPool == nil {
		GMysqlPool = pool
	}
	mysqltor.dbMutex.Unlock()
}
func (mysqltor *MysqlConnector) Start() rocommon.ServerNode {
	for {
		mysqltor.tryConnect()
		if mysqltor.reconDur == 0 || mysqltor.IsReady() {
			break
		}
		time.Sleep(mysqltor.reconDur)
	}
	return mysqltor
}

func (mysqltor *MysqlConnector) Stop() {
	db := mysqltor.dbConn()
	if db != nil {
		db.Close()
	}
}

func init() {
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := new(MysqlConnector)
		node.MySQLParameter.Init()
		return node
	})

}
