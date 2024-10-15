package repository

import (
	"database/sql"
	"fmt"
	"knittibot/api-service/models"

	_ "github.com/lib/pq" // импортируем драйвер PostgreSQL
	"github.com/sirupsen/logrus"
)

// Пример инициализации базы данных
func InitDB(dataSourceName string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		logrus.Errorf("Error opening database: %v", err)
		return nil, err
	}
	logrus.Info("Database connection established")
	return db, nil
}

// Добавление новой идеи
func AddIdea(db *sql.DB, idea models.AddIdeaRequest) error {
	if db == nil {
		return fmt.Errorf("database connection is nil")
	}
	_, err := db.Exec(`
		INSERT INTO ideas (title, type_of_item, number_of_balls, number_of_colors, tool_type, yarn_type, scheme_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		idea.Title, idea.TypeOfItem, idea.NumberOfBalls, idea.NumberOfColors, idea.ToolType, idea.YarnType, idea.SchemeURL,
	)
	if err != nil {
		logrus.Errorf("Error adding idea: %v", err)
		return err
	}
	logrus.Infof("Idea added: %s", idea.Title)
	return nil
}

// Получение всех идей
func GetAllIdeas(db *sql.DB) ([]models.Idea, error) {
	rows, err := db.Query("SELECT id, title, type_of_item, number_of_balls, number_of_colors, tool_type, yarn_type, scheme_url FROM ideas")
	if err != nil {
		logrus.Errorf("Error retrieving all ideas: %v", err)
		return nil, err
	}
	defer rows.Close()

	var ideas []models.Idea
	for rows.Next() {
		var idea models.Idea
		if err := rows.Scan(&idea.ID, &idea.Title, &idea.TypeOfItem, &idea.NumberOfBalls, &idea.NumberOfColors, &idea.ToolType, &idea.YarnType, &idea.SchemeURL); err != nil {
			logrus.Errorf("Error scanning row: %v", err)
			return nil, err
		}
		ideas = append(ideas, idea)
	}
	logrus.Infof("Retrieved %d ideas", len(ideas))
	return ideas, nil
}

// Получение идеи по ID
func GetIdeaByID(db *sql.DB, id string) (models.Idea, error) {
	var idea models.Idea
	row := db.QueryRow("SELECT id, title, type_of_item, number_of_balls, number_of_colors, tool_type, yarn_type, scheme_url FROM ideas WHERE id = $1", id)
	err := row.Scan(&idea.ID, &idea.Title, &idea.TypeOfItem, &idea.NumberOfBalls, &idea.NumberOfColors, &idea.ToolType, &idea.YarnType, &idea.SchemeURL)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.Warnf("Idea with ID %s not found", id)
			return idea, fmt.Errorf("Idea with ID %s not found", id)
		}
		logrus.Errorf("Error retrieving idea by ID %s: %v", id, err)
		return idea, err
	}
	logrus.Infof("Retrieved idea: %s", idea.Title)
	return idea, nil
}

// Удаление идеи по ID
func DeleteIdeaByID(db *sql.DB, id int) error {
	result, err := db.Exec("DELETE FROM ideas WHERE id = $1", id)
	if err != nil {
		logrus.Errorf("Error deleting idea with ID %d: %v", id, err)
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		logrus.Errorf("Error retrieving rows affected for ID %d: %v", id, err)
		return err
	}
	if rowsAffected == 0 {
		logrus.Warnf("Idea with ID %d not found for deletion", id)
		return fmt.Errorf("Idea with ID %d not found", id)
	}
	logrus.Infof("Deleted idea with ID %d", id)
	return nil
}

// Получение идей по названию
func GetIdeasByTitle(db *sql.DB, title string) ([]models.Idea, error) {
	rows, err := db.Query("SELECT id, title, type_of_item, number_of_balls, number_of_colors, tool_type, yarn_type, scheme_url FROM ideas WHERE title ILIKE $1", "%"+title+"%")
	if err != nil {
		logrus.Errorf("Error retrieving ideas by title %s: %v", title, err)
		return nil, err
	}
	defer rows.Close()

	var ideas []models.Idea
	for rows.Next() {
		var idea models.Idea
		if err := rows.Scan(&idea.ID, &idea.Title, &idea.TypeOfItem, &idea.NumberOfBalls, &idea.NumberOfColors, &idea.ToolType, &idea.YarnType, &idea.SchemeURL); err != nil {
			logrus.Errorf("Error scanning row: %v", err)
			return nil, err
		}
		ideas = append(ideas, idea)
	}
	logrus.Infof("Retrieved %d ideas with title containing '%s'", len(ideas), title)
	return ideas, nil
}

// FindIdeas ищет идеи в базе данных на основе заданных параметров
func FindIdeas(db *sql.DB, typeOfItem string, numberOfBalls int, numberOfColors int, toolType string, yarnType string) ([]models.Idea, error) {
	// Формируем базовый SQL-запрос
	query := `SELECT id, title, type_of_item, number_of_balls, number_of_colors, tool_type, yarn_type, scheme_url
              FROM ideas
              WHERE (type_of_item = $1 OR $1 = 'любой')
              AND (number_of_balls = $2 OR $2 = 0)
              AND (number_of_colors = $3 OR $3 = 0)
              AND (tool_type = $4 OR $4 = 'любой')
              AND (yarn_type = $5 OR $5 = 'любой')`

	// Выполняем запрос
	rows, err := db.Query(query, typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)
	if err != nil {
		logrus.Errorf("Error executing find ideas query: %v", err)
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var ideas []models.Idea
	for rows.Next() {
		var idea models.Idea
		if err := rows.Scan(&idea.ID, &idea.Title, &idea.TypeOfItem, &idea.NumberOfBalls, &idea.NumberOfColors, &idea.ToolType, &idea.YarnType, &idea.SchemeURL); err != nil {
			logrus.Errorf("Error scanning row: %v", err)
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		ideas = append(ideas, idea)
	}

	if err := rows.Err(); err != nil {
		logrus.Errorf("Error iterating rows: %v", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	logrus.Infof("Found %d ideas based on search criteria", len(ideas))
	return ideas, nil
}
