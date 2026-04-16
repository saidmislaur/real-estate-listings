# Flatly

Backend-сервис для публикации объявлений о продаже квартир с модерацией и ролевым доступом.

## Что реализовано

- `POST /dummyLogin` для получения токена роли `client` или `moderator`
- `POST /register` и `POST /login` для почтовой авторизации
- `POST /house/create` только для модератора
- `POST /flat/create` для клиента и модератора
- `POST /flat/update` только для модератора с защитой от двойного взятия в модерацию
- `GET /house/{id}` с разной выдачей для клиента и модератора
- `POST /house/{id}/subscribe` для подписки на новые квартиры
- PostgreSQL без ORM, Docker и Docker Compose
- модульные и интеграционные тесты на ключевые сценарии

## Быстрый старт

```bash
docker compose up --build
```

После старта примените миграции любым удобным способом, например через `goose`:

```bash
goose -dir migrations postgres "postgres://said:said12@localhost:5432/testavito?sslmode=disable" up
```

Локальный запуск без Docker:

```bash
go run ./cmd/app
```

## Тесты

```bash
go test ./...
```
