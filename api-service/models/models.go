package models

// Idea представляет собой структуру идеи для вязания
type Idea struct {
	ID             int    `json:"id,omitempty"` // Уникальный идентификатор идеи
	Title          string `json:"title"`
	TypeOfItem     string `json:"type_of_item"`         // Тип изделия
	NumberOfBalls  int    `json:"number_of_balls"`      // Количество мотков
	NumberOfColors int    `json:"number_of_colors"`     // Количество цветов
	ToolType       string `json:"tool_type"`            // Крючок или Спицы
	YarnType       string `json:"yarn_type"`            // Тип пряжи
	SchemeURL      string `json:"scheme_url,omitempty"` // Ссылка на схему (опционально)
}

// AddIdeaRequest представляет собой структуру запроса для добавления новой идеи
type AddIdeaRequest struct {
	Title          string `json:"title"`
	TypeOfItem     string `json:"type_of_item"`         // Тип изделия
	NumberOfBalls  int    `json:"number_of_balls"`      // Количество мотков
	NumberOfColors int    `json:"number_of_colors"`     // Количество цветов
	ToolType       string `json:"tool_type"`            // Крючок или Спицы
	YarnType       string `json:"yarn_type"`            // Тип пряжи
	SchemeURL      string `json:"scheme_url,omitempty"` // Ссылка на схему (опционально)
}
