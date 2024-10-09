package main

import (
	"database/sql"
	"fmt"
	"knitti/bot-service/config"
	"knitti/bot-service/handlers"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	_ "github.com/lib/pq" // Импортируйте драйвер для PostgreSQL
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		logger.Fatalf("Error loading config: %v", err)
	}

	// Получаем токен бота и строку подключения к базе данных из конфигурации
	botToken := cfg.Telegram.Token

	// Формируем строку подключения к базе данных
	dbConnStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	// Создание подключения к базе данных
	db, err := sql.Open("postgres", dbConnStr)
	if err != nil {
		logger.Fatalf("Error opening database: %v", err)
	}
	defer db.Close()

	// Создание нового бота
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Infof("Authorized on account %s", bot.Self.UserName)

	// Включаем логирование
	bot.Debug = true

	// Создаем канал для обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		logger.Panic(err)
	}

	// Обрабатываем входящие сообщения
	for update := range updates {
		if update.Message == nil {
			continue
		}

		logger.Infof("Received message: %s from user %s", update.Message.Text, update.Message.From.UserName)

		// Проверяем на команды
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				handlers.HandleStart(bot, update.Message)
			case "заново":
				handlers.HandleRedoRequest(bot, update.Message, db)
			case "delete":
				handlers.HandleDeleteIdeaRequest(bot, update.Message, db)

			}
		} else {
			// Если это не команда, обрабатываем обычное сообщение
			handlers.HandleMessage(bot, update.Message, db)
		}

	}
}
