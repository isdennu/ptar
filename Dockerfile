# Этап сборки
FROM golang:1.21-alpine AS builder

# Установка необходимых зависимостей для сборки
RUN apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование файлов проекта
COPY go.mod .
COPY main.go .

# Сборка приложения со статической линковкой
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o ptar main.go

# Финальный этап
FROM scratch

# Копирование бинарного файла из этапа сборки
COPY --from=builder /app/ptar /ptar

# Точка входа
ENTRYPOINT ["/ptar"] 