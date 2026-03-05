# 📑 Index — Навигация по проекту

Быстрая навигация по всем компонентам проекта Etalon Nomenclature Service.

---

## 🚀 Начало работы

**Новичок в проекте?** Начните с:
1. [README.md](README.md) — обзор проекта
2. [QUICKSTART.md](QUICKSTART.md) — запуск за 8 минут
3. [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) — краткая сводка

**Готовы к деплою?**
1. [DEPLOYMENT.md](DEPLOYMENT.md) — production развертывание
2. [BUILD_VERIFICATION.md](BUILD_VERIFICATION.md) — статус готовности

**Хотите понять архитектуру?**
1. [STRUCTURE.md](STRUCTURE.md) — структура проекта

---

## 📚 Документация

### Основная документация
- **[README.md](README.md)** — Главная документация проекта
  - Описание возможностей
  - Быстрый старт
  - Конфигурация
  - База данных
  - Makefile команды
  - Troubleshooting

### Руководства
- **[QUICKSTART.md](QUICKSTART.md)** — Быстрый старт за 8 минут
  - Подготовка (5 мин)
  - База данных (2 мин)
  - Запуск (1 мин)
  - Проверка работы
  - Troubleshooting

- **[DEPLOYMENT.md](DEPLOYMENT.md)** — Production развертывание
  - Подготовка сервера
  - Настройка Docker
  - Настройка PostgreSQL
  - Применение миграций
  - Мониторинг и обслуживание
  - Backup и восстановление

### Архитектура
- **[STRUCTURE.md](STRUCTURE.md)** — Архитектура проекта
  - Структура директорий
  - Описание компонентов
  - Поток данных
  - Жизненный цикл обработки
  - Обработка ошибок

- **[PROJECT_SUMMARY.md](PROJECT_SUMMARY.md)** — Сводка по проекту
  - Общая информация
  - Архитектура
  - Конфигурация
  - Безопасность
  - Production features
  - Мониторинг

### Для разработчиков
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — Гайд для контрибьюторов
  - Как внести вклад
  - Code style
  - Тестирование
  - Pull Request процесс

- **[CHANGELOG.md](CHANGELOG.md)** — История версий
  - Список изменений
  - Версии
  - Планы развития

- **[BUILD_VERIFICATION.md](BUILD_VERIFICATION.md)** — Статус сборки
  - Проверка компиляции
  - Результаты тестов
  - Готовность к деплою

### Прочее
- **[LICENSE](LICENSE)** — MIT License
- **[INDEX.md](INDEX.md)** — Этот файл

---

## 💻 Исходный код

### Точка входа
- **[cmd/app/main.go](cmd/app/main.go)** — Основная точка входа
  - Инициализация логгера
  - Загрузка конфигурации
  - Подключение к БД
  - Запуск процессора
  - Graceful shutdown

### Конфигурация
- **[config/config.go](config/config.go)** — Управление конфигурацией
  - Структуры Config, DatabaseConfig, MailboxConfig
  - Загрузка из YAML
  - Валидация параметров
  - Дефолтные значения

- **[config/config_test.go](config/config_test.go)** — Тесты конфигурации

### База данных
- **[internal/db/postgres.go](internal/db/postgres.go)** — PostgreSQL клиент
  - Подключение с SSL
  - Connection pooling
  - Проверка обработанных писем
  - Вставка данных номенклатуры
  - Транзакционные операции

- **[internal/db/postgres_test.go](internal/db/postgres_test.go)** — Тесты БД

### IMAP клиент
- **[internal/imap/client.go](internal/imap/client.go)** — Email обработка
  - Подключение к IMAP серверу
  - Поиск писем за текущий день
  - Парсинг email и вложений
  - Фильтрация .xlsx файлов
  - Retry логика

### Excel парсер
- **[internal/parser/excel.go](internal/parser/excel.go)** — Парсинг Excel
  - Streaming парсинг для экономии памяти
  - Обработка всех вкладок
  - Поиск заголовков колонок
  - Парсинг строк данных
  - Обработка опциональных колонок

- **[internal/parser/excel_test.go](internal/parser/excel_test.go)** — Тесты парсера

### Бизнес-логика
- **[internal/service/processor.go](internal/service/processor.go)** — Процессор
  - Основной polling loop
  - Обработка множественных mailbox
  - Координация компонентов
  - Обработка ошибок
  - Panic recovery

---

## 🗄️ База данных

### Миграции
- **[migrations/001_init.sql](migrations/001_init.sql)** — Начальная миграция
  - Создание таблицы etalon_nomenclature
  - Создание таблицы processed_emails
  - Создание индексов

### Таблицы

**etalon_nomenclature**
```sql
- id (SERIAL PRIMARY KEY)
- article (TEXT) — Артикул
- brand (TEXT) — Марка
- type (TEXT) — Тип
- size_model (TEXT) — Размер и Модель
- nomenclature (TEXT) — Номенклатура
- mrc (NUMERIC) — МРЦ
- isimport (INTEGER) — Флаг импорта
- created_at (TIMESTAMP) — Дата создания
```

**processed_emails**
```sql
- id (SERIAL PRIMARY KEY)
- message_id (TEXT UNIQUE) — Message-ID письма
- processed_at (TIMESTAMP) — Дата обработки
```

---

## 🐳 Docker

### Dockerfile
- **[docker/Dockerfile](docker/Dockerfile)** — Docker образ
  - Multi-stage build
  - Alpine Linux base
  - Non-root пользователь
  - Минимальный размер

### Docker Compose
- **[docker-compose.yml](docker-compose.yml)** — Оркестрация
  - Конфигурация сервиса
  - Volume mappings
  - Health checks
  - Resource limits
  - Logging configuration

### Конфигурация Docker
- **[.dockerignore](.dockerignore)** — Игнорируемые файлы

---

## 🔧 Конфигурация и скрипты

### Конфигурационные файлы
- **[config.example.yaml](config.example.yaml)** — Пример конфигурации
- **[.env.example](.env.example)** — Пример переменных окружения
- **[.gitignore](.gitignore)** — Игнорируемые файлы Git
- **[.editorconfig](.editorconfig)** — Настройки редактора
- **[.golangci.yml](.golangci.yml)** — Конфигурация линтера

### Скрипты
- **[scripts/apply-migrations.sh](scripts/apply-migrations.sh)** — Применение миграций
  - Проверка psql
  - Извлечение DSN из config.yaml
  - Применение всех миграций

- **[scripts/test-connection.sh](scripts/test-connection.sh)** — Проверка подключений
  - Проверка config.yaml
  - Проверка SSL сертификата
  - Тест IMAP соединения
  - Тест PostgreSQL соединения

### Makefile
- **[Makefile](Makefile)** — Автоматизация команд
  - `make build` — сборка
  - `make run` — локальный запуск
  - `make test` — запуск тестов
  - `make docker-build` — сборка Docker
  - `make docker-up` — запуск Docker
  - `make migrate` — применение миграций

---

## 📦 Go модули

- **[go.mod](go.mod)** — Go модули и зависимости
- **[go.sum](go.sum)** — Checksums зависимостей

### Основные зависимости
1. `github.com/emersion/go-imap` — IMAP протокол
2. `github.com/xuri/excelize/v2` — Excel парсинг
3. `github.com/lib/pq` — PostgreSQL драйвер
4. `go.uber.org/zap` — Structured logging
5. `gopkg.in/yaml.v3` — YAML парсинг

---

## 🔍 Поиск по темам

### Хочу настроить...
- **Почтовые ящики** → [config.example.yaml](config.example.yaml)
- **База данных** → [config.example.yaml](config.example.yaml) + [DEPLOYMENT.md](DEPLOYMENT.md)
- **Интервал проверки** → [config.example.yaml](config.example.yaml)
- **SSL сертификаты** → [DEPLOYMENT.md](DEPLOYMENT.md)
- **Docker ресурсы** → [docker-compose.yml](docker-compose.yml)

### Хочу понять...
- **Как работает IMAP** → [internal/imap/client.go](internal/imap/client.go) + [STRUCTURE.md](STRUCTURE.md)
- **Как парсится Excel** → [internal/parser/excel.go](internal/parser/excel.go)
- **Как работает БД** → [internal/db/postgres.go](internal/db/postgres.go)
- **Архитектуру** → [STRUCTURE.md](STRUCTURE.md)
- **Обработку ошибок** → [STRUCTURE.md](STRUCTURE.md)

### Хочу исправить...
- **Ошибку подключения к IMAP** → [README.md](README.md#troubleshooting)
- **Ошибку подключения к БД** → [DEPLOYMENT.md](DEPLOYMENT.md#troubleshooting)
- **Проблемы с Docker** → [DEPLOYMENT.md](DEPLOYMENT.md#troubleshooting)
- **Проблемы с парсингом** → [README.md](README.md#troubleshooting)

### Хочу добавить...
- **Новый mailbox** → [config.example.yaml](config.example.yaml)
- **Новую колонку в Excel** → [internal/parser/excel.go](internal/parser/excel.go)
- **Новую таблицу в БД** → создать новую миграцию
- **Новую функциональность** → [CONTRIBUTING.md](CONTRIBUTING.md)

---

## 🧪 Тестирование

### Unit тесты
- [config/config_test.go](config/config_test.go) — 5 тестов
- [internal/db/postgres_test.go](internal/db/postgres_test.go) — 1 тест
- [internal/parser/excel_test.go](internal/parser/excel_test.go) — 7 тестов

### Запуск тестов
```bash
make test              # Все тесты
make test-coverage     # С покрытием
go test -v ./config    # Конкретный пакет
```

---

## 📞 Получить помощь

### Документация
1. Проверьте [README.md](README.md) — раздел Troubleshooting
2. Проверьте [DEPLOYMENT.md](DEPLOYMENT.md) — детали развертывания
3. Проверьте [QUICKSTART.md](QUICKSTART.md) — быстрый старт

### Вопросы и проблемы
1. Создайте Issue в репозитории
2. Опишите проблему с логами
3. Укажите версию Go/Docker

### Контрибьюция
1. Прочитайте [CONTRIBUTING.md](CONTRIBUTING.md)
2. Fork репозитория
3. Создайте Pull Request

---

## 📊 Статус проекта

**Версия:** 1.0.0
**Дата:** 2026-03-05
**Статус:** Production Ready ✅

Полный checklist готовности см. в [BUILD_VERIFICATION.md](BUILD_VERIFICATION.md)

---

**Создано с ❤️ для Pro Koleso**
