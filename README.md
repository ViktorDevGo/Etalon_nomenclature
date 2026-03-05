# Etalon Nomenclature Service

Production-ready сервис на Golang для автоматической обработки email с Excel файлами и записи данных в PostgreSQL.

## Возможности

- ✉️ Автоматическая проверка почты по IMAP
- 📊 Парсинг Excel файлов (.xlsx) со всех вкладок
- 🗄️ Запись данных в PostgreSQL
- 🔄 Поддержка множественных почтовых ящиков
- 🛡️ Защита от повторной обработки (по Message-ID)
- 📦 Streaming парсинг для экономии памяти
- 🔒 SSL подключение к PostgreSQL
- 📝 Structured logging (zap)
- 🐳 Docker и Docker Compose
- ♻️ Graceful shutdown
- 🚨 Panic recovery
- 🔁 Автоматический retry при сбоях IMAP

## Архитектура

```
.
├── cmd/app/              # Точка входа приложения
├── config/               # Конфигурация
├── internal/
│   ├── db/              # PostgreSQL клиент
│   ├── imap/            # IMAP клиент
│   ├── parser/          # Excel парсер
│   └── service/         # Бизнес-логика
├── migrations/           # SQL миграции
├── docker/              # Dockerfile
├── docker-compose.yml   # Docker Compose конфигурация
└── config.example.yaml  # Пример конфигурации
```

## Требования

- Go 1.22+
- PostgreSQL 14+
- Docker и Docker Compose (для продакшена)

## Быстрый старт

### 1. Клонирование и настройка

```bash
git clone <repository>
cd Etalon_nomenclature
```

### 2. Создание конфигурации

```bash
cp config.example.yaml config.yaml
```

Отредактируйте `config.yaml`:

```yaml
poll_interval: 1m

database:
  dsn: "postgresql://user:password@host:5432/database?sslmode=verify-full"
  ssl_root_cert: "/app/certs/root.crt"

mailboxes:
  - email: "your-email@domain.com"
    password: "your-password"
    host: "mail.hosting.reg.ru"
    port: 993
```

### 3. Применение миграций

```bash
# Подключитесь к вашей PostgreSQL БД
psql "postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full"

# Выполните миграцию
\i migrations/001_init.sql
```

Или через переменную окружения:

```bash
export DATABASE_URL="postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full"
make migrate
```

### 4. Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/app/main.go
```

### 5. Docker запуск

```bash
# Сборка и запуск
docker compose up -d

# Просмотр логов
docker compose logs -f app

# Остановка
docker compose down
```

## Конфигурация

### Параметры config.yaml

| Параметр | Описание | Пример |
|----------|----------|--------|
| `poll_interval` | Интервал проверки почты | `1m` |
| `database.dsn` | PostgreSQL connection string | `postgresql://...` |
| `database.ssl_root_cert` | Путь к SSL сертификату | `/app/certs/root.crt` |
| `mailboxes[].email` | Email адрес | `user@domain.com` |
| `mailboxes[].password` | Пароль от почты | `password` |
| `mailboxes[].host` | IMAP хост | `mail.hosting.reg.ru` |
| `mailboxes[].port` | IMAP порт | `993` |

### Множественные почтовые ящики

Добавьте столько почтовых ящиков, сколько нужно:

```yaml
mailboxes:
  - email: "box1@domain.com"
    password: "pass1"
    host: "mail.hosting.reg.ru"
    port: 993

  - email: "box2@domain.com"
    password: "pass2"
    host: "mail.hosting.reg.ru"
    port: 993
```

## База данных

### Таблица: etalon_nomenclature

Хранит данные из Excel файлов:

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | SERIAL | Первичный ключ |
| `article` | TEXT | Артикул |
| `brand` | TEXT | Марка |
| `type` | TEXT | Тип (опционально) |
| `size_model` | TEXT | Размер и Модель |
| `nomenclature` | TEXT | Номенклатура |
| `mrc` | NUMERIC | МРЦ (цена) |
| `isimport` | INTEGER | Флаг импорта (0/1) |
| `created_at` | TIMESTAMP | Дата загрузки |

### Таблица: processed_emails

Защита от повторной обработки:

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | SERIAL | Первичный ключ |
| `message_id` | TEXT | Message-ID письма (уникальный) |
| `processed_at` | TIMESTAMP | Дата обработки |

## Логика работы

### Polling цикл

1. Каждую **1 минуту** сервис проверяет все настроенные почтовые ящики
2. Ищет письма за **текущий день** (SINCE today)
3. Фильтрует письма с вложениями `.xlsx`
4. Проверяет размер файла (макс. 10 MB)
5. Проверяет, не обработано ли письмо ранее (по Message-ID)

### Обработка Excel

1. **Streaming парсинг** — экономия памяти
2. Обработка **всех вкладок** в файле
3. Поиск строки с колонками:
   - Артикул
   - Марка
   - Тип (опционально)
   - Размер и Модель
   - Номенклатура
   - МРЦ
4. Парсинг всех строк после заголовка

### Запись в БД

1. Все данные вставляются в транзакции
2. После успешной вставки сохраняется Message-ID
3. При ошибке транзакция откатывается, Message-ID не сохраняется
4. Письмо можно будет обработать повторно

### Обработка ошибок

- **IMAP ошибки**: автоматический retry (3 попытки с задержкой 5 сек)
- **Парсинг ошибки**: логируются, письмо помечается как обработанное
- **DB ошибки**: транзакция откатывается, письмо не помечается
- **Panic**: перехватывается, логируется stack trace, сервис продолжает работу

## Makefile команды

```bash
make help           # Показать все команды
make build          # Собрать бинарь
make run            # Запустить локально
make test           # Запустить тесты
make docker-build   # Собрать Docker образ
make docker-up      # Запустить в Docker
make docker-down    # Остановить Docker
make docker-logs    # Показать логи
make migrate        # Применить миграции
```

## Мониторинг

### Логи

Сервис использует structured logging (zap):

```json
{
  "level": "info",
  "ts": 1234567890,
  "msg": "Processing email",
  "message_id": "<id>",
  "subject": "Subject",
  "attachments": 2
}
```

### Health Check

Docker Compose включает health check:

```bash
docker compose ps
```

### Метрики логов

Основные события для мониторинга:

- `"Starting email processing cycle"` — начало цикла
- `"Found emails with attachments"` — найдены письма
- `"Successfully processed email"` — письмо обработано
- `"Failed to process email"` — ошибка обработки
- `"Panic recovered"` — критическая ошибка

## Production deployment

### На сервере Timeweb VM

1. **Установите Docker и Docker Compose**

```bash
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh
```

2. **Загрузите SSL сертификат**

```bash
mkdir -p ~/.cloud-certs
# Поместите root.crt в ~/.cloud-certs/
```

3. **Создайте config.yaml**

```bash
nano config.yaml
# Вставьте вашу конфигурацию
```

4. **Запустите сервис**

```bash
docker compose up -d
```

5. **Проверьте логи**

```bash
docker compose logs -f app
```

### Автозапуск при перезагрузке

Docker Compose с `restart: unless-stopped` автоматически перезапустит сервис.

### Обновление

```bash
git pull
docker compose down
docker compose build
docker compose up -d
```

## Безопасность

- ✅ Не хранит пароли в коде
- ✅ SSL для PostgreSQL
- ✅ Non-root пользователь в Docker
- ✅ Ограничение ресурсов в Docker
- ✅ Проверка размера вложений
- ✅ Prepared statements для SQL

## Troubleshooting

### Не удается подключиться к IMAP

```
Проверьте:
1. Правильность host/port в config.yaml
2. Логин и пароль
3. Доступ к mail.hosting.reg.ru:993
4. Файрвол на сервере
```

### Не удается подключиться к PostgreSQL

```
Проверьте:
1. DSN в config.yaml
2. Наличие сертификата root.crt
3. Переменную PGSSLROOTCERT
4. Доступ к БД с сервера
```

### Письма не обрабатываются

```
Проверьте:
1. Логи: docker compose logs -f app
2. Таблицу processed_emails — возможно письмо уже обработано
3. Формат Excel файла — должны быть правильные колонки
```

### Высокое потребление памяти

```
1. Проверьте размер Excel файлов
2. Убедитесь, что используется streaming парсинг
3. Настройте limits в docker-compose.yml
```

## Разработка

### Добавление новых колонок

1. Обновите структуру `NomenclatureRow` в `internal/db/postgres.go`
2. Добавьте колонку в SQL миграцию
3. Обновите парсер в `internal/parser/excel.go`

### Тестирование

```bash
# Запуск всех тестов
make test

# С покрытием
make test-coverage

# Конкретный пакет
go test -v ./internal/parser
```

## Лицензия

MIT

## Поддержка

При возникновении проблем:
1. Проверьте логи
2. Изучите раздел Troubleshooting
3. Создайте issue в репозитории
