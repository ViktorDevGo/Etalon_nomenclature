# Deployment Guide

## 🚀 Quick Start (Environment Variables - Recommended for Cloud)

**Use this method if:**
- Your deployment platform doesn't allow volume mounts
- You're deploying to a managed service or CI/CD pipeline
- You want a more secure, cloud-native approach

### Step 1: Prepare environment variables

```bash
# Copy the example file
cp .env.example .env

# Edit the .env file
nano .env
```

### Step 2: Configure your environment

```bash
# Database connection
DATABASE_DSN=postgresql://gen_user:YOUR_PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full

# Encode SSL certificate (required for secure PostgreSQL connection)
cat ~/.cloud-certs/root.crt | base64 | tr -d '\n'
# Copy the output and set it as:
PGSSLROOTCERT_BASE64=LS0tLS1CRUdJTi...

# Configure mailboxes (JSON format)
MAILBOXES_JSON='[{"email":"your-email@domain.com","password":"yourpassword","host":"mail.hosting.reg.ru","port":993}]'

# Application settings
POLL_INTERVAL=1m
TZ=Europe/Moscow
```

### Step 3: Deploy

```bash
# Build and run
docker compose build
docker compose up -d

# Check logs
docker compose logs -f app
```

### Adding multiple mailboxes

Edit your `.env` file and add multiple mailbox objects in the JSON array:

```bash
MAILBOXES_JSON='[
  {"email":"email1@domain.com","password":"pass1","host":"mail.hosting.reg.ru","port":993},
  {"email":"email2@domain.com","password":"pass2","host":"mail.hosting.reg.ru","port":993},
  {"email":"email3@domain.com","password":"pass3","host":"mail.hosting.reg.ru","port":993}
]'
```

**Note:** Make sure to format it as a single line when setting in `.env`.

---

## 📋 Traditional Deployment (Volume Mounts)

**Use this method if:**
- You have full control over your server
- Your platform allows volume mounts
- You prefer file-based configuration

## Предварительные требования

- Сервер с Ubuntu/Debian
- Docker и Docker Compose установлены
- Доступ к PostgreSQL БД на Timeweb Cloud
- SSL сертификат для PostgreSQL
- Доступ к почтовым ящикам на REG.RU

## Шаг 1: Подготовка сервера

### Установка Docker

```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Установка Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Добавление пользователя в группу docker
sudo usermod -aG docker $USER

# Перелогиньтесь для применения изменений
```

### Установка Docker Compose

```bash
sudo apt install docker-compose-plugin -y

# Проверка
docker compose version
```

## Шаг 2: Настройка проекта

### Клонирование репозитория

```bash
cd ~
git clone <your-repository-url> etalon-nomenclature
cd etalon-nomenclature
```

### Настройка SSL сертификата

```bash
# Создание директории для сертификатов
mkdir -p ~/.cloud-certs

# Загрузка сертификата (замените на ваш способ получения)
# Вариант 1: копирование с локальной машины
scp /path/to/root.crt user@server:~/.cloud-certs/

# Вариант 2: создание вручную
nano ~/.cloud-certs/root.crt
# Вставьте содержимое сертификата

# Установка прав
chmod 600 ~/.cloud-certs/root.crt
```

### Создание конфигурации

```bash
# Копирование примера
cp config.example.yaml config.yaml

# Редактирование конфигурации
nano config.yaml
```

Заполните настройки:

```yaml
poll_interval: 1m

database:
  dsn: "postgresql://gen_user:YOUR_PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full"
  ssl_root_cert: "/app/certs/root.crt"
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

mailboxes:
  - email: "your-email@domain.com"
    password: "your-password"
    host: "mail.hosting.reg.ru"
    port: 993
```

## Шаг 3: Подготовка базы данных

### Применение миграций

```bash
# Установка psql (если нужно)
sudo apt install postgresql-client -y

# Применение миграций
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full" \
  -f migrations/001_init.sql
```

### Проверка таблиц

```bash
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full" \
  -c "\dt"
```

Вы должны увидеть:
- `etalon_nomenclature`
- `processed_emails`

## Шаг 4: Сборка и запуск

### Сборка Docker образа

```bash
docker compose build
```

### Запуск сервиса

```bash
docker compose up -d
```

### Проверка статуса

```bash
# Статус контейнера
docker compose ps

# Логи
docker compose logs -f app

# Остановка логов: Ctrl+C
```

## Шаг 5: Проверка работы

### Мониторинг логов

```bash
# Последние 100 строк
docker compose logs --tail=100 app

# Follow режим
docker compose logs -f app
```

### Проверка обработки писем

Отправьте тестовое письмо с Excel файлом на настроенный email и проверьте:

1. **Логи сервиса**:
```bash
docker compose logs -f app | grep "Processing email"
```

2. **Данные в БД**:
```bash
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -c "SELECT COUNT(*) FROM etalon_nomenclature;"
```

3. **Обработанные письма**:
```bash
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -c "SELECT * FROM processed_emails ORDER BY processed_at DESC LIMIT 10;"
```

## Шаг 6: Настройка автозапуска

Docker Compose с `restart: unless-stopped` автоматически запустит сервис после перезагрузки.

### Проверка после перезагрузки

```bash
# Перезагрузка сервера
sudo reboot

# После перезагрузки проверьте
docker compose ps
docker compose logs --tail=50 app
```

## Обновление сервиса

### Обновление кода

```bash
cd ~/etalon-nomenclature

# Получение изменений
git pull

# Пересборка и перезапуск
docker compose down
docker compose build
docker compose up -d

# Проверка
docker compose logs -f app
```

### Обновление конфигурации

```bash
# Редактирование конфигурации
nano config.yaml

# Перезапуск сервиса
docker compose restart app

# Проверка
docker compose logs -f app
```

### Обновление миграций

```bash
# Применение новых миграций
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -f migrations/00X_new_migration.sql
```

## Управление сервисом

### Команды Docker Compose

```bash
# Запуск
docker compose up -d

# Остановка
docker compose down

# Перезапуск
docker compose restart app

# Просмотр логов
docker compose logs -f app

# Статус
docker compose ps

# Просмотр ресурсов
docker stats etalon-nomenclature
```

### Просмотр логов

```bash
# Последние 100 строк
docker compose logs --tail=100 app

# С timestamp
docker compose logs -f --timestamps app

# Только ошибки
docker compose logs app | grep ERROR
```

## Мониторинг

### Health Check

```bash
# Проверка health check
docker inspect etalon-nomenclature | grep -A 10 Health
```

### Использование ресурсов

```bash
# CPU и память
docker stats etalon-nomenclature --no-stream

# Размер логов
du -h "$(docker inspect --format='{{.LogPath}}' etalon-nomenclature)"
```

### Очистка старых логов

```bash
# Docker автоматически ограничивает размер логов (настройка в docker-compose.yml)
# max-size: "10m"
# max-file: "3"

# Ручная очистка (если нужно)
docker compose down
sudo truncate -s 0 "$(docker inspect --format='{{.LogPath}}' etalon-nomenclature)"
docker compose up -d
```

## Backup и восстановление

### Backup конфигурации

```bash
# Создание backup
tar -czf backup-$(date +%Y%m%d).tar.gz config.yaml ~/.cloud-certs/

# Восстановление
tar -xzf backup-20260305.tar.gz
```

### Backup базы данных

```bash
# Dump таблиц
PGSSLROOTCERT=~/.cloud-certs/root.crt pg_dump \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -t etalon_nomenclature \
  -t processed_emails \
  -f backup-db-$(date +%Y%m%d).sql

# Восстановление
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -f backup-db-20260305.sql
```

## Troubleshooting

### Проблема: Контейнер не запускается

```bash
# Проверка логов
docker compose logs app

# Проверка конфигурации
docker compose config

# Проверка Docker
docker ps -a
```

### Проблема: Нет подключения к IMAP

```bash
# Проверка из контейнера
docker compose exec app sh
nc -zv mail.hosting.reg.ru 993

# Проверка с хоста
nc -zv mail.hosting.reg.ru 993
telnet mail.hosting.reg.ru 993
```

### Проблема: Нет подключения к PostgreSQL

```bash
# Проверка сертификата
ls -la ~/.cloud-certs/root.crt

# Проверка подключения из контейнера
docker compose exec app sh
# внутри контейнера
ls -la /app/certs/root.crt

# Тест подключения
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -c "SELECT 1;"
```

### Проблема: Высокое использование памяти

```bash
# Проверка ресурсов
docker stats etalon-nomenclature

# Ограничение ресурсов в docker-compose.yml уже настроено:
# limits:
#   memory: 512M
```

### Проблема: Письма не обрабатываются

```bash
# Проверка обработанных писем
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -c "SELECT message_id, processed_at FROM processed_emails ORDER BY processed_at DESC LIMIT 20;"

# Удаление конкретного message_id для повторной обработки
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:YOUR_PASSWORD@..." \
  -c "DELETE FROM processed_emails WHERE message_id = '<message-id-here>';"
```

## Безопасность

### Ограничение доступа к config.yaml

```bash
chmod 600 config.yaml
```

### Ограничение доступа к сертификату

```bash
chmod 600 ~/.cloud-certs/root.crt
```

### Регулярное обновление

```bash
# Обновление системы
sudo apt update && sudo apt upgrade -y

# Обновление Docker образа
docker compose pull
docker compose up -d
```

## Полезные команды

```bash
# Просмотр всех Docker контейнеров
docker ps -a

# Просмотр всех Docker образов
docker images

# Очистка неиспользуемых ресурсов
docker system prune -a

# Очистка volumes
docker volume prune

# Просмотр сети
docker network ls
```

## Контакты и поддержка

При возникновении проблем:
1. Проверьте логи: `docker compose logs -f app`
2. Проверьте этот гайд
3. Создайте issue в репозитории
