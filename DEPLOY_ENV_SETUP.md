# Установка переменных окружения на платформе деплоя

## 🔧 Проблема

Ошибка: `at least one mailbox must be configured`

Это означает, что переменные окружения не установлены на платформе деплоя.

## ✅ Решение

Нужно установить следующие переменные окружения через интерфейс вашей платформы деплоя:

### Обязательные переменные:

```bash
DATABASE_DSN=postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require

MAILBOXES_JSON=[{"email":"zakupki@etalon-shina.ru","password":"S69Y1ypojVLCZHO8","host":"mail.hosting.reg.ru","port":993}]

POLL_INTERVAL=1m

TZ=Europe/Moscow
```

### Опциональные переменные:

```bash
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
DATABASE_CONN_MAX_LIFETIME=5m
PGSSLROOTCERT_BASE64=
```

## 📋 Инструкции для разных платформ

### Если вы используете Railway:

1. Откройте ваш проект на Railway
2. Перейдите в **Variables**
3. Добавьте каждую переменную:
   - Name: `DATABASE_DSN`
   - Value: `postgresql://gen_user:uzShH%3CA8S%3B7c.e@...`
4. Повторите для всех переменных
5. Railway автоматически перезапустит сервис

### Если вы используете Render:

1. Откройте ваш сервис на Render
2. Перейдите в **Environment**
3. Добавьте переменные одну за другой
4. Нажмите **Save Changes**
5. Render автоматически перезапустит

### Если вы используете Heroku:

```bash
heroku config:set DATABASE_DSN="postgresql://gen_user:uzShH%3CA8S%3B7c.e@..."
heroku config:set MAILBOXES_JSON='[{"email":"zakupki@etalon-shina.ru","password":"S69Y1ypojVLCZHO8","host":"mail.hosting.reg.ru","port":993}]'
heroku config:set POLL_INTERVAL=1m
heroku config:set TZ=Europe/Moscow
```

### Если вы используете Docker Swarm/Kubernetes:

Создайте Secret или ConfigMap с переменными.

### Если вы используете обычный VPS:

Создайте systemd service файл:

```bash
sudo nano /etc/systemd/system/etalon-nomenclature.service
```

```ini
[Unit]
Description=Etalon Nomenclature Service
After=docker.service
Requires=docker.service

[Service]
Type=simple
Restart=always
WorkingDirectory=/path/to/project
Environment="DATABASE_DSN=postgresql://gen_user:uzShH%3CA8S%3B7c.e@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"
Environment="MAILBOXES_JSON=[{\"email\":\"zakupki@etalon-shina.ru\",\"password\":\"S69Y1ypojVLCZHO8\",\"host\":\"mail.hosting.reg.ru\",\"port\":993}]"
Environment="POLL_INTERVAL=1m"
Environment="TZ=Europe/Moscow"
ExecStart=/usr/bin/docker compose up
ExecStop=/usr/bin/docker compose down

[Install]
WantedBy=multi-user.target
```

## 🔍 Проверка

После установки переменных, проверьте логи. Должны увидеть:

```
INFO: Starting Etalon Nomenclature Service, mailboxes=1
INFO: Database connection established successfully
INFO: Processor started
```

## ⚠️ Важные замечания

1. **Экранирование кавычек**: В `MAILBOXES_JSON` используются одинарные кавычки снаружи и двойные внутри
2. **URL encoding**: В `DATABASE_DSN` пароль должен быть URL-encoded (`%3C` вместо `<`, `%3B` вместо `;`)
3. **Без пробелов**: В JSON не должно быть пробелов после запятых и двоеточий

## 🆘 Если не работает

1. Проверьте, что переменные установлены:
   ```bash
   # На платформе должна быть возможность просмотра env vars
   ```

2. Проверьте формат JSON:
   ```bash
   echo '[{"email":"zakupki@etalon-shina.ru","password":"S69Y1ypojVLCZHO8","host":"mail.hosting.reg.ru","port":993}]' | python3 -m json.tool
   ```

3. Проверьте логи на наличие других ошибок

## 📞 Какую платформу вы используете?

Напишите название платформы, и я дам точную инструкцию!
