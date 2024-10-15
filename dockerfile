# Используйте многослойную сборку для сборки Go приложений
FROM golang:1.18 AS builder

WORKDIR /app

# Копируем файлы go.mod и go.sum
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Компилируем приложения
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./api-service/api-service ./api-service/main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./bot-service/bot-service ./bot-service/main.go

# Создайте второй образ для запуска
FROM alpine:latest

WORKDIR /app

# Копируем исполняемые файлы и конфиги
COPY --from=builder /app/api-service/api-service /app/api-service
COPY --from=builder /app/bot-service/bot-service /app/bot-service
COPY --from=builder /app/api-service/config.yaml /app/
COPY --from=builder /app/bot-service/config.yaml /app/

# Устанавливаем необходимые утилиты
RUN apk add --no-cache bash

# Указываем команду запуска для api-service по умолчанию
CMD ["/app/api-service"]  # Это можно заменить в docker-compose
