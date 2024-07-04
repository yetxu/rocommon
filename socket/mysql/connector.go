package mysql

import (
	"database/sql"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
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

func (this *MysqlConnector) dbConn() *sql.DB {
	this.dbMutex.RLock()
	defer this.dbMutex.RUnlock()

	return this.db
}

func (this *MysqlConnector) IsReady() bool {
	return this.dbConn() != nil
}

func (this *MysqlConnector) Operate(cb func(client interface{}) interface{}) interface{} {
	return cb(this.dbConn())
}

func (this *MysqlConnector) TypeOfName() string {
	return "mysqlConnector"
}

func (this *MysqlConnector) SetReconnectDuration(v time.Duration) {
	this.reconDur = v
}

func (this *MysqlConnector) tryConnect() {
	_, err := mysql.ParseDSN(this.GetAddr())
	if err != nil {
		util.ErrorF("invalid mysql dns=%v err=%v", this.GetAddr(), err)
		return
	}
	//util.InfoF("connect to mysql name=%v addr=%v dbname=%v", this.GetName(), cfg.Addr, cfg.DBName)

	db, err := sql.Open("mysql", this.GetAddr())
	if err != nil {
		util.ErrorF("open mysql database err=%v", err)
		return
	}

	err = db.Ping()
	if err != nil {
		util.ErrorF("ping err=%v", err)
		return
	}

	db.SetMaxIdleConns(int(this.PoolConnCount))
	db.SetMaxIdleConns(int(this.PoolConnCount))

	this.dbMutex.Lock()
	this.db = db
	this.dbMutex.Unlock()
}
func (this *MysqlConnector) Start() rocommon.ServerNode {
	for {
		this.tryConnect()
		if this.reconDur == 0 || this.IsReady() {
			break
		}
		time.Sleep(this.reconDur)
	}
	return this
}

func (this *MysqlConnector) Stop() {
	db := this.dbConn()
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
