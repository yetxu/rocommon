package mysql

type MySQLParameter struct {
	PoolConnCount int
}

func (this *MySQLParameter) Init() {
	this.PoolConnCount = 32
}

func (this *MySQLParameter) SetPassword(v string) {
}

func (this *MySQLParameter) SetConnCount(val int) {
	this.PoolConnCount = val
}
