# 🚀 Production Deployment Checklist

## Перед деплоем на облачный сервер

### ✅ 1. Очистка проекта

```bash
# Запустить скрипт очистки
./cleanup_project.sh

# Или вручную удалить временные файлы
rm -rf scripts/
rm -f analyze_excel.go check_excel_sheets.go clear_processed.go clear_tables.go
rm -f clear_tables.sql decode_subject.go test_*.go
rm -f app app.log bin/
```

### ✅ 2. Проверка конфигурации

#### config.yaml (создать на сервере)

```bash
# На production сервере создать config.yaml из примера
cp config.example.yaml config.yaml
nano config.yaml
```

Заполнить реальными данными:
- [ ] `database.dsn` - подключение к PostgreSQL (с URL-encoded паролем!)
- [ ] `database.ssl_root_cert` - путь к сертификату
- [ ] `mailboxes[].email` - email ящики для мониторинга
- [ ] `mailboxes[].password` - пароли от email
- [ ] `poll_interval` - интервал проверки (по умолчанию 30m)

#### .env (создать на сервере)

```bash
# На production сервере создать .env из примера
cp .env.example .env
nano .env
```

Заполнить реальными данными:
- [ ] `DB_PASSWORD` - пароль от БД
- [ ] `PGSSLROOTCERT_BASE64` - сертификат в base64
- [ ] `MAILBOXES_JSON` - JSON с email ящиками
- [ ] `ALLOWED_SENDERS` - список разрешенных отправителей

### ✅ 3. Проверка email отправителей

Актуальный список поставщиков:
- [ ] `pna@sibzapaska.ru` - ЗАПАСКА (шины + диски)
- [ ] `m.timoshenkova@bigm.pro` - БИГМАШИН (шины)
- [ ] `b2bportal@brinex.ru` - ГРУППА БРИНЕКС (шины + диски)

Обновить в `.env`:
```bash
ALLOWED_SENDERS=pna@sibzapaska.ru,m.timoshenkova@bigm.pro,b2bportal@brinex.ru
```

### ✅ 4. Проверка .gitignore

Убедиться, что секреты не попадут в Git:
```bash
cat .gitignore | grep -E "^config.yaml$|^\.env$"
```

Должно вывести:
```
.env
config.yaml
```

### ✅ 5. Проверка документации

Актуальные документы:
- [ ] `README.md` - главная документация
- [ ] `DEPLOYMENT.md` - инструкции по деплою
- [ ] `DEDUPLICATION_LOGIC.md` - логика дедупликации
- [ ] `FILE_PROCESSING_LOGIC.md` - обработка файлов
- [ ] `QUICK_REFERENCE.md` - краткая справка
- [ ] `PRODUCTION_CHECKLIST.md` - этот чек-лист

### ✅ 6. Проверка кода

```bash
# Убедиться, что проект компилируется
go build -o app cmd/app/main.go

# Запустить тесты
go test ./...

# Проверить линтером (опционально)
golangci-lint run
```

### ✅ 7. Docker образ

```bash
# Собрать образ
docker compose build

# Проверить размер образа
docker images | grep etalon-nomenclature
```

---

## На production сервере

### ✅ 8. Подготовка сервера

```bash
# Установить Docker и Docker Compose (если не установлено)
sudo apt update
sudo apt install -y docker.io docker-compose

# Клонировать репозиторий
git clone https://github.com/ViktorDevGo/Etalon_nomenclature.git
cd Etalon_nomenclature
```

### ✅ 9. Настройка конфигурации

```bash
# Создать config.yaml
cp config.example.yaml config.yaml
nano config.yaml
# Заполнить реальными данными

# Создать .env
cp .env.example .env
nano .env
# Заполнить реальными данными

# Создать директорию для сертификата
mkdir -p certs/

# Скопировать SSL сертификат
cp ~/.cloud-certs/root.crt certs/root.crt
```

### ✅ 10. Запуск приложения

```bash
# Собрать и запустить
docker compose up -d

# Проверить логи
docker compose logs -f app

# Проверить статус
docker compose ps
```

### ✅ 11. Проверка работы

Проверить в логах:
- [ ] ✅ `Database connection established successfully`
- [ ] ✅ `Database schema is up to date`
- [ ] ✅ Индексы созданы: `idx_price_tires_dedup`, `idx_price_disks_dedup`, `idx_etalon_nomenclature_dedup`
- [ ] ✅ `Processor started`
- [ ] ✅ `Processing mailbox`
- [ ] ✅ Письма обрабатываются без ошибок

### ✅ 12. Мониторинг

```bash
# Проверить работающие контейнеры
docker compose ps

# Следить за логами
docker compose logs -f app

# Проверить использование ресурсов
docker stats
```

### ✅ 13. Проверка БД

```bash
# Подключиться к БД
psql "postgresql://gen_user:PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

# Проверить таблицы
\dt

# Проверить индексы
SELECT tablename, indexname
FROM pg_indexes
WHERE tablename IN ('etalon_nomenclature', 'price_tires', 'price_disks')
ORDER BY tablename, indexname;

# Проверить количество записей
SELECT
    (SELECT COUNT(*) FROM etalon_nomenclature) as nomenclature,
    (SELECT COUNT(*) FROM price_tires) as tires,
    (SELECT COUNT(*) FROM price_disks) as disks,
    (SELECT COUNT(*) FROM processed_emails) as emails;
```

---

## Безопасность

### ⚠️ Важно!

1. **НЕ коммитить в Git:**
   - ❌ `config.yaml` (пароли БД и email)
   - ❌ `.env` (переменные окружения)
   - ❌ `certs/*.crt` (SSL сертификаты)

2. **Проверить перед push:**
   ```bash
   git status
   # НЕ должно быть: config.yaml, .env, *.crt
   ```

3. **Пароли должны быть:**
   - ✅ Сложными (минимум 12 символов)
   - ✅ Уникальными для каждого сервиса
   - ✅ Храниться только на production сервере

4. **SSL сертификат:**
   - ✅ Хранить в `certs/root.crt` на сервере
   - ✅ Или в base64 в `.env` как `PGSSLROOTCERT_BASE64`

---

## Troubleshooting

### Проблема: Password authentication failed

**Решение:**
```bash
# Проверить URL-encoding пароля
# Если пароль содержит < > ; : @ / ? # &
# Нужно закодировать: < = %3C, > = %3E, ; = %3B

# Пример: Poison-79<test
# Должно быть: Poison-79%3Ctest
```

### Проблема: SSL certificate verify failed

**Решение:**
```bash
# Проверить сертификат
ls -la certs/root.crt

# Или использовать sslmode=require вместо verify-full
# В config.yaml:
dsn: "postgresql://...?sslmode=require"
```

### Проблема: No emails found

**Решение:**
```bash
# Проверить ALLOWED_SENDERS в .env
# Проверить, что email отправителя в списке

# Проверить логи
docker compose logs -f app | grep "IMAP"
```

---

## Финальный чек-лист

Перед объявлением production ready:

- [ ] Временные файлы удалены
- [ ] `config.yaml` создан на сервере с реальными данными
- [ ] `.env` создан на сервере с реальными данными
- [ ] SSL сертификат настроен
- [ ] Email отправители актуальны
- [ ] Docker контейнер запущен
- [ ] Логи показывают успешное подключение к БД
- [ ] Логи показывают обработку писем
- [ ] Индексы дедупликации созданы
- [ ] Таблицы заполняются данными
- [ ] `config.yaml` и `.env` не в Git
- [ ] Документация актуальна

---

## 🎉 Готово к production!

Проект готов к деплою на облачный сервер.

Для обновления кода в будущем:
```bash
cd ~/Etalon_nomenclature
git pull
docker compose down
docker compose build
docker compose up -d
docker compose logs -f app
```
