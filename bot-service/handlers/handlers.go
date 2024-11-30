package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"knittibot/api-service/repository"
	"knittibot/bot-service/models"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
)

// Объявляю логер
var logger = logrus.New()

// Объявляем ID группового чата (можно также использовать переменные окружения)
var groupChatID int64 = -1002433

// HandleStart обрабатывает команду /start
func HandleStart(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	msgText := "*👋 Привет! Меня зовут Knitty,* я бот для поиска идей для вязания. Я могу:\n\n" +
		"❤️ Предложить тебе новую идею для вязания ❤️\n\n" +
		"  - Для этого нажми кнопку *`Новая идея`* в меню. Я отправлю тебе шаблон сообщения, воспользуйся им, чтобы ввести данные корректно.\n\n" +
		"❤️ Добавить твою собственную идею, чтобы другие тоже могли ей воспользоваться ❤️\n\n" +
		"  - Для этого используй кнопку *`Добавить свою идею`* в меню. Я отправлю тебе шаблон сообщения, воспользуйся им, чтобы ввести данные корректно.\n\n" +
		"❤️ Предоставить тебе инструкции, если возникли сложности ❤️\n\n" +
		"  - Для этого нажми кнопку *`Помощь`* в меню.\n\n" +
		"❤️ *Разработчик старается сделать меня лучше!* Если ты столкнулся с какой-то проблемой или ошибкой, отправь сообщение с жалобой, и она обязательно все починит! ❤️ \n\n" +
		"  -К сожалению, практически невозможно отфильтровать весь контент, который добавляют другие пользователи. Если ты столкнулся с чем-то неприемлемого содержания, пожалуйста, отправь жалобу.\n\n" +
		"  -Для отправки жалобы, перешли сообщение, на которое хочешь пожаловаться, с подписью *`Жалоба: текст вашей жалобы`*, либо просто напиши в чат *`Жалоба: текст вашей жалобы`*.\n\n" +
		" ❗❗❗    Всегда помните про безопасность! Если какая-то ссылка кажется вам подозрительной - не переходите по ней. Как распознать опасную ссылку - читайте в разделе *Помощь*    ❗❗❗  \n\n" +
		"*Если готовы приступить, то нажмите на кнопку \"Начать!\".*"

	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Начать!"),
		),
	)
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, msgText)
	msgConfig.ParseMode = "Markdown" // режим разметки
	msgConfig.ReplyMarkup = keyboard

	if _, err := bot.Send(msgConfig); err != nil {
		logger.WithError(err).Error("Error sending start message")
	}
}

func HandleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, db *sql.DB) {
	switch msg.Text {
	case "Начать!":
		HandleMenu(bot, msg)
	case "Новая идея":
		HandleNewIdeaRequest(bot, msg)
	case "Добавить свою идею":
		HandleAddNewIdeaRequest(bot, msg)
	case "Заново":
		HandleRedoRequest(bot, msg, db)
	case "Жалоба":
		HandleComplaintRequest(bot, msg)
	case "Помощь":
		HandleHelpRequest(bot, msg)
		/*case "Поддержать разработчика":
		HandleSupportRequest(bot, msg) */
	default:
		if strings.HasPrefix(msg.Text, "Жалоба:") {
			forwardComplaintToGroup(bot, msg) // Пересылаем жалобу в групповой чат
		} else if strings.HasPrefix(msg.Text, "/delete ") {
			handleDeleteCommand(bot, msg, db)
		} else {
			// Проверяем формат на добавление идеи
			if isAddIdeaFormat(msg.Text) {
				HandleProcessAddIdeaRequest(bot, msg)
			} else if isSearchIdeaFormat(msg.Text) { // Проверяем на поиск идеи
				HandleProcessIdeaRequest(bot, msg, db) // Передача db
			} else {
				// Если не подходит ни под одну категорию, отправляем сообщение об ошибке
				if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "К сожалению я не знаю такой команды( Возможно в вашем сообщении допущена ошибка, либо использован неправильный формат?")); err != nil {
					logrus.WithError(err).Warn("Error sending invalid format message")
				}
			}
		}
	}
}

// handleDeleteCommand обрабатывает команду удаления идеи
func handleDeleteCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, db *sql.DB) {
	idStr := strings.TrimSpace(msg.Text[8:]) // Извлекаем ID после команды

	// Проверяем, что команда содержит ID
	if idStr == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка: Пожалуйста, укажите ID идеи после команды /delete."))
		return
	}

	// Преобразуем ID в int
	id, err := strconv.Atoi(idStr)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Неверный формат ID. Пожалуйста, используйте целое число."))
		return
	}

	// Удаляем идею
	err = repository.DeleteIdeaByID(db, id)
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Не удалось удалить идею с ID %d: %v", id, err)))
		return
	}

	// Сообщение об успешном удалении
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Идея с ID %d успешно удалена.", id)))
}

// isSearchIdeaFormat проверяет, является ли сообщение запросом на поиск идей
func isSearchIdeaFormat(text string) bool {
	// Приводим текст к нижнему регистру и удаляем пробелы по краям
	text = strings.ToLower(strings.TrimSpace(text))

	// Разделяем текст на строки
	lines := strings.Split(text, "\n")

	// Проверяем, что количество строк равно 5
	if len(lines) != 5 {
		return false
	}

	// Проверяем, что каждая строка начинается с ожидаемого формата
	for _, line := range lines {
		// Проверяем, что каждая строка имеет формат "X) текст:"
		if !strings.Contains(line, ":") {
			return false
		}
	}

	return true
}

// isAddIdeaFormat проверяет, является ли сообщение запросом на добавление идеи
func isAddIdeaFormat(text string) bool {
	// Разделяем текст по новой строке
	parts := strings.Split(text, "\n")
	return len(parts) >= 6 && len(parts) <= 7 // Проверяем количество параметров
}

var lastSearches = make(map[int64][]interface{}) // Карта для хранения параметров последнего поиска

var proposedIdeas []int // карта для хранения уже предложенных идей

// SaveLastSearch сохраняет параметры последнего поиска для пользователя
func SaveLastSearch(userID int64, typeOfItem string, numberOfBalls, numberOfColors int, toolType, yarnType string) {
	lastSearches[userID] = []interface{}{typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType}
}

// HandleMenu обрабатывает команду "Начать!"
func HandleMenu(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	menuText := "Выберите действие: "
	keyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Новая идея"),
			tgbotapi.NewKeyboardButton("Добавить свою идею"),
		),
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Заново"),
			tgbotapi.NewKeyboardButton("Помощь"),
		//	tgbotapi.NewKeyboardButton("Поддержать разработчика"),
		),
	)
	msgConfig := tgbotapi.NewMessage(msg.Chat.ID, menuText)
	msgConfig.ReplyMarkup = keyboard

	if _, err := bot.Send(msgConfig); err != nil {
		logger.WithError(err).Error("Error sending menu message")
	}
}

// HandleNewIdeaRequest обрабатывает запрос на новую идею
func HandleNewIdeaRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	instructions := "Если вам не понравилась предложенная идея, то нажмите на кнопку `Заново` в меню, и бот подберет вам что-нибудь другое \n\n" +
		"  - Используйте только те варианты данных, которые предлагаются в сообщении-подсказке, иначе могут возникнуть сложности с поиском.\n\n" +
		"  - Если вы хотите расширить поиск, выберите некоторые значения параметров произвольными. Для этого используйте значения `любой` или `0`.\n\n" +
		"  - Воспользуйтесь шаблоном в следующем сообщении, так будет проще ввести нужные параметры.\n\n" +
		"  - Чтобы я подобрал тебе идею для вязания, укажите данные:\n" +
		"1. Тип изделия: женское, мужское, детское, питомцам, аксессуар, интерьер, игрушка или любой.\n" +
		"2. Количество мотков: от 1 до 20 (если количество не важно, то укажи цифру 0)\n" +
		"3. Количество цветов: от 1 до 20 (если количество не важно, то укажи цифру 0) \n" +
		"4. Инструмент, который будете использовать: крючок, спицы, пальцы\n" +
		"5. Тип пряжи: плюшевая, обычная, пуффи, мохер или любой \n\n"

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, instructions)); err != nil {
		logger.WithError(err).Error("Error sending new idea request message")
	}
	// Отправляем шаблон для ввода данных
	templateMessage := "1) Тип изделия: \n" +
		"2) Количество мотков пряжи: \n" +
		"3) Количество цветов пряжи: \n" +
		"4) Инструмент, который будете использовать: \n" +
		"5) Тип пряжи: \n"

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, templateMessage)); err != nil {
		logger.WithError(err).Error("Error sending data input template message")
	}
}

// HandleAddNewIdeaRequest обрабатывает запрос на добавление новой идеи
func HandleAddNewIdeaRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	instructions := "  - Воспользуйтесь шаблоном в следующем сообщении, так будет проще ввести нужные параметры\n\n" +
		"  - Используйте только те варианты данных, которые предлагаются в сообщении-подсказке, иначе в дальнейшем могут возникнуть сложности с поиском этой идеи.\n\n" +
		"  - К типу пряжи `Обычная` следует относить стандартную пряжу, не зависимо от её материала и толщины. Если вы хотите добавить идею с использованием необычной пряжи (Например меховая или букле), то укажите тип `Любой` \n\n" +
		"  - Пожалуйста, не используйте нецензурную лексику в названии идей — ботом могут пользоваться люди, не достигшие 18 лет.\n\n" +
		"Добавленная вами идея станет доступна всем остальным пользователям. Для того, чтобы добавить новую идею, укажите следующее:\n" +
		"1. Название изделия.\n" +
		"2. Тип изделия: женское, мужское, детское, питомцам, аксессуар, интерьер, игрушка или любой.\n" +
		"3. Количество мотков: от 1 до 20.\n" +
		"4. Количество цветов: от 1 до 20.\n" +
		"5. Инструмент, который нужно использовать: крючок, спицы, пальцы\n" +
		"6. Тип пряжи: плюшевая, обычная, пуффи, мохер, или любой\n" +
		"7. Ссылка на схему:\n\n"

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, instructions)); err != nil {
		logger.WithError(err).Error("Error sending add new idea request message")
	}
	// Отправляем шаблон для ввода данных
	templateMessage := "1) Название изделия: \n" +
		"2) Тип изделия: \n" +
		"3) Количество мотков пряжи: \n" +
		"4) Количество цветов пряжи: \n" +
		"5) Инструмент, который нужно использовать: \n" +
		"6) Тип пряжи: \n" +
		"7) Ссылка на схему или видео: \n"

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, templateMessage)); err != nil {
		logger.WithError(err).Error("Error sending data input template message")

	}
}

// Функция поиска новых идей
func HandleProcessIdeaRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, db *sql.DB) {
	// Проверка на текстовое сообщение
	if msg.Text == "" {
		return // Игнорируем пустые сообщения
	}

	// Приводим текст сообщения к нижнему регистру
	text := strings.ToLower(msg.Text)

	// Разделяем текст на строки
	lines := strings.Split(text, "\n")
	if len(lines) != 5 {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка в формате запроса. Пожалуйста, используйте формат:\n1) Тип изделия:\n2) Количество мотков пряжи:\n3) Количество цветов пряжи:\n4) Инструмент, который будете использовать:\n5) Тип пряжи:")); err != nil {
			logger.WithError(err).Warn("User provided invalid format for idea request")
		}
		return
	}

	// Извлекаем и очищаем значения
	typeOfItem := strings.TrimSpace(strings.Split(lines[0], ":")[1])
	numberOfBalls, err := strconv.Atoi(strings.TrimSpace(strings.Split(lines[1], ":")[1]))
	if err != nil {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, укажите количество мотков цифрой")); err != nil {
			logger.WithError(err).Error("Error sending process idea request message")
		}
		return
	}

	numberOfColors, err := strconv.Atoi(strings.TrimSpace(strings.Split(lines[2], ":")[1]))
	if err != nil {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, укажите количество цветов цифрой")); err != nil {
			logger.WithError(err).Error("Error sending process idea request message")
		}
		return
	}

	toolType := strings.TrimSpace(strings.Split(lines[3], ":")[1])
	yarnType := strings.TrimSpace(strings.Split(lines[4], ":")[1])

	log.Printf("Received parameters: typeOfItem=%s, numberOfBalls=%d, numberOfColors=%d, toolType=%s, yarnType=%s",
		typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)

	// Логика поиска идей на основе введенных параметров
	result, err := repository.FindIdeas(db, typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)
	if err != nil {
		logger.WithError(err).Error("Error in FindIdeas function")
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ой, произошла ошибка, скореее всего с нашей стороны. Попробуйте еще раз, а если она повторится - отправьте жалобу, скоро все починим.")); err != nil {
			logger.WithError(err).Error("Error sending search ideas error message")
		}
		return
	}

	if len(result) == 0 {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "К сожалению, я не нашел идей по вашему запросу. Пожалуйста, попробуйте поменять некоторые параметры или проверьте правильность сообщения")); err != nil {
			logger.WithError(err).Error("Error sending no ideas found message")
		}
		return
	}

	// Генерация случайной идеи из результата
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(result))
	randomIdea := result[randomIndex]

	// Обновленный результат с добавлением названия идеи
	resultMessage := fmt.Sprintf("Вот что я нашел для тебя:\n\nID: %d\nНазвание изделия: %s\nТип изделия: %s\nКоличество мотков: %d\nКоличество цветов: %d\nИнструмент: %s\nТип пряжи: %s",
		randomIdea.ID, randomIdea.Title, randomIdea.TypeOfItem, randomIdea.NumberOfBalls, randomIdea.NumberOfColors, randomIdea.ToolType, randomIdea.YarnType)

	if randomIdea.SchemeURL != "" {
		resultMessage += fmt.Sprintf("\nСсылка на схему: %s", randomIdea.SchemeURL)
	}

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, resultMessage)); err != nil {
		logger.WithError(err).Error("Error sending found ideas message")
	}

	// Сохраняем ID предложенной идеи
	proposedIdeas = append(proposedIdeas, randomIdea.ID)

	// Сохранение параметров последнего поиска
	SaveLastSearch(msg.Chat.ID, typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)
}

// функция очищения списка предложенных идей
func ResetProposedIdeas() {
	proposedIdeas = []int{}
}

// Структура Idea для функции  заново
type Idea struct {
	ID             int
	Title          string
	TypeOfItem     string
	NumberOfBalls  int
	NumberOfColors int
	ToolType       string
	YarnType       string
	SchemeURL      string
}

// HandleRedoRequest обрабатывает запрос "Заново"
func HandleRedoRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, db *sql.DB) {
	logger.Infof("User %d requested to redo their last search", msg.Chat.ID)
	lastSearch, exists := lastSearches[msg.Chat.ID]
	if !exists {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ранее вы не осуществляли поиск, пожалуйста, воспользуйтесь кнопкой `Новая идея`")); err != nil {
			logger.WithError(err).Error("Error sending no previous searches message")
		}
		return
	}

	// Извлечение параметров последнего поиска
	typeOfItem := lastSearch[0].(string)
	numberOfBalls := lastSearch[1].(int)
	numberOfColors := lastSearch[2].(int)
	toolType := lastSearch[3].(string)
	yarnType := lastSearch[4].(string)

	// Повторный поиск идеи
	result, err := repository.FindIdeas(db, typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)
	if err != nil {
		logger.WithError(err).Error("Error in FindIdeas function")
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ой, произошла ошибка, скореее всего с нашей стороны. Попробуйте еще раз, а если она повторится - отправьте жалобу, скоро все починим.")); err != nil {
			logger.WithError(err).Error("Error sending redo request error message")
		}
		return
	}

	// Если идеи не найдены
	if len(result) == 0 {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "К сожалению, я не нашел идей для вашего предыдущего запроса.")); err != nil {
			logger.WithError(err).Error("Error sending no ideas found message")
		}
		return
	}

	// Фильтруем идеи, чтобы исключить уже предложенные
	var availableIdeas []Idea
	for _, idea := range result {
		if !contains(proposedIdeas, idea.ID) { // Проверяем, была ли идея предложена
			availableIdeas = append(availableIdeas, Idea(idea))
		}
	}

	// Если все идеи уже были предложены, сообщаем об этом и сбрасываем список
	if len(availableIdeas) == 0 {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Кажется вы посмотрели все доступные идеи! Сейчас обновлю список, и вы сможете посмотреть их еще раз")); err != nil {
			logger.WithError(err).Error("Error sending all ideas suggested message")
		}

		// Сбрасываем список предложенных идей
		ResetProposedIdeas()

		// Сообщаем пользователю о том, что можно начать заново
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Список идей обновлен :) ")); err != nil {
			logger.WithError(err).Error("Error sending reset message")
		}
		return
	}

	// Генерация случайной идеи из доступных
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(availableIdeas))
	randomIdea := availableIdeas[randomIndex]

	// Сохраняем ID предложенной идеи
	proposedIdeas = append(proposedIdeas, randomIdea.ID)

	// Формируем сообщение с новой идеей (с добавлением названия)
	message := fmt.Sprintf("Вот новая идея для вязания:\n\nID: %d\nНазвание изделия: %s\nТип изделия: %s\nКоличество мотков: %d\nКоличество цветов: %d\nИнструмент: %s\nТип пряжи: %s",
		randomIdea.ID, randomIdea.Title, randomIdea.TypeOfItem, randomIdea.NumberOfBalls, randomIdea.NumberOfColors, randomIdea.ToolType, randomIdea.YarnType)

	if randomIdea.SchemeURL != "" {
		message += fmt.Sprintf("\nСсылка на схему: %s", randomIdea.SchemeURL)
	}

	// Отправляем сообщение пользователю
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, message))
}

func contains(slice []int, item int) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}

// isProfane проверяет наличие нецензурных слов в названии
func isProfane(title string) bool {
	// Пример списка нецензурных слов
	profaneWords := []string{"хуй", "в последствии дополнить список"}

	// Приводим название к нижнему регистру для корректной проверки
	titleLower := strings.ToLower(title)

	// Проверяем наличие нецензурных слов в названии
	for _, word := range profaneWords {
		if strings.Contains(titleLower, word) {
			return true
		}
	}
	return false
}

// HandleProcessAddIdeaRequest обрабатывает добавление новой идеи
func HandleProcessAddIdeaRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	// Разбиваем сообщение на строки
	lines := strings.Split(msg.Text, "\n")
	if len(lines) != 7 {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка в формате запроса. Пожалуйста, используйте формат:\n"+
			"1. Название изделия:\n"+
			"2. Тип изделия: женское, мужское, детское, питомцам, аксессуар, интерьер, игрушка или любой.\n"+
			"3. Количество мотков: от 1 до 20.\n"+
			"4. Количество цветов: от 1 до 20.\n"+
			"5. Инструмент, который нужно использовать: крючок, спицы, пальцы.\n"+
			"6. Тип пряжи: плюшевая, обычная, пуффи, мохер или любой.\n"+
			"7. Ссылка на схему: ")); err != nil {
			logger.WithError(err).Error("Error sending process add idea request message")
		}
		return
	}

	// Извлекаем значения из строк
	title := extractValue(lines[0])
	logger.Infof("Title extracted: %s", title) // Логируем извлеченное значение
	// Проверяем название на нецензурную лексику
	if isProfane(title) {
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, не используйте нецензурную лексику!")); err != nil {
			logger.WithError(err).Error("Error sending profane title message")
		}
		return
	}

	typeOfItem := extractValue(lines[1])
	logger.Infof("Type of item extracted: %s", typeOfItem)

	numberOfBallsStr := extractValue(lines[2])
	numberOfBalls, err := strconv.Atoi(numberOfBallsStr)
	if err != nil || numberOfBalls < 1 || numberOfBalls > 20 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Количество мотков должно быть числом от 1 до 20"))
		return
	}
	logger.Infof("Number of balls: %d", numberOfBalls)

	numberOfColorsStr := extractValue(lines[3])
	numberOfColors, err := strconv.Atoi(numberOfColorsStr)
	if err != nil || numberOfColors < 1 || numberOfColors > 20 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Количество цветов должно быть числом от 1 до 20"))
		return
	}

	logger.Infof("Number of colors: %d", numberOfColors)

	toolType := extractValue(lines[4])
	yarnType := extractValue(lines[5])
	schemeURL := extractValue(lines[6])

	// Проверяем, что все поля заполнены
	if title == "" || typeOfItem == "" || toolType == "" || yarnType == "" || schemeURL == "" {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Кажется, некоторые поля не заполнены. Пожалуйста, заполните все поля."))
		return
	}

	// Проверяем, является ли введенная строка ссылкой
	if !isValidURL(schemeURL) {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Кажется, ссылка в неверном формате. Пожалуйста, укажите правильный URL."))
		return
	}

	// Приводим тип изделия, инструмент и пряжу к нижнему регистру для поиска
	typeOfItem = strings.ToLower(typeOfItem)
	toolType = strings.ToLower(toolType)
	yarnType = strings.ToLower(yarnType)

	idea := models.AddIdeaRequest{
		Title:          title,
		TypeOfItem:     typeOfItem,
		NumberOfBalls:  numberOfBalls,
		NumberOfColors: numberOfColors,
		ToolType:       toolType,
		YarnType:       yarnType,
		SchemeURL:      schemeURL,
	}

	if err := addIdeaToAPI(idea); err != nil {
		logger.WithError(err).Error("Failed to add idea to API")
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ой, произошла ошибка. Пожалуйста, проверьте, в правильном ли формате ваше сообщение. Если ошибка повторяется - отправьте жалобу"))
		return
	}

	result := fmt.Sprintf("Идея успешно добавлена: %s, %s, %d, %d, %s, %s", title, typeOfItem, numberOfBalls, numberOfColors, toolType, yarnType)
	if schemeURL != "" {
		result += fmt.Sprintf(", Ссылка на схему: %s", schemeURL)
	}

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, result)); err != nil {
		logger.WithError(err).Error("Error sending process add idea result message")
	}
}

// extractValue извлекает только нужное значение из строки, убирая лишний текст
func extractValue(line string) string {
	// Ищем значение после двоеточия
	parts := strings.SplitN(line, ":", 2) // Разделяем строку на 2 части
	if len(parts) < 2 {
		return "" // Если не найдено, возвращаем пустую строку
	}
	return strings.TrimSpace(parts[1]) // Возвращаем значение, убирая лишние пробелы
}

// isValidURL проверяет, является ли строка валидным URL
func isValidURL(urlStr string) bool {
	_, err := url.ParseRequestURI(urlStr)
	return err == nil
}

func addIdeaToAPI(idea models.AddIdeaRequest) error {
	apiURL := "http://api-service:8080/ideas"

	// Кодируем структуру в JSON
	ideaJSON, err := json.Marshal(idea)
	if err != nil {
		return fmt.Errorf("error marshalling idea to JSON: %w", err)
	}

	// Отправляем POST запрос
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(ideaJSON))
	if err != nil {
		return fmt.Errorf("error sending POST request: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус-код ответа
	if resp.StatusCode != http.StatusCreated {
		// Читаем тело ответа для получения дополнительной информации о ошибке
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("failed to read response body: %w", readErr)
		}
		return fmt.Errorf("failed to add idea: status code %d, response body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// HandleComplaintRequest обрабатывает запрос на жалобу
func HandleComplaintRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	complaintInstructions := "Пожалуйста, перешлите сообщение, на которое хочешь пожаловаться с подписью `Жалоба: текст вашей жалобы`, либо если ваша жалоба не связана с определенным сообщением, то просто напишите её в чат в формате `Жалоба: текст вашей жалобы`."
	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, complaintInstructions)); err != nil {
		logger.WithError(err).Error("Error sending complaint instructions")
	}
}

// forwardComplaintToGroup отправляет жалобу в групповой чат
func forwardComplaintToGroup(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	originalMessage := ""
	if msg.ReplyToMessage != nil {
		originalMessage = msg.ReplyToMessage.Text // Получаем текст оригинального сообщения
	}

	complaintMessage := fmt.Sprintf("Пользователь %s оставил жалобу:\n%s\n\nОригинальное сообщение:\n%s",
		msg.From.UserName, strings.TrimPrefix(msg.Text, "Жалоба:"), originalMessage)

	if _, err := bot.Send(tgbotapi.NewMessage(groupChatID, complaintMessage)); err != nil {
		logger.WithError(err).Error("Error sending complaint to group chat")
	} else {
		// Подтверждение пользователю
		confirmationMessage := "Ваша жалоба была отправлена модератору, извините за неудобства"
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, confirmationMessage)); err != nil {
			logger.WithError(err).Error("Error sending complaint confirmation message")
		}
	}
}

// HandleDeleteIdeaRequest обрабатывает запрос на удаление идеи
func HandleDeleteIdeaRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, db *sql.DB) {
	logrus.Infof("Received command: %s", msg.Text)
	// Разбираем команду и её аргументы
	parts := strings.Fields(msg.Text) // разбиваем текст на части

	if len(parts) != 2 {
		// Сообщаем, что формат команды некорректен
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка: Пожалуйста, используйте команду в формате / <ID>."))
		return
	}

	// Пробуем преобразовать вторую часть в ID
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Ошибка: Пожалуйста, укажите корректный ID идеи."))
		return
	}
	// Удаляем идею из базы данных
	if err := repository.DeleteIdeaByID(db, id); err != nil {
		logger.WithError(err).Error("Error deleting idea from database")
		if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Произошла ошибка при удалении идеи. Проверьте ID.")); err != nil {
			logger.WithError(err).Error("Error sending delete idea error message")
		}
		return
	}

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Идея с ID %d была успешно удалена.", id))); err != nil {
		logger.WithError(err).Error("Error sending delete success message")
	}
}

func HandleHelpRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	msgText := "Если у вас появились сложности при работе с ботом - ознакомьтесь с этим разделом, возможно он сможет вам помочь! \n\n" +
		"- Если вы не хотите указывать точное количество мотков или цветов пряжи, то используйте цифру 0.\n\n" +
		"- Для поиска идей с пряжей `пуффи` укажите как инструмент пальцы или крючок.\n\n" +
		"- Если бот не распознает ваше сообщение - пожалуйста, проверьте, нет ли в нем грамматических ошибок и все параметры введены после двоеточия\n\n" +
		"- Если содержимое какого-то сообщения показалось вам неприемлимым, то перешлите его боту с подписью `Жалоба: текст ваше жалобы` и бот обязательно передаст это мне.\n\n" +
		"- Если вы заметили какую-то ошибку в работе бота или вам просто что-то не нравится, то напишите в чат  `Жалоба: текст вашей жалобы` и бот обязательно передаст это мне.\n\n" +
		"❗❗❗   Осторожно, фишинговые ссылки!    ❗❗❗\n\n" +
		" К сожалению проверить все ссылки, которые добавляют другие пользователи - невозможно, по этому для того, чтобы обезопасить себя - используйте эти советы:\n\n" +
		"1) Всегда обращайте внимание на предпросмотр ссылки. Под ссылкой должно быть краткое описание сайта, на который она ведет, а также как правило будет присутствовать изображение. Если подобное отсутствует, или описание не имеет ничего общего с вязанием - по ссылке лучше не переходить .\n\n" +
		"2) Если сссылка направляет вас на сайт, где нужно обязательно ввести личные данные, перед просмотром схемы для вязания - делать этого точно не надо! Вероятнее всего сайт не безопасен\n\n" +
		"3) Не переходите по коротким ссылкам, они часто скрывают настоящий адрес \n" +
		"- Пример короткой ссылки  https://bit.ly/3xyz123 \n" +
		"- Обычная ссылка  https://www.example.com/category/item?id=12345&ref=67890 \n\n" +
		"4) Не переходите по ссылке состоящей в основном из цифр, например http://192.168.1.1/login \n\n" +
		"5) Обратите внимание на символы в ссылке. Если вы видите там символы # или % , то лучше по ссылке не переходить\n\n" +
		"Если вы заметили что-то из вышеуказанного в идее предложенной ботом, то пожалуйста - отпраьте жалобу, и я оперативно удалю это.\n\n" +
		"Если вдруг вы не нашли здесь ответ на свой вопрос, у вас есть предложения по улучшению бота или вы просто хотите пообщаться, то вы можете написать в чат. Ссылка на него есть в разделе информации."

	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
		logrus.WithError(err).Warn("Error sending help message")
	}
}

//func HandleSupportRequest(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
//	msgText := "❤️ Спасибо за желание поддержать разработчика! Если ты хочешь сделать донат, напиши в личные сообщения."
//	if _, err := bot.Send(tgbotapi.NewMessage(msg.Chat.ID, msgText)); err != nil {
//		logrus.WithError(err).Warn("Error sending support message")
//	}
// }
