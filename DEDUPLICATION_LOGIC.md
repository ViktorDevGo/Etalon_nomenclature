# Логика дедупликации в проекте

## 📊 Обзор таблиц

Проект использует 3 основные таблицы для хранения данных:

| Таблица | Назначение | Дедупликация |
|---|---|---|
| `etalon_nomenclature` | Номенклатура с МРЦ | По **(article, mrc)** |
| `price_tires` | Цены на шины | По **(article, price, balance, store)** |
| `price_disks` | Цены на диски | По **(article, price, balance, store)** |

---

## 1️⃣ ETALON_NOMENCLATURE - Номенклатура

### Условие дубля:

Запись считается дублем, если **одновременно** совпадают:
- `article` (с TRIM пробелов)
- `mrc` (numeric сравнение)

### Логика вставки:

```sql
WITH new_data AS (
    SELECT * FROM unnest(
        $1::text[],    -- articles (с TRIM)
        $2::text[],    -- brands
        $3::text[],    -- types
        $4::text[],    -- size_models
        $5::text[],    -- nomenclatures
        $6::numeric[], -- mrcs
        $7::timestamp[] -- email_dates
    ) AS t(article, brand, type, size_model, nomenclature, mrc, email_date)
)
INSERT INTO etalon_nomenclature (article, brand, type, size_model, nomenclature, mrc, email_date, isimport)
SELECT article, brand, type, size_model, nomenclature, mrc, email_date, 0
FROM new_data nd
WHERE NOT EXISTS (
    SELECT 1 FROM etalon_nomenclature en
    WHERE TRIM(en.article) = TRIM(nd.article)
      AND en.mrc = nd.mrc
)
```

### Примеры:

| # | Article | MRC | Результат | Причина |
|---|---|---|---|---|
| 1 | `"TEST001"` | `1000.00` | ✅ **Добавлен** | Новая запись |
| 2 | `"TEST001"` | `1000.00` | ❌ **Дубль** | Точное совпадение с #1 |
| 3 | `"TEST001"` | `1100.00` | ✅ **Добавлен** | МРЦ отличается → не дубль |
| 4 | `"TEST002"` | `1000.00` | ✅ **Добавлен** | Артикул отличается → не дубль |
| 5 | `" TEST001 "` | `1000.00` | ❌ **Дубль** | TRIM(" TEST001 ") = "TEST001" |
| 6 | `"TEST001"` | `1000.0` | ❌ **Дубль** | 1000.0 = 1000.00 (numeric) |

### Индекс для производительности:

```sql
CREATE INDEX idx_etalon_nomenclature_dedup ON etalon_nomenclature(article, mrc);
```

### Особенности:

- ✅ **Append-only** - старые записи не удаляются
- ✅ **Историчность** - все уникальные (article, mrc) сохраняются
- ✅ **Нормализация** - TRIM для article, numeric для mrc
- ✅ **Производительность** - составной индекс для быстрого поиска

---

## 2️⃣ PRICE_TIRES - Цены на шины

### Условие дубля:

Запись считается дублем, если **одновременно** совпадают:
- `article`
- `price`
- `balance`
- `store`

### Логика вставки:

```sql
WITH new_data AS (
    SELECT * FROM unnest(
        $1::text[],    -- articles
        $2::numeric[], -- prices
        $3::integer[], -- balances
        $4::text[],    -- stores
        $5::text[],    -- providers
        $6::timestamp[] -- email_dates
    ) AS t(article, price, balance, store, provider, email_date)
)
INSERT INTO price_tires (article, price, balance, store, provider, email_date, isimport)
SELECT article, price, balance, store, provider, email_date, 0
FROM new_data nd
WHERE NOT EXISTS (
    SELECT 1 FROM price_tires pt
    WHERE pt.article = nd.article
      AND pt.price = nd.price
      AND pt.balance = nd.balance
      AND pt.store = nd.store
)
```

### Примеры:

| # | Article | Price | Balance | Store | Результат |
|---|---|---|---|---|---|
| 1 | `"A001"` | `5000` | `10` | `"Склад А"` | ✅ **Добавлен** |
| 2 | `"A001"` | `5000` | `10` | `"Склад А"` | ❌ **Дубль** |
| 3 | `"A001"` | `5100` | `10` | `"Склад А"` | ✅ **Добавлен** (цена изменилась) |
| 4 | `"A001"` | `5000` | `15` | `"Склад А"` | ✅ **Добавлен** (остаток изменился) |
| 5 | `"A001"` | `5000` | `10` | `"Склад Б"` | ✅ **Добавлен** (склад другой) |

### Индекс для производительности:

```sql
CREATE INDEX idx_price_tires_dedup ON price_tires(article, price, balance, store);
```

---

## 3️⃣ PRICE_DISKS - Цены на диски

### Условие дубля:

Запись считается дублем, если **одновременно** совпадают:
- `article`
- `price`
- `balance`
- `store`

### Логика вставки:

```sql
WITH new_data AS (
    SELECT * FROM unnest(
        $1::text[],    -- articles
        $2::text[],    -- manufacturers
        $3::text[],    -- models
        $4::numeric[], -- widths
        $5::numeric[], -- diameters
        $6::text[],    -- drillings
        $7::text[],    -- radiuses
        $8::text[],    -- central_holes
        $9::text[],    -- colors
        $10::numeric[], -- prices
        $11::text[],   -- stores
        $12::integer[], -- balances
        $13::text[],   -- providers
        $14::timestamp[] -- email_dates
    ) AS t(article, manufacturer, model, width, diameter, drilling, radius,
           central_hole, color, price, store, balance, provider, email_date)
)
INSERT INTO price_disks
(article, manufacturer, model, width, diameter, drilling, radius,
 central_hole, color, price, store, balance, provider, email_date, isimport)
SELECT article, manufacturer, model, width, diameter, drilling, radius,
       central_hole, color, price, store, balance, provider, email_date, 0
FROM new_data nd
WHERE NOT EXISTS (
    SELECT 1 FROM price_disks pd
    WHERE pd.article = nd.article
      AND pd.price = nd.price
      AND pd.balance = nd.balance
      AND pd.store = nd.store
)
```

### Индекс для производительности:

```sql
CREATE INDEX idx_price_disks_dedup ON price_disks(article, price, balance, store);
```

---

## 📋 Сравнительная таблица

| Таблица | Поля дедупликации | Нормализация | Append-only | Индекс |
|---|---|---|---|---|
| **etalon_nomenclature** | article, mrc | TRIM(article), numeric(mrc) | ✅ Да | (article, mrc) |
| **price_tires** | article, price, balance, store | Нет | ✅ Да | (article, price, balance, store) |
| **price_disks** | article, price, balance, store | Нет | ✅ Да | (article, price, balance, store) |

---

## 🔄 История изменений

### До (старая логика для etalon_nomenclature):

```go
// 1. Дедупликация внутри батча по article
articleMap := make(map[string]NomenclatureRow)
for _, row := range rows {
    articleMap[row.Article] = row // Последний побеждает
}

// 2. УДАЛЕНИЕ записей за сегодня
DELETE FROM etalon_nomenclature
WHERE article = ANY($1) AND DATE(created_at) = CURRENT_DATE;

// 3. Простая вставка
INSERT INTO etalon_nomenclature (...) VALUES (...);
```

**Проблемы:**
- ❌ Удаляются все записи с таким article за день
- ❌ Теряется историчность (одна МРЦ перезаписывает другую)
- ❌ Нет проверки по mrc

### После (новая логика):

```go
// 1. БЕЗ дедупликации внутри батча
// 2. БЕЗ удалений (append-only)
// 3. INSERT ... SELECT WHERE NOT EXISTS по (article, mrc)

INSERT INTO etalon_nomenclature (...)
SELECT ... FROM new_data nd
WHERE NOT EXISTS (
    SELECT 1 FROM etalon_nomenclature en
    WHERE TRIM(en.article) = TRIM(nd.article)
      AND en.mrc = nd.mrc
);
```

**Преимущества:**
- ✅ Историчность сохранена
- ✅ Точная дедупликация по (article, mrc)
- ✅ Нормализация данных (TRIM, numeric)
- ✅ Производительность (композитный индекс)

---

## 🧪 Тестирование

Создан тест: `test_deduplication.go`

### Тестовые сценарии:

```
Запись #1: TEST001 + 1000.00    → ✅ Добавлен
Запись #2: TEST001 + 1000.00    → ❌ Дубль #1
Запись #3: TEST001 + 1100.00    → ✅ Добавлен (другая МРЦ)
Запись #4: TEST002 + 1000.00    → ✅ Добавлен (другой артикул)
Запись #5: " TEST001 " + 1000.00 → ❌ Дубль #1 (TRIM)
Запись #6: TEST001 + 1000.0     → ❌ Дубль #1 (numeric)
```

**Ожидаемый результат:**
- Вставлено: 3 записи (#1, #3, #4)
- Пропущено: 3 дубля (#2, #5, #6)

---

## 🚀 Деплой

```bash
cd ~/etalon-nomenclature
git pull
docker compose down
docker compose build
docker compose up -d
docker compose logs -f app
```

### Проверка индекса после деплоя:

```sql
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'etalon_nomenclature'
  AND indexname = 'idx_etalon_nomenclature_dedup';
```

Должен вернуть:
```
indexname: idx_etalon_nomenclature_dedup
indexdef: CREATE INDEX idx_etalon_nomenclature_dedup ON etalon_nomenclature(article, mrc)
```

---

## 💡 Важные моменты

### 1. Append-only (только добавление)

- ✅ Старые записи **никогда не удаляются**
- ✅ Старые записи **никогда не обновляются**
- ✅ Добавляются только **новые уникальные** комбинации

### 2. Нормализация

- **article**: применяется `TRIM()` перед сравнением
- **mrc**: используется numeric сравнение (1000 = 1000.00)

### 3. Производительность

- Составной индекс `(article, mrc)` обеспечивает **быстрый** поиск дублей
- Batch-обработка (1000 записей за раз) для эффективности
- `WHERE NOT EXISTS` использует индекс для оптимизации

### 4. Логирование

```
INFO Batch processed with deduplication
  batch_num: 1
  batch_size: 1000
  inserted: 750          ← сколько добавлено
  skipped_duplicates: 250 ← сколько пропущено
```

---

## 🎯 Итог

**Таблица `etalon_nomenclature` теперь:**
- ✅ Дедуплицируется по `(article, mrc)`
- ✅ Сохраняет историчность (append-only)
- ✅ Нормализует данные (TRIM, numeric)
- ✅ Работает быстро (составной индекс)
- ✅ Production-ready

**Таблицы `price_tires` и `price_disks`:**
- ✅ Без изменений
- ✅ Дедуплицируются по `(article, price, balance, store)`
