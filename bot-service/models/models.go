package models

import (
	"database/sql"
	"fmt"
	"knittibot/api-service/models"

	"github.com/sirupsen/logrus"
)

// Idea представляет собой структуру для хранения информации о существующей идее
type Idea struct {
	Title          string `json:"title"`            // Название идеи
	TypeOfItem     string `json:"type_of_item"`     // Тип изделия
	NumberOfBalls  int    `json:"number_of_balls"`  // Количество мотков
	NumberOfColors int    `json:"number_of_colors"` // Количество цветов
	ToolType       string `json:"tool_type"`        // Крючок или Спицы
	YarnType       string `json:"yarn_type"`        // Тип пряжи
	SchemeURL      string `json:"scheme_url"`       // Ссылка на схему
}

// IdeaRequest представляет собой структуру для запроса информации о идее
type IdeaRequest struct {
	Title          string `json:"title"`            // Название идеи
	TypeOfItem     string `json:"type_of_item"`     // Тип изделия
	NumberOfBalls  int    `json:"number_of_balls"`  // Количество мотков
	NumberOfColors int    `json:"number_of_colors"` // Количество цветов
	ToolType       string `json:"tool_type"`        // Крючок или Спицы
	YarnType       string `json:"yarn_type"`        // Тип пряжи
}

// AddIdeaRequest представляет собой структуру для обработки запроса на добавление новой идеи
type AddIdeaRequest struct {
	Title          string `json:"title"`                // Название идеи
	TypeOfItem     string `json:"type_of_item"`         // Тип изделия
	NumberOfBalls  int    `json:"number_of_balls"`      // Количество мотков
	NumberOfColors int    `json:"number_of_colors"`     // Количество цветов
	ToolType       string `json:"tool_type"`            // Крючок или Спицы
	YarnType       string `json:"yarn_type"`            // Тип пряжи
	SchemeURL      string `json:"scheme_url,omitempty"` // Ссылка на схему (необязательно)
}

// findIdeas ищет идеи в базе данных на основе заданных параметров
func FindIdeas(db *sql.DB, typeOfItem string, numberOfBalls int, numberOfColors int, toolType string, yarnType string) ([]models.Idea, error) {
	// Логируем входные параметры
	logrus.Infof("FindIdeas called with typeOfItem=%s, numberOfBalls=%d, numberOfColors=%d, toolType=%s, yarnType=%s",
		typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)

	query := `SELECT title, type_of_item, number_of_balls, number_of_colors, tool_type, yarn_type, scheme_url FROM ideas WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if typeOfItem != "любой" {
		query += fmt.Sprintf(" AND type_of_item = $%d", argIdx)
		args = append(args, typeOfItem)
		argIdx++
	}
	if numberOfBalls > 0 {
		query += fmt.Sprintf(" AND number_of_balls = $%d", argIdx)
		args = append(args, numberOfBalls)
		argIdx++
	}
	if numberOfColors > 0 {
		query += fmt.Sprintf(" AND number_of_colors = $%d", argIdx)
		args = append(args, numberOfColors)
		argIdx++
	}
	if toolType != "любой" {
		query += fmt.Sprintf(" AND tool_type = $%d", argIdx)
		args = append(args, toolType)
		argIdx++
	}
	if yarnType != "любой" {
		query += fmt.Sprintf(" AND yarn_type = $%d", argIdx)
		args = append(args, yarnType)
	}

	// Логируем финальный запрос и параметры
	logrus.Debugf("Executing query: %s with args: %v", query, args)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	var ideas []models.Idea
	for rows.Next() {
		var idea models.Idea
		if err := rows.Scan(&idea.Title, &idea.TypeOfItem, &idea.NumberOfBalls, &idea.NumberOfColors, &idea.ToolType, &idea.YarnType, &idea.SchemeURL); err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		ideas = append(ideas, idea)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Логируем количество найденных идей
	logrus.Infof("FindIdeas found %d ideas", len(ideas))

	return ideas, nil
}
