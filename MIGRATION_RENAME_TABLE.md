# Миграция: Переименование таблицы etalon_nomenclature → MRC_Etalon

## Что изменилось

### Переименование таблицы и индексов

**Старые названия:**
- Таблица: `etalon_nomenclature`
- Индексы:
  - `idx_etalon_nomenclature_article`
  - `idx_etalon_nomenclature_brand`
  - `idx_etalon_nomenclature_isimport`
  - `idx_etalon_nomenclature_created_at`
  - `idx_etalon_nomenclature_dedup`

**Новые названия:**
- Таблица: `MRC_Etalon`
- Индексы:
  - `idx_MRC_Etalon_article`
  - `idx_MRC_Etalon_brand`
  - `idx_MRC_Etalon_isimport`
  - `idx_MRC_Etalon_created_at`
  - `idx_MRC_Etalon_dedup`

### Причина переименования

Название `MRC_Etalon` более точно отражает назначение таблицы:
- **MRC** = Минимальная Розничная Цена (главное поле таблицы)
- **Etalon** = название компании/проекта

## Структура таблицы (без изменений)

```sql
CREATE TABLE MRC_Etalon (
    id SERIAL PRIMARY KEY,
    article TEXT,              -- Артикул товара
    brand TEXT,                -- Бренд
    type TEXT,                 -- Тип (Ш - шины)
    size_model TEXT,           -- Размер/модель
    nomenclature TEXT,         -- Полное название
    mrc NUMERIC,               -- МРЦ (минимальная розничная цена)
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);
```

## Применение миграции

### Автоматическое применение (рекомендуется)

При следующем запуске приложения миграция применится автоматически.

### Ручное применение (если нужно до деплоя)

```bash
# Подключитесь к БД
PGPASSWORD='ваш_пароль' psql "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require"

# Примените миграцию
\i migrations/004_rename_etalon_nomenclature.sql
```

Или одной командой:
```bash
PGPASSWORD='ваш_пароль' psql "postgresql://gen_user@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=require" \
  -f migrations/004_rename_etalon_nomenclature.sql
```

## Проверка после миграции

```sql
-- Проверить, что новая таблица существует
SELECT COUNT(*) FROM MRC_Etalon;

-- Проверить, что старая таблица удалена
SELECT COUNT(*) FROM information_schema.tables
WHERE table_name = 'etalon_nomenclature';  -- должно быть 0

-- Проверить индексы
SELECT indexname FROM pg_indexes
WHERE tablename = 'MRC_Etalon'
ORDER BY indexname;

-- Ожидаемый результат:
-- idx_MRC_Etalon_article
-- idx_MRC_Etalon_brand
-- idx_MRC_Etalon_created_at
-- idx_MRC_Etalon_dedup
-- idx_MRC_Etalon_isimport
```

## Deployment на продакшн

```bash
cd ~/etalon-nomenclature

# Получить изменения
git pull

# Пересобрать и перезапустить
docker compose down
docker compose build
docker compose up -d

# Проверить логи
docker compose logs -f app
```

В логах вы увидите:
```
INFO: Checking database schema...
INFO: Renamed table etalon_nomenclature to MRC_Etalon
INFO: Database schema is up to date
```

## Важные примечания

### ⚠️ Безопасность миграции

- ✅ **Данные сохраняются:** `ALTER TABLE RENAME` только меняет название, данные остаются
- ✅ **Нет простоя:** операция выполняется мгновенно (lock на долю секунды)
- ✅ **Идемпотентность:** `IF EXISTS` - безопасно запускать повторно
- ✅ **Откат:** просто переименовать обратно (см. ниже)

### 📊 Что НЕ меняется

- Содержимое таблицы (все записи остаются)
- Структура таблицы (колонки, типы данных)
- Логика работы (append-only с дедупликацией по article+mrc)
- Производительность (индексы сохраняются)

## Rollback (откат изменений)

Если нужно вернуть старое название:

```sql
-- Переименовать обратно
ALTER TABLE IF EXISTS MRC_Etalon RENAME TO etalon_nomenclature;

-- Переименовать индексы обратно
ALTER INDEX IF EXISTS idx_MRC_Etalon_article RENAME TO idx_etalon_nomenclature_article;
ALTER INDEX IF EXISTS idx_MRC_Etalon_brand RENAME TO idx_etalon_nomenclature_brand;
ALTER INDEX IF EXISTS idx_MRC_Etalon_isimport RENAME TO idx_etalon_nomenclature_isimport;
ALTER INDEX IF EXISTS idx_MRC_Etalon_created_at RENAME TO idx_etalon_nomenclature_created_at;
ALTER INDEX IF EXISTS idx_MRC_Etalon_dedup RENAME TO idx_etalon_nomenclature_dedup;
```

Затем откатить код:
```bash
git checkout <previous-commit>
docker compose build
docker compose up -d
```

## Изменения в коде

Все ссылки на `etalon_nomenclature` в коде заменены на `MRC_Etalon`:

### Измененные файлы:
- ✅ `internal/db/postgres.go` - определение таблицы и SQL запросы
- ✅ `migrations/001_init.sql` - инициализация таблицы
- ✅ `migrations/004_rename_etalon_nomenclature.sql` - новая миграция

### Без изменений:
- `internal/db/postgres.go` - структура `NomenclatureRow` (название структуры не меняется)
- `internal/parser/excel.go` - парсер Excel файлов
- `internal/service/processor.go` - логика обработки

## FAQ

**Q: Потеряются ли данные при переименовании?**
A: Нет, `ALTER TABLE RENAME` только меняет название таблицы, все данные остаются.

**Q: Нужно ли пересоздавать индексы?**
A: Нет, индексы тоже просто переименовываются и продолжают работать.

**Q: Будет ли простой сервиса?**
A: Нет, переименование таблицы происходит мгновенно. Приложение будет перезапущено при деплое (стандартная процедура).

**Q: Можно ли откатить изменения?**
A: Да, просто переименуйте таблицу обратно (см. раздел Rollback).

**Q: Изменится ли логика работы с таблицей?**
A: Нет, логика остается прежней:
  - Append-only (только INSERT, нет UPDATE/DELETE)
  - Дедупликация по (article, mrc)
  - Batch insert по 1000 записей

**Q: Нужно ли обновлять внешние системы?**
A: Если есть внешние системы, которые напрямую обращаются к таблице `etalon_nomenclature` через SQL, их нужно обновить на `MRC_Etalon`.
