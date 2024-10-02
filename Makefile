.PHONY: restart stop build up help

stop: ## Останавливает все Docker-контейнеры
	@echo "Stopping all Docker containers..."
	docker-compose down

build: ## Пересобирает все Docker-контейнеры
	@echo "Building Docker containers..."
	docker-compose build --no-cache

up-build: ## Бмлдит и запускает все Docker-контейнеры
	@echo "Starting Docker containers..."
	docker-compose up -d --build

restart: stop build up ## Перезапускает приложение (остановка, пересборка, запуск контейнеров)
	@echo "Restart complete. The application is up and running."

status:	## Проверка состояния Docker-контейнеров
	@echo "Status containers"
	docker-compose ps

network: ## Создание сети для docker
	@echo  "Create network ..."
	docker network create backend-app

down: ## Остановка Удаление всех docker контейнеров
	@echo "Down containers ..."
	docker-compose down

remove: ## Удаление докер сети приложения
	@echo "Removing networks"
	docker network rm backend-app

start: network build up-build ## Первый запуск РЕКОМЕНДУЕТСЯ использовать

up: ## Запуск готовых контейнеров для работы
	docker-compose up -d

help: ## Показывает это сообщение с описанием всех команд
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
