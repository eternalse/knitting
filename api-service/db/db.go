package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

// Глобальная переменная для базы данных
var DB *sql.DB

// Инициализация базы данных
func InitDB(dataSourceName string) {
	var err error
	DB, err = sql.Open("postgres", dataSourceName)
	if err != nil {
		log.Fatalf("Could not connect to database: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Could not ping database: %v", err)
	}
}
