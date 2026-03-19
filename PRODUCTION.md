# Production Deployment Guide

Полное руководство по развертыванию Etalon Nomenclature Service на продакшн.

## 📋 Pre-Deployment Checklist

### ✅ Код и конфигурация

- [ ] Все изменения закоммичены в Git
- [ ] Тесты пройдены (`go test ./...`)
- [ ] Код скомпилирован без ошибок (`go build ./...`)
- [ ] `config.yaml` настроен для продакшена (не в Git!)
- [ ] `.env` файл отсутствует или настроен
- [ ] SSL сертификаты размещены (если нужны)

### ✅ База данных

- [ ] PostgreSQL 14+ установлен и доступен
- [ ] Пользователь БД создан с правами CREATE, INSERT, UPDATE, DELETE
- [ ] DSN строка подключения корректна
- [ ] SSL подключение настроено (если используется)
- [ ] Firewall разрешает подключения к БД

### ✅ Email

- [ ] IMAP доступ к почтовому ящику настроен
- [ ] Логин/пароль корректны
- [ ] Порт 993 открыт (IMAPS)
- [ ] Список `allowed_senders` заполнен

### ✅ Сервер

- [ ] Docker и Docker Compose установлены
- [ ] Достаточно места на диске (минимум 10GB)
- [ ] Достаточно RAM (минимум 512MB для приложения)
- [ ] Порты не заняты другими сервисами

---

## 🚀 Deployment Steps

### 1. Подготовка сервера

```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка Docker (если еще не установлен)
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Установка Docker Compose
sudo apt install docker-compose-plugin -y

# Проверка установки
docker --version
docker compose version
```

### 2. Клонирование репозитория

```bash
# Переход в домашнюю директорию
cd ~

# Клонирование (если первый раз)
git clone <repository-url> etalon-nomenclature

# Или обновление (если уже есть)
cd etalon-nomenclature
git pull
```

### 3. Настройка конфигурации

```bash
# Создание config.yaml из примера
cp config.example.yaml config.yaml

# Редактирование конфигурации
nano config.yaml
```

**Минимальный config.yaml:**

```yaml
poll_interval: 30m

database:
  dsn: "postgresql://gen_user:PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

mailboxes:
  - email: "zakupki@etalon-shina.ru"
    password: "EMAIL_PASSWORD"
    host: "mail.hosting.reg.ru"
    port: 993

allowed_senders:
  - "pna@sibzapaska.ru"
```

### 4. Размещение SSL сертификатов (опционально)

```bash
# Создание директории для сертификатов
mkdir -p certs

# Копирование сертификата (если используется SSL verify-full)
cp /path/to/root.crt certs/

# Обновление config.yaml
nano config.yaml
# Добавить: ssl_root_cert: "/app/certs/root.crt"
```

### 5. Проверка docker-compose.yml

```bash
# Просмотр конфигурации
cat docker-compose.yml

# Проверка синтаксиса
docker compose config
```

### 6. Сборка и запуск

```bash
# Сборка образа
docker compose build

# Запуск в фоновом режиме
docker compose up -d

# Просмотр логов
docker compose logs -f app
```

### 7. Проверка работоспособности

```bash
# Статус контейнеров
docker compose ps

# Логи (должны быть без FATAL ошибок)
docker compose logs app | grep -i error

# Проверка подключения к БД
docker compose logs app | grep -i "database"

# Проверка подключения к email
docker compose logs app | grep -i "imap"

# Проверка обработки писем
docker compose logs app | grep -i "processing"
```

---

## 🔄 Updates and Maintenance

### Обновление до новой версии

```bash
# Остановка сервиса
cd ~/etalon-nomenclature
docker compose down

# Получение обновлений
git pull

# Пересборка образа
docker compose build

# Запуск обновленной версии
docker compose up -d

# Проверка логов
docker compose logs -f app
```

### Просмотр логов

```bash
# Живые логи
docker compose logs -f app

# Последние 100 строк
docker compose logs --tail=100 app

# Логи с временными метками
docker compose logs -f --timestamps app

# Поиск ошибок
docker compose logs app | grep -i error
```

### Перезапуск сервиса

```bash
# Мягкий перезапуск (без пересборки)
docker compose restart app

# Полный перезапуск (с пересборкой)
docker compose down
docker compose up -d --build
```

### Очистка ресурсов

```bash
# Удаление неиспользуемых образов
docker system prune -a

# Удаление неиспользуемых volumes
docker volume prune

# Полная очистка (осторожно!)
docker system prune -a --volumes
```

---

## 🗄️ Database Management

### Проверка состояния БД

```bash
# Подключение к БД
PGPASSWORD='your_password' psql "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"
```

```sql
-- Проверка таблиц
SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;

-- Количество записей
SELECT 'mrc_etalon' as table, COUNT(*) FROM mrc_etalon
UNION ALL SELECT 'tyres_prices_stock', COUNT(*) FROM tyres_prices_stock
UNION ALL SELECT 'rims_prices_stock', COUNT(*) FROM rims_prices_stock
UNION ALL SELECT 'nomenclature_rims', COUNT(*) FROM nomenclature_rims
UNION ALL SELECT 'processed_emails', COUNT(*) FROM processed_emails;

-- Последние обработанные письма
SELECT message_id, email_date, processed_at
FROM processed_emails
ORDER BY processed_at DESC
LIMIT 10;
```

### Очистка обработанных писем

```sql
-- Удалить записи старше 30 дней
DELETE FROM processed_emails
WHERE processed_at < NOW() - INTERVAL '30 days';

-- Удалить все записи (письма будут обработаны повторно!)
TRUNCATE processed_emails;
```

### Backup БД

```bash
# Создание backup
PGPASSWORD='your_password' pg_dump \
  "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require" \
  > backup_$(date +%Y%m%d_%H%M%S).sql

# Восстановление из backup
PGPASSWORD='your_password' psql \
  "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require" \
  < backup_20260317_120000.sql
```

---

## 🔐 Security Best Practices

### Защита конфигурации

```bash
# Права доступа только для владельца
chmod 600 config.yaml

# Проверка прав
ls -la config.yaml
# Должно быть: -rw------- (600)
```

### Ротация паролей

1. Обновите пароль в БД
2. Обновите `config.yaml`
3. Перезапустите сервис: `docker compose restart app`

### SSL сертификаты

```bash
# Проверка срока действия сертификата
openssl x509 -in certs/root.crt -noout -dates

# Обновление сертификата
cp /path/to/new/root.crt certs/
docker compose restart app
```

---

## 🐛 Troubleshooting

### Проблема: Контейнер не запускается

```bash
# Проверить логи
docker compose logs app

# Проверить конфигурацию
docker compose config

# Проверить порты
sudo netstat -tulpn | grep -E ':(993|5432)'
```

### Проблема: Нет подключения к БД

```bash
# Проверить DSN в config.yaml
# Проверить firewall
sudo ufw status

# Тест подключения
telnet c37e696087932476c61fd621.twc1.net 5432
```

### Проблема: Письма не обрабатываются

```bash
# Проверить логи IMAP
docker compose logs app | grep -i imap

# Проверить allowed_senders в config.yaml

# Тест IMAP подключения (вручную)
openssl s_client -connect mail.hosting.reg.ru:993
```

### Проблема: Высокое использование ресурсов

```bash
# Проверить использование
docker stats

# Ограничить ресурсы в docker-compose.yml
deploy:
  resources:
    limits:
      memory: 512M
      cpus: '1.0'
```

---

## 📊 Monitoring

### Основные метрики

```sql
-- Статистика по таблицам
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Активность за последний час
SELECT
    DATE_TRUNC('hour', processed_at) as hour,
    COUNT(*) as emails_processed
FROM processed_emails
WHERE processed_at > NOW() - INTERVAL '24 hours'
GROUP BY hour
ORDER BY hour DESC;

-- Статистика по поставщикам
SELECT
    provider,
    COUNT(*) as records,
    MAX(created_at) as last_update
FROM tyres_prices_stock
GROUP BY provider
ORDER BY last_update DESC;
```

### Логи приложения

```bash
# Количество обработанных писем сегодня
docker compose logs app | grep "Successfully processed email" | grep $(date +%Y-%m-%d) | wc -l

# Ошибки за последние 24 часа
docker compose logs --since 24h app | grep -i error

# Статистика парсинга
docker compose logs app | grep "Parsed.*attachment"
```

---

## 🔄 Rollback Plan

### В случае проблем после обновления

```bash
# 1. Остановить сервис
docker compose down

# 2. Откатить код к предыдущей версии
git log --oneline -5  # Найти хеш предыдущего коммита
git checkout <previous-commit-hash>

# 3. Пересобрать и запустить
docker compose build
docker compose up -d

# 4. Проверить логи
docker compose logs -f app
```

### Откат миграций БД

Если миграция вызвала проблемы, см. соответствующий файл MIGRATION_*.md для инструкций по откату.

---

## 📞 Support

### Полезные ссылки

- [README.md](README.md) - Общая информация
- [docs/INDEX.md](docs/INDEX.md) - Индекс документации
- [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - Быстрая справка
- [DEPLOYMENT.md](DEPLOYMENT.md) - Детальный деплой

### Логи для отправки при проблемах

```bash
# Сохранить логи в файл
docker compose logs app > app_logs_$(date +%Y%m%d_%H%M%S).txt

# Конфигурация Docker
docker compose config > docker_config.txt

# Системная информация
docker info > docker_info.txt
```

---

**Версия:** 1.0.0
**Дата:** 17.03.2026
**Статус:** Production Ready ✅
