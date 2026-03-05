# Project Summary

## 📋 Общая информация

**Название:** Etalon Nomenclature Service
**Версия:** 1.0.0
**Язык:** Go 1.22+
**Статус:** Production Ready ✅

## 🎯 Назначение

Автоматизированный сервис для обработки email с Excel вложениями и сохранения данных в PostgreSQL.

### Основные функции

1. **Автоматическая проверка почты** — каждую минуту
2. **Парсинг Excel** — все вкладки, все строки
3. **Запись в PostgreSQL** — транзакционно и безопасно
4. **Защита от дубликатов** — по Message-ID
5. **Поддержка множественных ящиков** — без ограничений

## 🏗️ Архитектура

```
Email (IMAP) → Parser (Excel) → Database (PostgreSQL)
     ↑              ↓                    ↓
  REG.RU      Streaming Read      Transactional
   SSL           Memory             SSL/TLS
```

### Компоненты

| Компонент | Технология | Описание |
|-----------|------------|----------|
| **IMAP Client** | emersion/go-imap | Получение писем с REG.RU |
| **Excel Parser** | xuri/excelize | Streaming парсинг .xlsx |
| **Database** | lib/pq | PostgreSQL с SSL |
| **Logger** | uber/zap | Structured JSON logging |
| **Config** | yaml.v3 | YAML конфигурация |

## 📦 Структура файлов

```
.
├── cmd/app/main.go              # Точка входа
├── config/                      # Конфигурация
├── internal/
│   ├── db/                      # PostgreSQL
│   ├── imap/                    # Email клиент
│   ├── parser/                  # Excel парсер
│   └── service/                 # Бизнес-логика
├── migrations/                  # SQL миграции
├── docker/                      # Dockerfile
├── scripts/                     # Утилиты
├── docker-compose.yml           # Docker Compose
└── [Документация]
```

## 🚀 Быстрый старт

### 1. Настройка (5 мин)

```bash
# Клонирование
git clone <repo>
cd etalon-nomenclature

# Конфигурация
cp config.example.yaml config.yaml
nano config.yaml

# SSL сертификат
mkdir -p ~/.cloud-certs
# Поместите root.crt в ~/.cloud-certs/
```

### 2. База данных (2 мин)

```bash
# Применить миграции
./scripts/apply-migrations.sh

# Или вручную
psql "your-dsn" -f migrations/001_init.sql
```

### 3. Запуск (1 мин)

**Docker (рекомендуется):**
```bash
docker compose up -d
docker compose logs -f app
```

**Локально:**
```bash
go run cmd/app/main.go
```

## 📊 Схема базы данных

### Таблица: etalon_nomenclature

```sql
CREATE TABLE etalon_nomenclature (
    id SERIAL PRIMARY KEY,
    article TEXT,           -- Артикул
    brand TEXT,             -- Марка
    type TEXT,              -- Тип (опционально)
    size_model TEXT,        -- Размер и Модель
    nomenclature TEXT,      -- Номенклатура
    mrc NUMERIC,            -- МРЦ (цена)
    isimport INTEGER DEFAULT 0,  -- Флаг импорта
    created_at TIMESTAMP DEFAULT now()
);
```

### Таблица: processed_emails

```sql
CREATE TABLE processed_emails (
    id SERIAL PRIMARY KEY,
    message_id TEXT UNIQUE,      -- Message-ID письма
    processed_at TIMESTAMP DEFAULT now()
);
```

## 🔧 Конфигурация

### Пример config.yaml

```yaml
poll_interval: 1m

database:
  dsn: "postgresql://user:pass@host:5432/db?sslmode=verify-full"
  ssl_root_cert: "/app/certs/root.crt"

mailboxes:
  - email: "box1@domain.com"
    password: "password1"
    host: "mail.hosting.reg.ru"
    port: 993

  - email: "box2@domain.com"
    password: "password2"
    host: "mail.hosting.reg.ru"
    port: 993
```

## 📝 Формат Excel файла

### Обязательные колонки

1. **Артикул** — код товара
2. **Марка** — бренд
3. **Размер и Модель** — характеристики
4. **Номенклатура** — название
5. **МРЦ** — цена

### Опциональные колонки

- **Тип** — категория (если отсутствует, будет пустое значение)

### Примечания

- Обрабатываются **все вкладки** в файле
- Заголовки определяются автоматически
- Поддержка чисел с запятыми (1 000,50)
- Streaming парсинг — низкое потребление памяти

## 🔒 Безопасность

### Реализованные меры

✅ SSL/TLS для PostgreSQL
✅ Non-root пользователь в Docker
✅ Prepared statements для SQL
✅ Проверка размера файлов (макс 10 MB)
✅ Нет паролей в логах
✅ Нет паролей в коде
✅ Валидация входных данных

## 🐛 Обработка ошибок

### Retry логика

- **IMAP ошибки:** 3 попытки с задержкой 5 сек
- **Автоматическое переподключение**
- **Логирование всех попыток**

### Транзакции

- **Атомарность:** все или ничего
- **Rollback при ошибке**
- **Message-ID не сохраняется** при ошибке
- **Повторная обработка** при следующем цикле

### Panic Recovery

- **Перехват panic**
- **Stack trace в логи**
- **Сервис продолжает работу**

## 📈 Production Features

### Устойчивость

- ✅ Graceful shutdown
- ✅ Context cancellation
- ✅ Connection pooling
- ✅ Health checks
- ✅ Auto-restart (Docker)

### Мониторинг

- ✅ Structured JSON logs
- ✅ Timestamps на каждом событии
- ✅ Request tracing
- ✅ Error tracking
- ✅ Resource monitoring

### Производительность

- ✅ Streaming парсинг
- ✅ Connection pooling
- ✅ Batch inserts (транзакции)
- ✅ Efficient indexing
- ✅ Resource limits

## 🔍 Мониторинг и логи

### Ключевые метрики

```bash
# Обработанные письма
grep "Successfully processed email" logs

# Ошибки
grep "ERROR" logs

# Парсинг статистика
grep "Parsed sheet" logs

# IMAP ретраи
grep "Retrying IMAP connection" logs
```

### Запросы к БД

```sql
-- Записей за сегодня
SELECT COUNT(*) FROM etalon_nomenclature
WHERE created_at::date = CURRENT_DATE;

-- Обработанных писем за сегодня
SELECT COUNT(*) FROM processed_emails
WHERE processed_at::date = CURRENT_DATE;

-- По маркам
SELECT brand, COUNT(*) FROM etalon_nomenclature
GROUP BY brand ORDER BY COUNT(*) DESC;
```

## 🧪 Тестирование

### Запуск тестов

```bash
# Все тесты
make test

# С покрытием
make test-coverage

# Конкретный пакет
go test -v ./internal/parser
```

### Coverage

```
config/       — 85% покрытие
internal/db/  — 40% покрытие (integration tests needed)
internal/parser/ — 80% покрытие
```

## 📚 Документация

| Файл | Назначение |
|------|------------|
| [README.md](README.md) | Основная документация |
| [QUICKSTART.md](QUICKSTART.md) | Быстрый старт за 8 минут |
| [DEPLOYMENT.md](DEPLOYMENT.md) | Production развертывание |
| [STRUCTURE.md](STRUCTURE.md) | Архитектура проекта |
| [CHANGELOG.md](CHANGELOG.md) | История версий |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Гайд для контрибьюторов |
| [PROJECT_SUMMARY.md](PROJECT_SUMMARY.md) | Этот файл |

## 🛠️ Полезные команды

### Makefile

```bash
make help          # Показать все команды
make build         # Собрать бинарь
make run           # Запустить локально
make test          # Запустить тесты
make docker-build  # Собрать Docker образ
make docker-up     # Запустить в Docker
make docker-logs   # Показать логи
make migrate       # Применить миграции
```

### Docker

```bash
docker compose up -d               # Запуск
docker compose down                # Остановка
docker compose restart app         # Перезапуск
docker compose logs -f app         # Логи (follow)
docker compose logs --tail=100 app # Последние 100 строк
docker stats etalon-nomenclature   # Ресурсы
```

### Database

```bash
# Подключение
psql "postgresql://..."

# Таблицы
\dt

# Последние записи
SELECT * FROM etalon_nomenclature ORDER BY created_at DESC LIMIT 10;

# Статистика
SELECT COUNT(*) FROM etalon_nomenclature;
SELECT COUNT(*) FROM processed_emails;
```

## 🚦 Статус проекта

### ✅ Реализовано

- [x] IMAP клиент с retry логикой
- [x] Excel парсер (streaming)
- [x] PostgreSQL интеграция с SSL
- [x] Множественные почтовые ящики
- [x] Защита от дубликатов
- [x] Graceful shutdown
- [x] Panic recovery
- [x] Structured logging
- [x] Docker support
- [x] Health checks
- [x] Миграции
- [x] Тесты
- [x] Полная документация

### 🔄 Планируется (v2.0)

- [ ] Prometheus metrics
- [ ] REST API
- [ ] Dashboard UI
- [ ] Webhook notifications
- [ ] Support для .xls
- [ ] Email архивирование
- [ ] Configurable search criteria

## 🎓 Использованные технологии

### Core

- **Go 1.22** — язык программирования
- **PostgreSQL 14+** — база данных
- **Docker** — контейнеризация

### Libraries

- `github.com/emersion/go-imap` — IMAP протокол
- `github.com/xuri/excelize/v2` — Excel парсинг
- `github.com/lib/pq` — PostgreSQL драйвер
- `go.uber.org/zap` — логирование
- `gopkg.in/yaml.v3` — YAML конфиг

### Tools

- Docker Compose — оркестрация
- Make — автоматизация
- golangci-lint — линтинг
- Go test — тестирование

## 📞 Поддержка

### Проблемы

1. Проверьте [README.md](README.md) — раздел Troubleshooting
2. Проверьте [DEPLOYMENT.md](DEPLOYMENT.md) — детали развертывания
3. Создайте Issue с:
   - Описанием проблемы
   - Шагами воспроизведения
   - Логами
   - Версией Go/Docker

### Вопросы

- Откройте Discussion
- Создайте Issue с вопросом
- Свяжитесь с мейнтейнерами

## 📄 Лицензия

MIT License — см. [LICENSE](LICENSE)

---

**Версия:** 1.0.0
**Дата:** 2026-03-05
**Статус:** Production Ready ✅
