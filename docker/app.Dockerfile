# Используем образ с Go для сборки
FROM golang:1.22-bullseye AS builder

WORKDIR /app

# Копируем файлы для установки зависимостей
COPY go.mod go.sum ./
RUN go mod download

# Копируем весь проект
COPY . .

# Собираем исполняемый файл для Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /usr/local/bin/app ./cmd

# Проверяем, что файл скомпилирован
RUN ls -la /usr/local/bin/app

# Используем минимальный образ для запуска приложения
FROM ubuntu:20.04 AS runner

# Копируем собранный исполняемый файл из builder-образа
COPY --from=builder /usr/local/bin/app /usr/local/bin/app

# Указываем переменную окружения для исполняемых файлов
ENV PATH="/usr/local/bin:$PATH"

EXPOSE 8080

# Запускаем приложение
CMD ["app"]
