package mysql

import (
	"database/sql"
	"sync"
	"time"

	"github.com/yetxu/rocommon"
	"github.com/yetxu/rocommon/socket"
	"github.com/yetxu/rocommon/util"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type MysqlOrmConnector struct {
	MySQLParameter
	socket.NetServerNodeProperty
	db      *sql.DB
	ormDb   *gorm.DB
	dbMutex sync.RWMutex

	reconDur time.Duration
}

func (a *MysqlOrmConnector) dbConn() *sql.DB {
	a.dbMutex.RLock()
	defer a.dbMutex.RUnlock()

	return a.db
}
func (a *MysqlOrmConnector) DbConnORM() *gorm.DB {
	a.dbMutex.RLock()
	defer a.dbMutex.RUnlock()

	return a.ormDb
}
func (a *MysqlOrmConnector) IsReady() bool {
	return a.dbConn() != nil
}

func (a *MysqlOrmConnector) Operate(cb func(client interface{}) interface{}) interface{} {
	return cb(a.dbConn())
}

func (a *MysqlOrmConnector) TypeOfName() string {
	return "MysqlOrmConnector"
}

func (a *MysqlOrmConnector) SetReconnectDuration(v time.Duration) {
	a.reconDur = v
}
func (a *MysqlOrmConnector) tryConnect() {
	tmpOrmDb, err := gorm.Open(mysql.Open(a.GetAddr()), &gorm.Config{})
	//_, err := mysql.ParseDSN(a.GetAddr())
	if err != nil {
		util.ErrorF("invalid mysql dns=%v err=%v", a.GetAddr(), err)
		return
	}

	//util.InfoF("connect to mysql name=%v addr=%v dbname=%v", a.GetName(), cfg.Addr, cfg.DBName)
	a.ormDb = tmpOrmDb
	tmpDb, err := a.ormDb.DB()
	if err != nil {
		util.ErrorF("invalid mysql DB() err=%v", a.GetAddr(), err)
		return
	}

	err = tmpDb.Ping()
	if err != nil {
		util.ErrorF("ping err=%v", err)
		return
	}

	tmpDb.SetMaxIdleConns(int(a.PoolConnCount))
	tmpDb.SetMaxIdleConns(int(a.PoolConnCount))

	a.dbMutex.Lock()
	a.db = tmpDb
	a.dbMutex.Unlock()
}

func (a *MysqlOrmConnector) Start() rocommon.ServerNode {
	for {
		a.tryConnect()
		if a.reconDur == 0 || a.IsReady() {
			break
		}
		time.Sleep(a.reconDur)
	}
	return a
}

func (a *MysqlOrmConnector) Stop() {
	db := a.dbConn()
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
