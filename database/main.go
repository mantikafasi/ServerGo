package database

import (
	"database/sql"
	"log"
	"runtime"

	"server-go/common"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

var DB *bun.DB

func InitDB() {
	config := common.Config
	DB = bun.NewDB(sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithAddr(config.DB.IP),
		pgdriver.WithUser(config.DB.User),
		pgdriver.WithPassword(config.DB.Password),
		pgdriver.WithDatabase(config.DB.Name),
		pgdriver.WithTLSConfig(nil),
	)), pgdialect.New())

	maxOpenConns := 4 * runtime.GOMAXPROCS(0)
	DB.SetMaxOpenConns(maxOpenConns)
	DB.SetMaxIdleConns(maxOpenConns)

	// create database structure if doesn't exist
	if err := createSchema(); err != nil {
		log.Println("Failed to create schema")
		log.Panic(err)
	}
}
