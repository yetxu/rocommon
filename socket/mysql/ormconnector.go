package mysql

import (
	"database/sql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"rocommon"
	"rocommon/socket"
	"rocommon/util"
	"sync"
	"time"
)

type MysqlOrmConnector struct {
	MySQLParameter
	socket.NetServerNodeProperty
	db      *sql.DB
	ormDb   *gorm.DB
	dbMutex sync.RWMutex

	reconDur time.Duration
}

func (this *MysqlOrmConnector) dbConn() *sql.DB {
	this.dbMutex.RLock()
	defer this.dbMutex.RUnlock()

	return this.db
}
func (this *MysqlOrmConnector) DbConnORM() *gorm.DB {
	this.dbMutex.RLock()
	defer this.dbMutex.RUnlock()

	return this.ormDb
}
func (this *MysqlOrmConnector) IsReady() bool {
	return this.dbConn() != nil
}

func (this *MysqlOrmConnector) Operate(cb func(client interface{}) interface{}) interface{} {
	return cb(this.dbConn())
}

func (this *MysqlOrmConnector) TypeOfName() string {
	return "MysqlOrmConnector"
}

func (this *MysqlOrmConnector) SetReconnectDuration(v time.Duration) {
	this.reconDur = v
}
func (this *MysqlOrmConnector) tryConnect() {
	tmpOrmDb, err := gorm.Open(mysql.Open(this.GetAddr()), &gorm.Config{})
	//_, err := mysql.ParseDSN(this.GetAddr())
	if err != nil {
		util.ErrorF("invalid mysql dns=%v err=%v", this.GetAddr(), err)
		return
	}

	//util.InfoF("connect to mysql name=%v addr=%v dbname=%v", this.GetName(), cfg.Addr, cfg.DBName)
	this.ormDb = tmpOrmDb
	tmpDb, err := this.ormDb.DB()
	if err != nil {
		util.ErrorF("invalid mysql DB() err=%v", this.GetAddr(), err)
		return
	}

	err = tmpDb.Ping()
	if err != nil {
		util.ErrorF("ping err=%v", err)
		return
	}

	tmpDb.SetMaxIdleConns(int(this.PoolConnCount))
	tmpDb.SetMaxIdleConns(int(this.PoolConnCount))

	this.dbMutex.Lock()
	this.db = tmpDb
	this.dbMutex.Unlock()
}

func (this *MysqlOrmConnector) Start() rocommon.ServerNode {
	for {
		this.tryConnect()
		if this.reconDur == 0 || this.IsReady() {
			break
		}
		time.Sleep(this.reconDur)
	}
	return this
}

func (this *MysqlOrmConnector) Stop() {
	db := this.dbConn()
	if db != nil {
		db.Close()
	}
}

func init() {
	socket.RegisterServerNode(func() rocommon.ServerNode {
		node := new(MysqlOrmConnector)
		node.MySQLParameter.Init()
		return node
	})

}
