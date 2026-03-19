# Etalon Nomenclature Service

Production-ready сервис на Golang для автоматической обработки email с Excel прайс-листами и записи данных в PostgreSQL.

## 🎯 Основные возможности

- ✉️ **Автоматическая обработка email** - IMAP мониторинг писем от поставщиков
- 📊 **Парсинг Excel файлов** - поддержка .xls и .xlsx, streaming парсинг
- 🗄️ **4 типа данных** - номенклатура МРЦ, цены шин/дисков, номенклатура дисков
- 🔄 **Умная дедупликация** - APPEND-ONLY, UPSERT и SKIP логики
- 🛡️ **Защита от повторов** - отслеживание обработанных писем
- 🔒 **SSL подключения** - безопасное подключение к PostgreSQL
- 🔧 **Автомиграции БД** - автоматическое применение при старте
- 📝 **Structured logging** - JSON логи с ротацией
- 🐳 **Docker ready** - полная поддержка контейнеризации
- ♻️ **Graceful shutdown** - корректная остановка с завершением транзакций

## 📦 Структура БД

### Таблицы

| Таблица | Назначение | Источник | Логика |
|---------|-----------|----------|--------|
| `mrc_etalon` | Номенклатура с МРЦ | ЗАПАСКА (Excel с "МРЦ") | Append-only |
| `tyres_prices_stock` | Цены и остатки шин | Все поставщики | UPSERT |
| `rims_prices_stock` | Цены и остатки дисков | Все поставщики | UPSERT |
| `nomenclature_rims` | Номенклатура дисков | ЗАПАСКА (COX/FF/Koko/Sakura) | SKIP |
| `processed_emails` | Обработанные письма | Система | Append-only |

### Поставщики

- **ЗАПАСКА** (pna@sibzapaska.ru) - номенклатура МРЦ, прайсы, номенклатура дисков
- **БИГМАШИН** - прайсы шин и дисков
- **БРИНЕКС** - прайсы шин и дисков

## 🚀 Быстрый старт

### Требования

- Go 1.22+ (для разработки)
- PostgreSQL 14+
- Docker и Docker Compose (для продакшена)
- IMAP доступ к почтовому ящику

### Установка

```bash
# 1. Клонирование репозитория
git clone <repository>
cd Etalon_nomenclature

# 2. Создание конфигурации
cp config.example.yaml config.yaml

# 3. Редактирование config.yaml
# Укажите параметры БД и почтовых ящиков

# 4. Запуск с Docker Compose
docker compose up -d

# 5. Просмотр логов
docker compose logs -f app
```

### Конфигурация

Минимальный `config.yaml`:

```yaml
poll_interval: 30m

database:
  dsn: "postgresql://user:password@host:5432/db?sslmode=require"

mailboxes:
  - email: "zakupki@etalon-shina.ru"
    password: "your-password"
    host: "mail.hosting.reg.ru"
    port: 993

allowed_senders:
  - "pna@sibzapaska.ru"
```

## 📚 Документация

### Основная документация

- [📖 Индекс документации](docs/INDEX.md) - Полный список документов
- [🚀 Деплой на продакшн](DEPLOYMENT.md) - Подробные инструкции
- [⚡ Быстрая справка](QUICK_REFERENCE.md) - Частые команды и запросы
- [✅ Production Checklist](PRODUCTION_CHECKLIST.md) - Чеклист перед деплоем

### Логика работы

- [📋 Обработка файлов](FILE_PROCESSING_LOGIC.md) - Детальная логика парсинга
- [🔄 Дедупликация](DEDUPLICATION_LOGIC.md) - Логика дедупликации данных

### Миграции БД

- [🔄 Миграция tyres](MIGRATION_TYRES_PRICES_STOCK.md) - price_tires → tyres_prices_stock
- [🔄 Миграция rims](MIGRATION_RIMS_PRICES_STOCK.md) - price_disks → rims_prices_stock + nomenclature_rims
- [🏷️ Переименование](MIGRATION_RENAME_TABLE.md) - etalon_nomenclature → mrc_etalon
- [➕ Добавление isimport_1С](MIGRATION_ADD_ISIMPORT_1C.md) - Флаг импорта в 1С

## 🏗️ Архитектура

```
etalon-nomenclature/
├── cmd/app/              # Точка входа
├── internal/
│   ├── db/              # PostgreSQL + миграции
│   ├── imap/            # IMAP клиент
│   ├── parser/          # Excel парсеры
│   │   ├── excel.go     # Парсер номенклатуры МРЦ
│   │   ├── price.go     # Парсер цен шин
│   │   └── disk.go      # Парсер цен/номенклатуры дисков
│   └── service/         # Бизнес-логика
├── migrations/           # SQL миграции
├── docker/              # Dockerfile
├── docs/                # Документация
└── config.yaml          # Конфигурация (не в git)
```

## 🔧 Разработка

### Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/app/main.go
```

### Тестирование

```bash
# Запуск всех тестов
go test ./...

# Тестирование парсеров
go test ./internal/parser/...

# Тестирование БД
go test ./internal/db/...
```

### Сборка

```bash
# Локальная сборка
go build -o app cmd/app/main.go

# Сборка для продакшена (в Docker)
docker compose build
```

## 📊 Мониторинг

### Логи

```bash
# Просмотр логов
docker compose logs -f app

# Последние 100 строк
docker compose logs --tail=100 app

# Логи с временными метками
docker compose logs -f --timestamps app
```

### Статус

```bash
# Статус контейнеров
docker compose ps

# Использование ресурсов
docker stats
```

### Проверка БД

```sql
-- Количество записей
SELECT 'mrc_etalon' as table, COUNT(*) as count FROM mrc_etalon
UNION ALL
SELECT 'tyres_prices_stock', COUNT(*) FROM tyres_prices_stock
UNION ALL
SELECT 'rims_prices_stock', COUNT(*) FROM rims_prices_stock
UNION ALL
SELECT 'nomenclature_rims', COUNT(*) FROM nomenclature_rims
UNION ALL
SELECT 'processed_emails', COUNT(*) FROM processed_emails;

-- Последние обработанные письма
SELECT message_id, email_date, processed_at
FROM processed_emails
ORDER BY processed_at DESC
LIMIT 10;
```

## 🔐 Безопасность

### Конфиденциальные файлы

Следующие файлы содержат секреты и **не должны** попадать в Git:

- `config.yaml` - пароли БД и email
- `.env` - переменные окружения
- `certs/*.crt` - SSL сертификаты

Все они добавлены в `.gitignore`.

### SSL Подключение

Для SSL подключения к PostgreSQL:

```yaml
database:
  dsn: "postgresql://user:password@host:5432/db?sslmode=verify-full"
  ssl_root_cert: "/path/to/root.crt"
```

## 🐛 Troubleshooting

### Проблема: Письма не обрабатываются

```bash
# Проверить подключение к IMAP
docker compose logs app | grep -i "imap"

# Проверить allowed_senders в config.yaml
```

### Проблема: Ошибки БД

```bash
# Проверить подключение к БД
docker compose logs app | grep -i "database"

# Проверить миграции
docker compose logs app | grep -i "migration"
```

### Проблема: Парсинг не работает

```bash
# Включить debug логи
# В config.yaml установите log_level: debug

# Проверить логи парсинга
docker compose logs app | grep -i "parsing"
```

## 📄 Лицензия

MIT License - см. файл [LICENSE](LICENSE)

## 🤝 Вклад в проект

Проект готов к продакшену. Для изменений:

1. Создайте feature branch
2. Внесите изменения
3. Напишите тесты
4. Создайте Pull Request

## 📞 Контакты

При возникновении вопросов:
- Проверьте [документацию](docs/INDEX.md)
- Изучите [QUICK_REFERENCE.md](QUICK_REFERENCE.md)
- Проверьте [DEPLOYMENT.md](DEPLOYMENT.md)

---

**Версия:** 1.0.0
**Статус:** Production Ready ✅
**Дата релиза:** 17.03.2026
