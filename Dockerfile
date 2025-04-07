# Этап сборки
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

# Установка необходимых зависимостей для сборки
RUN apk add --no-cache git

# Установка рабочей директории
WORKDIR /app

# Копирование файлов проекта
COPY go.mod .
COPY main.go .

# Сборка приложения со статической линковкой
ARG TARGETPLATFORM
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -a -ldflags '-extldflags "-static"' -o ptar main.go

# Финальный этап
FROM scratch

# Копирование бинарного файла из этапа сборки
COPY --from=builder /app/ptar /ptar

# Точка входа
ENTRYPOINT ["/ptar"] 