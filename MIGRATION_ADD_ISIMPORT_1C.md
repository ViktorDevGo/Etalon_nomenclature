# Миграция: Добавление колонки isimport_1С в таблицу MRC_Etalon

## Что изменилось

### Новая колонка

**Добавлена колонка `isimport_1С` в таблицу `MRC_Etalon`:**

```sql
ALTER TABLE MRC_Etalon ADD COLUMN isimport_1С INTEGER DEFAULT 0;
```

**Создан индекс для быстрой фильтрации:**
```sql
CREATE INDEX idx_MRC_Etalon_isimport_1С ON MRC_Etalon(isimport_1С);
```

### Назначение колонки

Колонка `isimport_1С` предназначена для отслеживания импорта данных в систему 1С:

| Значение | Описание |
|----------|----------|
| `0` | Запись **не импортирована** в 1С (по умолчанию) |
| `1` | Запись **импортирована** в 1С |

### Обновленная структура таблицы

```sql
CREATE TABLE MRC_Etalon (
    id SERIAL PRIMARY KEY,
    article TEXT,
    brand TEXT,
    type TEXT,
    size_model TEXT,
    nomenclature TEXT,
    mrc NUMERIC,
    isimport INTEGER DEFAULT 0,        -- Флаг импорта (старый)
    isimport_1С INTEGER DEFAULT 0,     -- Флаг импорта в 1С (новый)
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);
```

## Применение миграции

### Автоматическое применение (рекомендуется)

При следующем запуске приложения миграция применится автоматически:

```bash
cd ~/etalon-nomenclature
git pull
docker compose down
docker compose build
docker compose up -d
```

В логах вы увидите:
```
INFO: Adding column isimport_1С to table MRC_Etalon...
INFO: Column isimport_1С added successfully
INFO: Creating index idx_MRC_Etalon_isimport_1С...
INFO: Index created successfully
```

### Ручное применение (опционально)

Если нужно применить миграцию вручную до деплоя:

```bash
# Подключитесь к БД
PGPASSWORD='ваш_пароль' psql "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

# Примените миграцию
\i migrations/005_add_isimport_1C.sql
```

Или одной командой:
```bash
PGPASSWORD='ваш_пароль' psql "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require" \
  -f migrations/005_add_isimport_1C.sql
```

## Проверка после миграции

```sql
-- Проверить наличие колонки
SELECT column_name, data_type, column_default
FROM information_schema.columns
WHERE table_name = 'MRC_Etalon' AND column_name = 'isimport_1С';

-- Ожидаемый результат:
-- column_name  | data_type | column_default
-- isimport_1С  | integer   | 0

-- Проверить индекс
SELECT indexname FROM pg_indexes
WHERE tablename = 'MRC_Etalon' AND indexname = 'idx_MRC_Etalon_isimport_1С';

-- Ожидаемый результат:
-- indexname
-- idx_MRC_Etalon_isimport_1С

-- Проверить данные (все существующие записи должны иметь isimport_1С = 0)
SELECT isimport_1С, COUNT(*) FROM MRC_Etalon GROUP BY isimport_1С;

-- Ожидаемый результат:
-- isimport_1С | count
-- 0           | 12797 (или другое количество записей)
```

## Использование новой колонки

### Получение записей, не импортированных в 1С

```sql
SELECT *
FROM MRC_Etalon
WHERE isimport_1С = 0
ORDER BY created_at DESC;
```

### Маркировка записей как импортированных

```sql
-- Пометить конкретную запись как импортированную
UPDATE MRC_Etalon
SET isimport_1С = 1
WHERE id = 123;

-- Пометить записи по артикулу
UPDATE MRC_Etalon
SET isimport_1С = 1
WHERE article = 'ABC123';

-- Пакетная маркировка (например, импортировали все записи за определенную дату)
UPDATE MRC_Etalon
SET isimport_1С = 1
WHERE DATE(created_at) = '2026-03-17'
  AND isimport_1С = 0;
```

### Статистика по импорту в 1С

```sql
-- Общая статистика
SELECT
    isimport_1С,
    COUNT(*) as count,
    ROUND(COUNT(*) * 100.0 / SUM(COUNT(*)) OVER (), 2) as percentage
FROM MRC_Etalon
GROUP BY isimport_1С;

-- Статистика по брендам
SELECT
    brand,
    SUM(CASE WHEN isimport_1С = 0 THEN 1 ELSE 0 END) as not_imported,
    SUM(CASE WHEN isimport_1С = 1 THEN 1 ELSE 0 END) as imported
FROM MRC_Etalon
GROUP BY brand
ORDER BY not_imported DESC;
```

## Важные примечания

### ⚠️ Безопасность миграции

- ✅ **Обратная совместимость:** Существующие записи получат значение `0` автоматически
- ✅ **Нет простоя:** Добавление колонки с DEFAULT происходит мгновенно
- ✅ **Идемпотентность:** `ADD COLUMN IF NOT EXISTS` - безопасно запускать повторно
- ✅ **Индекс:** Автоматически создается для быстрой фильтрации

### 📊 Что НЕ меняется

- Существующая логика работы приложения
- Парсинг Excel файлов
- Вставка новых записей (они получат `isimport_1С = 0`)
- Дедупликация по (article, mrc)

### 🔄 Автоматическая установка значения

При вставке новых записей через приложение колонка `isimport_1С` автоматически получает значение `0`:

```sql
INSERT INTO MRC_Etalon (article, brand, type, size_model, nomenclature, mrc, email_date, isimport)
VALUES ('ABC123', 'Hankook', 'Ш', '195/55 R 16', 'Hankook Ш 195/55 R 16', 1500.00, now(), 0);
-- isimport_1С автоматически станет 0 (DEFAULT)
```

## Rollback (откат изменений)

Если нужно удалить колонку:

```sql
-- Удалить индекс
DROP INDEX IF EXISTS idx_MRC_Etalon_isimport_1С;

-- Удалить колонку
ALTER TABLE MRC_Etalon DROP COLUMN IF EXISTS isimport_1С;
```

⚠️ **Внимание:** Удаление колонки приведет к потере всех данных о том, какие записи были импортированы в 1С!

## Интеграция с 1С

После внедрения этой колонки можно создать скрипт/процедуру в 1С, которая:

1. **Получает данные для импорта:**
   ```sql
   SELECT * FROM MRC_Etalon WHERE isimport_1С = 0 LIMIT 1000;
   ```

2. **Импортирует данные в 1С**

3. **Помечает импортированные записи:**
   ```sql
   UPDATE MRC_Etalon
   SET isimport_1С = 1
   WHERE id IN (список_импортированных_id);
   ```

Это позволит избежать повторного импорта одних и тех же данных.

## FAQ

**Q: Что произойдет с существующими записями?**
A: Все существующие записи автоматически получат значение `isimport_1С = 0`.

**Q: Будет ли простой сервиса?**
A: Нет, добавление колонки с DEFAULT происходит мгновенно (lock на долю секунды).

**Q: Нужно ли обновлять код приложения?**
A: Нет, колонка имеет DEFAULT значение, поэтому код продолжит работать без изменений.

**Q: Как сбросить флаг импорта для всех записей?**
A: Выполните: `UPDATE MRC_Etalon SET isimport_1С = 0;`

**Q: Можно ли использовать другие значения кроме 0 и 1?**
A: Да, но рекомендуется использовать:
  - `0` = не импортировано
  - `1` = импортировано
  - (опционально) `2` = ошибка импорта
  - (опционально) `3` = требует повторного импорта

**Q: Влияет ли эта колонка на производительность?**
A: Нет, благодаря индексу `idx_MRC_Etalon_isimport_1С` фильтрация будет быстрой даже на больших объемах данных.
