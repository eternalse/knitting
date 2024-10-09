package main

import (
	"database/sql"
	"fmt"
	"knitti/api-service/config"
	"knitti/api-service/handlers"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

var db *sql.DB
var cfg *config.Config

func main() {
	var err error

	// Загрузка конфигурации
	cfg, err = config.LoadConfig("config.yaml")
	if err != nil {
		logrus.Fatalf("Error loading configuration: %v", err)
	}

	// Проверка конфигурации
	if cfg.Server.Port == "" {
		logrus.Fatal("Server port not specified in configuration.")
	}
	if cfg.Database.Host == "" || cfg.Database.Port == "" || cfg.Database.User == "" ||
		cfg.Database.Password == "" || cfg.Database.DBName == "" {
		logrus.Fatal("Database configuration is incomplete.")
	}

	// Подключение к базе данных
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		logrus.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Проверка подключения
	if err := db.Ping(); err != nil {
		logrus.Fatalf("Error pinging database: %v", err)
	}
	logrus.Info("Successfully connected to the database")

	// Инициализация обработчиков
	handlers.SetDB(db)

	// Создаем новый роутер
	r := mux.NewRouter()

	// Регистрируем маршруты
	handlers.RegisterRoutes(r)

	// Печать зарегистрированных маршрутов
	printRoutes(r)

	// Запускаем HTTP-сервер
	serverAddr := fmt.Sprintf(":%s", cfg.Server.Port)
	logrus.Infof("Starting API service on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, r); err != nil {
		logrus.Fatalf("Error starting server: %v", err)
	}
}

// printRoutes выводит все зарегистрированные маршруты для отладки
func printRoutes(r *mux.Router) {
	_ = r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, _ := route.GetPathTemplate()
		methods, _ := route.GetMethods()
		logrus.Infof("Path: %s, Methods: %v", pathTemplate, methods)
		return nil
	})
}
