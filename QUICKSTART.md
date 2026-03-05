# Quick Start Guide

Быстрое руководство для запуска сервиса обработки email с Excel файлами.

## Шаг 1: Подготовка (5 минут)

### 1.1 Клонирование проекта

```bash
cd ~
git clone <repository-url> etalon-nomenclature
cd etalon-nomenclature
```

### 1.2 Создание конфигурации

```bash
cp config.example.yaml config.yaml
nano config.yaml
```

Минимальная конфигурация:

```yaml
poll_interval: 1m

database:
  dsn: "postgresql://user:password@host:5432/database?sslmode=verify-full"
  ssl_root_cert: "$HOME/.cloud-certs/root.crt"

mailboxes:
  - email: "your-email@domain.com"
    password: "your-password"
    host: "mail.hosting.reg.ru"
    port: 993
```

### 1.3 Настройка SSL сертификата

```bash
mkdir -p ~/.cloud-certs
# Поместите root.crt в ~/.cloud-certs/
chmod 600 ~/.cloud-certs/root.crt
```

## Шаг 2: База данных (2 минуты)

### 2.1 Применение миграций

```bash
# Установите psql если нужно
sudo apt install postgresql-client -y

# Примените миграции
./scripts/apply-migrations.sh
```

Или вручную:

```bash
export PGSSLROOTCERT="$HOME/.cloud-certs/root.crt"
psql "postgresql://user:password@host:5432/database?sslmode=verify-full" \
  -f migrations/001_init.sql
```

### 2.2 Проверка таблиц

```bash
psql "your-dsn-here" -c "\dt"
```

Ожидаемый результат:
```
 etalon_nomenclature
 processed_emails
```

## Шаг 3: Запуск (выберите способ)

### Вариант A: Локальный запуск

```bash
# Установка зависимостей
go mod download

# Запуск
go run cmd/app/main.go
```

### Вариант B: Docker (рекомендуется)

```bash
# Сборка
docker compose build

# Запуск
docker compose up -d

# Просмотр логов
docker compose logs -f app
```

## Шаг 4: Проверка работы (3 минуты)

### 4.1 Проверка логов

```bash
# Для Docker
docker compose logs -f app

# Для локального запуска
# Смотрите вывод в терминале
```

Ожидаемые логи:
```json
{"level":"info","msg":"Starting Etalon Nomenclature Service"}
{"level":"info","msg":"Database connection established"}
{"level":"info","msg":"Processor started"}
{"level":"info","msg":"Starting email processing cycle"}
```

### 4.2 Отправка тестового письма

1. Отправьте email с Excel файлом на настроенный адрес
2. Excel должен содержать колонки: Артикул, Марка, Размер и Модель, Номенклатура, МРЦ
3. Подождите до 1 минуты

### 4.3 Проверка обработки

**Проверка в логах:**
```bash
docker compose logs app | grep "Processing email"
```

**Проверка в базе данных:**
```bash
psql "your-dsn-here" -c "SELECT COUNT(*) FROM etalon_nomenclature;"
psql "your-dsn-here" -c "SELECT * FROM processed_emails ORDER BY processed_at DESC LIMIT 5;"
```

## Управление сервисом

### Docker команды

```bash
# Запуск
docker compose up -d

# Остановка
docker compose down

# Перезапуск
docker compose restart app

# Логи (follow)
docker compose logs -f app

# Логи (последние 100 строк)
docker compose logs --tail=100 app

# Статус
docker compose ps

# Ресурсы
docker stats etalon-nomenclature
```

### Makefile команды

```bash
# Показать все команды
make help

# Локальная сборка
make build

# Локальный запуск
make run

# Тесты
make test

# Docker сборка
make docker-build

# Docker запуск
make docker-up

# Docker остановка
make docker-down

# Docker логи
make docker-logs

# Применение миграций
make migrate
```

## Проверка подключений

Используйте скрипт для проверки:

```bash
./scripts/test-connection.sh
```

Ожидаемый результат:
```
✅ config.yaml found
✅ SSL certificate found
✅ IMAP port 993 is accessible
✅ PostgreSQL connection successful
```

## Troubleshooting

### Проблема: Сервис не запускается

```bash
# Проверьте конфигурацию
cat config.yaml

# Проверьте логи
docker compose logs app

# Проверьте конфигурацию docker-compose
docker compose config
```

### Проблема: Не подключается к IMAP

```bash
# Проверьте доступность порта
nc -zv mail.hosting.reg.ru 993

# Проверьте логин/пароль в config.yaml
grep -A 5 "mailboxes:" config.yaml
```

### Проблема: Не подключается к PostgreSQL

```bash
# Проверьте сертификат
ls -la ~/.cloud-certs/root.crt

# Проверьте подключение
PGSSLROOTCERT=~/.cloud-certs/root.crt psql "your-dsn" -c "SELECT 1;"

# Проверьте переменную окружения в контейнере
docker compose exec app env | grep PGSSLROOTCERT
```

### Проблема: Письма не обрабатываются

```bash
# Проверьте, не обработано ли письмо ранее
psql "your-dsn" -c "SELECT * FROM processed_emails WHERE message_id LIKE '%subject%';"

# Удалите Message-ID для повторной обработки
psql "your-dsn" -c "DELETE FROM processed_emails WHERE message_id = '<message-id>';"

# Проверьте формат Excel файла
# Должны быть колонки: Артикул, Марка, Размер и Модель, Номенклатура, МРЦ
```

## Примеры запросов к БД

### Просмотр последних 10 записей

```sql
SELECT * FROM etalon_nomenclature
ORDER BY created_at DESC
LIMIT 10;
```

### Количество записей по маркам

```sql
SELECT brand, COUNT(*) as count
FROM etalon_nomenclature
GROUP BY brand
ORDER BY count DESC;
```

### Записи за сегодня

```sql
SELECT COUNT(*)
FROM etalon_nomenclature
WHERE created_at::date = CURRENT_DATE;
```

### Обработанные письма за сегодня

```sql
SELECT * FROM processed_emails
WHERE processed_at::date = CURRENT_DATE
ORDER BY processed_at DESC;
```

### Удаление старых записей (осторожно!)

```sql
-- Удалить записи старше 30 дней
DELETE FROM etalon_nomenclature
WHERE created_at < NOW() - INTERVAL '30 days';

-- Удалить обработанные письма старше 90 дней
DELETE FROM processed_emails
WHERE processed_at < NOW() - INTERVAL '90 days';
```

## Следующие шаги

1. **Настройте мониторинг логов**
   ```bash
   # Установите регулярную проверку
   watch -n 60 'docker compose logs --tail=20 app'
   ```

2. **Настройте автоматический backup**
   ```bash
   # Создайте cron job для backup конфигурации
   0 2 * * * tar -czf ~/backups/etalon-config-$(date +\%Y\%m\%d).tar.gz ~/etalon-nomenclature/config.yaml ~/.cloud-certs/
   ```

3. **Добавьте больше почтовых ящиков**
   ```bash
   nano config.yaml
   # Добавьте новые mailbox записи
   docker compose restart app
   ```

4. **Настройте алерты**
   - Мониторинг логов с ключевыми словами: ERROR, Failed, Panic
   - Проверка количества обработанных писем
   - Мониторинг использования ресурсов

## Полезные ссылки

- [README.md](README.md) — полная документация
- [DEPLOYMENT.md](DEPLOYMENT.md) — детальное руководство по развертыванию
- [STRUCTURE.md](STRUCTURE.md) — архитектура проекта

## Поддержка

При возникновении проблем:
1. Проверьте логи
2. Изучите раздел Troubleshooting
3. Создайте issue в репозитории
