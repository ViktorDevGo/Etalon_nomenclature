# Структура проекта

```
Etalon_nomenclature/
│
├── cmd/
│   └── app/
│       └── main.go                 # Точка входа приложения
│
├── config/
│   ├── config.go                   # Загрузка и валидация конфигурации
│   └── config_test.go              # Тесты конфигурации
│
├── internal/
│   ├── db/
│   │   ├── postgres.go             # PostgreSQL клиент и операции с БД
│   │   └── postgres_test.go        # Тесты для БД
│   │
│   ├── imap/
│   │   └── client.go               # IMAP клиент для работы с email
│   │
│   ├── parser/
│   │   ├── excel.go                # Парсер Excel файлов
│   │   └── excel_test.go           # Тесты парсера
│   │
│   └── service/
│       └── processor.go            # Основная бизнес-логика обработки
│
├── migrations/
│   └── 001_init.sql                # SQL миграция для создания таблиц
│
├── scripts/
│   ├── apply-migrations.sh         # Скрипт применения миграций
│   └── test-connection.sh          # Скрипт проверки подключений
│
├── docker/
│   └── Dockerfile                  # Dockerfile для сборки образа
│
├── .editorconfig                   # Настройки форматирования кода
├── .dockerignore                   # Игнорируемые файлы для Docker
├── .gitignore                      # Игнорируемые файлы для Git
├── .golangci.yml                   # Конфигурация линтера
├── .env.example                    # Пример переменных окружения
│
├── config.example.yaml             # Пример конфигурации
├── docker-compose.yml              # Docker Compose конфигурация
├── go.mod                          # Go модули
│
├── LICENSE                         # Лицензия MIT
├── Makefile                        # Makefile с полезными командами
├── README.md                       # Основная документация
├── DEPLOYMENT.md                   # Руководство по развертыванию
└── STRUCTURE.md                    # Этот файл
```

## Описание компонентов

### cmd/app/main.go
- Инициализация логгера (zap)
- Загрузка конфигурации
- Подключение к БД
- Запуск процессора
- Graceful shutdown

### config/config.go
- Структуры конфигурации
- Загрузка из YAML
- Валидация параметров
- Установка дефолтных значений

### internal/db/postgres.go
- Подключение к PostgreSQL с SSL
- Проверка обработанных писем (Message-ID)
- Вставка данных номенклатуры
- Транзакционные операции
- Connection pooling

### internal/imap/client.go
- Подключение к IMAP серверу по TLS
- Поиск писем за текущий день
- Парсинг email и вложений
- Фильтрация .xlsx файлов
- Проверка размера вложений
- Retry логика

### internal/parser/excel.go
- Streaming парсинг Excel
- Обработка всех вкладок
- Поиск заголовков колонок
- Парсинг строк данных
- Обработка опциональных колонок
- Парсинг чисел с запятыми

### internal/service/processor.go
- Основной polling loop
- Обработка множественных почтовых ящиков
- Координация IMAP, Parser, DB
- Обработка ошибок
- Panic recovery
- Логирование процесса

### migrations/001_init.sql
- Создание таблицы etalon_nomenclature
- Создание таблицы processed_emails
- Индексы для оптимизации
- Комментарии к таблицам

### docker/Dockerfile
- Multi-stage build
- Alpine Linux base
- Non-root пользователь
- Минимальный размер образа
- CA certificates для HTTPS

### docker-compose.yml
- Конфигурация сервиса
- Volume mappings
- Health checks
- Resource limits
- Logging configuration

## Поток данных

```
[Email Server (REG.RU)]
         ↓
    [IMAP Client] ← проверка каждые 1 мин
         ↓
  [Email с .xlsx]
         ↓
    [Excel Parser] ← streaming парсинг
         ↓
   [Data Rows]
         ↓
   [PostgreSQL] ← транзакционная вставка
         ↓
  [etalon_nomenclature]
         +
  [processed_emails]
```

## Жизненный цикл обработки письма

1. **Поиск писем**
   - Подключение к IMAP
   - SEARCH SINCE today
   - Получение списка писем

2. **Фильтрация**
   - Проверка наличия вложений
   - Фильтр по расширению .xlsx
   - Проверка размера (макс 10 MB)

3. **Проверка дубликатов**
   - Запрос Message-ID в processed_emails
   - Если есть — пропуск
   - Если нет — продолжение

4. **Парсинг**
   - Открытие Excel файла
   - Итерация по всем sheets
   - Поиск заголовков
   - Парсинг строк
   - Валидация данных

5. **Сохранение**
   - BEGIN транзакция
   - INSERT данные в etalon_nomenclature
   - INSERT Message-ID в processed_emails
   - COMMIT
   - При ошибке — ROLLBACK

6. **Логирование**
   - Успешная обработка
   - Количество строк
   - Статистика

## Обработка ошибок

### IMAP ошибки
- Retry 3 раза с задержкой 5 сек
- Логирование каждой попытки
- Graceful degradation

### Парсинг ошибки
- Логирование с filename
- Продолжение с другими файлами
- Письмо помечается как обработанное

### База данных ошибки
- Rollback транзакции
- Message-ID не сохраняется
- Письмо будет обработано повторно

### Panic
- Recovery в процессоре
- Stack trace в логи
- Сервис продолжает работу

## Конфигурация

### Обязательные параметры
- `poll_interval` — интервал проверки
- `database.dsn` — подключение к БД
- `mailboxes[].email` — email адрес
- `mailboxes[].password` — пароль
- `mailboxes[].host` — IMAP сервер
- `mailboxes[].port` — IMAP порт

### Опциональные параметры
- `database.ssl_root_cert` — SSL сертификат
- `database.max_open_conns` — макс подключений (default: 25)
- `database.max_idle_conns` — idle подключений (default: 5)
- `database.conn_max_lifetime` — время жизни (default: 5m)

## Требования

### Runtime
- Go 1.22+
- PostgreSQL 14+
- Docker 20+ (для продакшена)
- Docker Compose 2+ (для продакшена)

### Зависимости
- `github.com/emersion/go-imap` — IMAP клиент
- `github.com/emersion/go-message` — парсинг email
- `github.com/lib/pq` — PostgreSQL драйвер
- `github.com/xuri/excelize/v2` — Excel парсер
- `go.uber.org/zap` — structured logging
- `gopkg.in/yaml.v3` — YAML парсер

## Production features

✅ Graceful shutdown
✅ Structured logging
✅ Context cancellation
✅ Connection pooling
✅ Transaction safety
✅ Panic recovery
✅ Retry логика
✅ Health checks
✅ Resource limits
✅ Non-root container
✅ Multi-stage build
✅ SSL support
✅ Streaming парсинг
✅ Duplicate protection

## Мониторинг и логи

### Ключевые метрики для мониторинга
- Количество обработанных писем
- Количество ошибок парсинга
- Количество ошибок БД
- Количество ошибок IMAP
- Время обработки письма
- Количество строк в письме
- Использование памяти

### Логи для алертов
- `"Failed to connect"` — проблемы с IMAP
- `"Failed to save data"` — проблемы с БД
- `"Panic recovered"` — критические ошибки
- `"Failed to parse"` — проблемы с файлами
