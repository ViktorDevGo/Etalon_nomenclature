# Деплой исправления: БРИНЕКС disk parsing

## 🐛 Проблема которую исправили

**Симптом:** Отсутствуют диски за 06-08.03 (~10,000 записей)

**Причина:**
- Ошибки парсинга дисков для БРИНЕКС логировались как **Warning** (line 256)
- Email помечался как processed даже без дисков (только с шинами)
- Результат: шины записались, диски потерялись

**Исправление (commit 0bb5d6e):**
- Для БРИНЕКС ошибка парсинга дисков теперь **FATAL** → email НЕ обработается
- Для ЗАПАСКА остается Warning (диски опциональны)
- Добавлена проверка: БРИНЕКС должен иметь len(diskRows) > 0

## 📋 План действий

### 1. Деплой нового кода

```bash
ssh u2827889@62.122.170.171 << 'EOF'
cd ~/etalon-nomenclature
git pull
docker compose down
docker compose build
docker compose up -d
echo "✓ Деплой завершен"
EOF
```

### 2. Очистить processed_emails для БРИНЕКС за 06-08.03

```bash
ssh u2827889@62.122.170.171 << 'EOF'
cd etalon-nomenclature
docker compose exec -T app sh -c "PGPASSWORD='<3B;hH5EFDH' psql -h c37e696087932476c61fd621.twc1.net -U gen_user -d default_db" << 'SQL'

-- Проверим какие письма удалим
SELECT
    message_id,
    email_date,
    processed_at
FROM processed_emails
WHERE message_id LIKE '%brinex.ru%'
  AND DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08'
ORDER BY email_date;

-- Удалим для повторной обработки
DELETE FROM processed_emails
WHERE message_id LIKE '%brinex.ru%'
  AND DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08';

SELECT 'Удалено писем: ' || COUNT(*) FROM processed_emails WHERE false;

SQL
EOF
```

### 3. Мониторинг обработки (в реальном времени)

```bash
ssh u2827889@62.122.170.171 'cd etalon-nomenclature && docker compose logs -f app'
```

**Что искать в логах:**

✅ **Успешная обработка:**
```
"Parsed disk section from price attachment" provider="ГРУППА БРИНЕКС" rows=XXX
"Successfully processed email and saved ALL data atomically"
```

❌ **Ошибка парсинга (критическая информация!):**
```
"Failed to parse disk section from БРИНЕКС file (should always have disks)"
error="..."
```

Если видите ошибку - это важно! Она покажет ПОЧЕМУ диски не парсились.

### 4. Проверка результата (через 2-3 минуты)

```bash
ssh u2827889@62.122.170.171 << 'EOF'
cd etalon-nomenclature
docker compose exec -T app sh -c "PGPASSWORD='<3B;hH5EFDH' psql -h c37e696087932476c61fd621.twc1.net -U gen_user -d default_db" << 'SQL'

-- Проверим диски по датам
SELECT
    provider,
    DATE(email_date) as date,
    COUNT(*) as count
FROM price_disks
WHERE provider LIKE '%БРИНЕКС%'
GROUP BY provider, DATE(email_date)
ORDER BY date;

-- Ожидаемый результат:
-- ГРУППА БРИНЕКС | 2026-03-06 | ~6000-7000
-- ГРУППА БРИНЕКС | 2026-03-07 | ~6000-7000
-- ГРУППА БРИНЕКС | 2026-03-08 | ~6000-7000
-- ГРУППА БРИНЕКС | 2026-03-09 | ~10000

SQL
EOF
```

## 🎯 Возможные сценарии

### Сценарий 1: Успех ✅
- Логи показывают "Parsed disk section" для всех дат
- В БД ~22,000+ дисков от БРИНЕКС
- Проблема решена!

### Сценарий 2: Ошибка парсинга ❌
- Логи показывают "Failed to parse disk section from БРИНЕКС file"
- Email НЕ обработан (остается в непрочитанных)
- **Действие:** Нужно исправить причину ошибки (см. error в логах)

Возможные ошибки:
- LibreOffice не установлен → нужно установить
- Проблема с кодировкой → нужно фиксить parser
- Лист не найден → проблема с регистром/названием

### Сценарий 3: Нет дисков (len=0) ❌
- Parser не возвращает ошибку, но diskRows пустой
- Логи: "No disks found in БРИНЕКС file"
- **Действие:** Проверить shouldProcessSheet и parseSheet логику

## 📊 Унификация provider naming

После успешной обработки унифицируем названия:

```bash
ssh u2827889@62.122.170.171 << 'EOF'
cd etalon-nomenclature
docker compose exec -T app sh -c "PGPASSWORD='<3B;hH5EFDH' psql -h c37e696087932476c61fd621.twc1.net -U gen_user -d default_db" << 'SQL'

UPDATE price_disks SET provider = 'ГРУППА БРИНЕКС' WHERE provider = 'БРИНЕКС';
UPDATE price_tires SET provider = 'ГРУППА БРИНЕКС' WHERE provider = 'БРИНЕКС';

SELECT 'price_disks:' as info, provider, COUNT(*) FROM price_disks GROUP BY provider;
SELECT 'price_tires:' as info, provider, COUNT(*) FROM price_tires GROUP BY provider;

SQL
EOF
```

## 🎯 Итог

Это исправление гарантирует что для БРИНЕКС:
- ✅ Либо сохраняются И шины И диски
- ✅ Либо ничего не сохраняется (email остается непрочитанным)
- ❌ Невозможна ситуация "шины есть, дисков нет"
