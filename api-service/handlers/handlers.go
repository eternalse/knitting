package handlers

import (
	"database/sql"
	"encoding/json"
	"knittibot/api-service/db"
	"knittibot/api-service/models"
	"knittibot/api-service/repository"
	"net/http"
	"strconv"

	"github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
)

// Инициализация глобальной переменной базы данных
func SetDB(database *sql.DB) {
	db.DB = database
}

func CreateIdeaHandler(w http.ResponseWriter, r *http.Request) {
	var request models.AddIdeaRequest

	// Декодирование JSON-запроса
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		logrus.Errorf("Error decoding request: %v", err)
		return
	}

	// Логируем декодированные данные для отладки
	logrus.Infof("Decoded request: %+v", request)

	// Проверка данных перед добавлением в базу
	if request.NumberOfBalls < 1 || request.NumberOfBalls > 20 {
		http.Error(w, "Invalid number_of_balls", http.StatusBadRequest)
		logrus.Warn("Invalid number_of_balls provided")
		return
	}

	if request.NumberOfColors < 1 || request.NumberOfColors > 20 {
		http.Error(w, "Invalid number_of_colors", http.StatusBadRequest)
		logrus.Warn("Invalid number_of_colors provided")
		return
	}

	// Проверка на нулевое значение базы данных
	if db.DB == nil {
		http.Error(w, "Database connection is not initialized", http.StatusInternalServerError)
		logrus.Error("Database connection is nil")
		return
	}

	// Добавление новой идеи в базу данных
	if err := repository.AddIdea(db.DB, request); err != nil {
		http.Error(w, "Failed to add idea", http.StatusInternalServerError)
		logrus.Errorf("Error adding idea to database: %v", err)
		return
	}

	// Отправка успешного ответа
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Idea successfully added"})
	logrus.Infof("Idea successfully added: %s", request.Title)
}

// GetIdeasHandler обрабатывает запросы на получение списка идей
func GetIdeasHandler(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Query().Get("title") // Получаем параметр названия из запроса

	var ideas []models.Idea
	var err error

	if title != "" {
		// Получаем идеи по названию
		ideas, err = repository.GetIdeasByTitle(db.DB, title)
		logrus.Infof("Searching ideas by title: %s", title)
	} else {
		// Получаем все идеи
		ideas, err = repository.GetAllIdeas(db.DB)
		logrus.Info("Retrieving all ideas")
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		logrus.Errorf("Error retrieving ideas: %v", err)
		return
	}

	// Отправка успешного ответа
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ideas)
	logrus.Infof("Retrieved %d ideas", len(ideas))
}

// GetIdeaHandler обрабатывает запросы на получение конкретной идеи по ID
func GetIdeaHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]

	// Получаем идею по ID
	idea, err := repository.GetIdeaByID(db.DB, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		logrus.Warnf("Idea with ID %s not found: %v", id, err)
		return
	}

	// Отправка успешного ответа
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(idea)
	logrus.Infof("Retrieved idea: %+v", idea)
}

// DeleteIdeaHandler обрабатывает удаление идеи по ID
func DeleteIdeaHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	idStr := params["id"]

	// Преобразуем строку в int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "неверный ID", http.StatusBadRequest)
		logrus.Warnf("Invalid ID format: %s", idStr)
		return
	}

	// Удаление идеи по ID
	err = repository.DeleteIdeaByID(db.DB, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		logrus.Warnf("Failed to delete idea with ID %d: %v", id, err)
		return
	}

	// Отправка успешного ответа
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Idea successfully deleted"})
	logrus.Infof("Deleted idea with ID %d", id)
}

// SearchIdeasHandler обрабатывает запросы на поиск идей по параметрам
func SearchIdeasHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры запроса
	typeOfItem := r.URL.Query().Get("type_of_item")
	numberOfBalls := r.URL.Query().Get("number_of_balls")
	numberOfColors := r.URL.Query().Get("number_of_colors")
	toolType := r.URL.Query().Get("tool_type")
	yarnType := r.URL.Query().Get("yarn_type")

	// Преобразование параметров в нужные типы
	numBalls, err := strconv.Atoi(numberOfBalls)
	if err != nil {
		numBalls = 0 // если ошибка преобразования, устанавливаем значение по умолчанию
	}

	numColors, err := strconv.Atoi(numberOfColors)
	if err != nil {
		numColors = 0
	}

	// Проверка на нулевое значение базы данных
	if db.DB == nil {
		http.Error(w, "Database connection is not initialized", http.StatusInternalServerError)
		logrus.Error("Database connection is nil")
		return
	}

	// Поиск идей в базе данных
	ideas, err := repository.FindIdeas(db.DB, typeOfItem, numBalls, numColors, toolType, yarnType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logrus.Errorf("Error searching ideas: %v", err)
		return
	}

	// Отправка успешного ответа
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ideas)
	logrus.Infof("Found %d ideas based on search criteria", len(ideas))
}

// RegisterRoutes регистрирует все маршруты для API
func RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/ideas", CreateIdeaHandler).Methods(http.MethodPost)
	r.HandleFunc("/ideas", GetIdeasHandler).Methods(http.MethodGet)
	r.HandleFunc("/ideas/{id:[0-9]+}", GetIdeaHandler).Methods(http.MethodGet)
	r.HandleFunc("/ideas/{id:[0-9]+}", DeleteIdeaHandler).Methods(http.MethodDelete)
	r.HandleFunc("/search-ideas", SearchIdeasHandler).Methods(http.MethodGet)
}
