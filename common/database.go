package common

import(
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"database/sql"
)
var DB *bun.DB
func GetDB() *bun.DB{
	if DB == nil {
		config := GetConfig()
		DB = bun.NewDB(sql.OpenDB(pgdriver.NewConnector(
			pgdriver.WithAddr(config.DBIP),
			pgdriver.WithUser(config.DBUSER),
			pgdriver.WithPassword(config.DBPASSWORD),
			pgdriver.WithDatabase(config.DBNAME),
			pgdriver.WithTLSConfig(nil),
		)), pgdialect.New())
	}
	return DB
}
