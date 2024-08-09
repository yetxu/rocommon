package mysql

type MySQLParameter struct {
	PoolConnCount int
}

func (a *MySQLParameter) Init() {
	a.PoolConnCount = 32
}

func (a *MySQLParameter) SetPassword(v string) {
}

func (a *MySQLParameter) SetConnCount(val int) {
	a.PoolConnCount = val
}
