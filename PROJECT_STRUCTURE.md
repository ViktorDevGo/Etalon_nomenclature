# Etalon Nomenclature - Структура проекта

## 📁 Финальная структура (Production Ready)

```
etalon-nomenclature/
├── 📄 README.md                          # Основная документация
├── 📄 PRODUCTION.md                      # Руководство по продакшн деплою
├── 📄 DEPLOYMENT.md                      # Детальные инструкции по деплою
├── 📄 QUICK_REFERENCE.md                 # Быстрая справка
├── 📄 PRODUCTION_CHECKLIST.md            # Чеклист перед деплоем
├── 📄 FILE_PROCESSING_LOGIC.md           # Логика обработки файлов
├── 📄 DEDUPLICATION_LOGIC.md             # Логика дедупликации
├── 📄 MIGRATION_TYRES_PRICES_STOCK.md    # Миграция шин
├── 📄 MIGRATION_RIMS_PRICES_STOCK.md     # Миграция дисков
├── 📄 MIGRATION_RENAME_TABLE.md          # Переименование таблицы
├── 📄 MIGRATION_ADD_ISIMPORT_1C.md       # Добавление isimport_1С
├── 📄 LICENSE                            # MIT лицензия
├── 📄 Makefile                           # Build команды
├── 📄 go.mod                             # Go зависимости
├── 📄 go.sum                             # Go checksums
├── 📄 docker-compose.yml                 # Docker Compose конфигурация
├── 📄 .dockerignore                      # Docker ignore файлы
├── 📄 .gitignore                         # Git ignore файлы
├── 📄 .editorconfig                      # Editor конфигурация
├── 📄 .golangci.yml                      # Linter конфигурация
├── 📄 config.example.yaml                # Пример конфигурации
├── 📄 .env.example                       # Пример env файлов
├── 📄 cleanup_project.sh                 # Скрипт очистки проекта
│
├── 📁 docs/                              # Документация
│   └── 📄 INDEX.md                       # Индекс всей документации
│
├── 📁 cmd/                               # Точки входа
│   └── 📁 app/
│       └── 📄 main.go                    # Main приложения
│
├── 📁 internal/                          # Внутренний код
│   ├── 📁 db/
│   │   └── 📄 postgres.go                # PostgreSQL клиент + миграции
│   ├── 📁 imap/
│   │   └── 📄 client.go                  # IMAP клиент
│   ├── 📁 parser/
│   │   ├── 📄 excel.go                   # Парсер номенклатуры МРЦ
│   │   ├── 📄 price.go                   # Парсер цен шин
│   │   ├── 📄 disk.go                    # Парсер цен/номенклатуры дисков
│   │   ├── 📄 detector.go                # Детектор типов файлов
│   │   ├── 📄 smart_row_parser.go        # Умный парсер строк
│   │   ├── 📄 xls_converter.go           # Конвертер XLS
│   │   ├── 📄 libreoffice_converter.go   # LibreOffice конвертер
│   │   ├── 📄 excel_test.go              # Тесты парсера
│   │   ├── 📄 price_test.go              # Тесты цен
│   │   └── 📄 detector_test.go           # Тесты детектора
│   └── 📁 service/
│       └── 📄 processor.go               # Главная бизнес-логика
│
├── 📁 migrations/                        # SQL миграции
│   ├── 📄 001_init.sql                   # Инициализация БД
│   ├── 📄 002_replace_price_tires.sql    # Замена price_tires
│   ├── 📄 003_replace_price_disks.sql    # Замена price_disks
│   ├── 📄 004_rename_etalon_nomenclature.sql  # Переименование
│   └── 📄 005_add_isimport_1C.sql        # Добавление isimport_1С
│
├── 📁 docker/                            # Docker файлы
│   └── 📄 Dockerfile                     # Dockerfile для продакшена
│
└── 📁 config/                            # Конфигурация (пример)
    └── 📄 config.example.yaml            # Пример конфига
```

## 🚫 Файлы НЕ в Git (.gitignore)

```
├── 📄 config.yaml                        # ⚠️ Содержит пароли!
├── 📄 .env                               # ⚠️ Секретные переменные
├── 📄 app                                # Скомпилированный бинарник
├── 📄 etalon_nomenclature                # Скомпилированный бинарник
├── 📄 *.log                              # Лог файлы
├── 📁 certs/*.crt                        # ⚠️ SSL сертификаты
└── 📁 .claude/                           # Claude AI настройки
```

## 📦 Таблицы БД (Production)

```sql
-- Номенклатура с МРЦ
mrc_etalon (
    id, article, brand, type, size_model,
    nomenclature, mrc, isimport, isimport_1С,
    created_at, email_date
)

-- Цены и остатки шин
tyres_prices_stock (
    id, cae, price, stock, warehouse_name,
    provider, isimport, created_at, email_date
)

-- Цены и остатки дисков
rims_prices_stock (
    id, cae, price, stock, warehouse_name,
    provider, isimport, created_at, email_date
)

-- Номенклатура дисков (ЗАПАСКА)
nomenclature_rims (
    id, cae, name, width, diameter,
    bolts_count, bolts_spacing, et, dia,
    model, brand, color, isimport,
    created_at, email_date
)

-- Обработанные письма
processed_emails (
    id, message_id, email_date, processed_at
)
```

## 🔧 Основные команды

```bash
# Разработка
go run cmd/app/main.go
go test ./...
go build ./...

# Продакшн
docker compose build
docker compose up -d
docker compose logs -f app
docker compose down

# Обновление
git pull
docker compose down
docker compose build
docker compose up -d
```

## 📊 Метрики проекта

- **Язык:** Go 1.22
- **Строк кода:** ~5000
- **Файлов:** ~30
- **Таблиц БД:** 5
- **Миграций:** 5
- **Документов:** 13
- **Поставщиков:** 3 (ЗАПАСКА, БИГМАШИН, БРИНЕКС)

## ✅ Production Ready

- [x] Код скомпилирован без ошибок
- [x] Тесты написаны и проходят
- [x] Документация полная
- [x] Docker образ оптимизирован
- [x] Миграции протестированы
- [x] Логирование настроено
- [x] Graceful shutdown реализован
- [x] Безопасность проверена
- [x] .gitignore настроен
- [x] README актуален

---

**Статус:** ✅ Production Ready
**Версия:** 1.0.0
**Дата:** 17.03.2026
