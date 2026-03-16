# Миграция price_tires → tyres_prices_stock

## Что изменилось

### Таблица БД: `price_tires` → `tyres_prices_stock`

**Старая таблица (`price_tires`):**
```sql
CREATE TABLE price_tires (
    id SERIAL PRIMARY KEY,
    article TEXT NOT NULL,
    price NUMERIC,
    balance INTEGER,
    store TEXT,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);
```

**Новая таблица (`tyres_prices_stock`):**
```sql
CREATE TABLE tyres_prices_stock (
    id SERIAL PRIMARY KEY,
    cae TEXT NOT NULL,              -- article → cae
    price NUMERIC,                  -- price (без изменений)
    stock INTEGER,                  -- balance → stock
    warehouse_name TEXT,            -- store → warehouse_name
    provider TEXT,                  -- provider (без изменений)
    isimport INTEGER DEFAULT 0,     -- isimport (без изменений)
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);
```

### Маппинг колонок

| Старое название | Новое название | Описание |
|----------------|----------------|----------|
| `article` | `cae` | Артикул товара |
| `price` | `price` | Цена (без изменений) |
| `balance` | `stock` | Остаток на складе |
| `store` | `warehouse_name` | Название склада |
| `provider` | `provider` | Поставщик (без изменений) |
| `isimport` | `isimport` | Флаг импорта (без изменений) |

### Изменение логики сохранения

**Старая логика (append-only):**
- Если запись с точным совпадением `(article, price, balance, store)` существует → SKIP
- Если записи нет → INSERT с `isimport=0`

**Новая логика (UPSERT):**
- **Уникальный ключ:** `(cae, warehouse_name, provider)`
- **Если полный дубль** (включая price и stock) → **SKIP**
- **Если запись существует**, но price или stock изменились → **UPDATE** + установить `isimport=0` + обновить `created_at`
- **Если записи нет** → **INSERT** с `isimport=0`

**SQL запрос (UPSERT):**
```sql
INSERT INTO tyres_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 0, now())
ON CONFLICT (cae, warehouse_name, provider)
DO UPDATE SET
    price = EXCLUDED.price,
    stock = EXCLUDED.stock,
    email_date = EXCLUDED.email_date,
    isimport = 0,
    created_at = now()
WHERE tyres_prices_stock.price != EXCLUDED.price
   OR tyres_prices_stock.stock != EXCLUDED.stock
```

## Применение миграции

### Автоматическое применение (рекомендуется)

При следующем запуске приложения:
1. Автоматически создастся таблица `tyres_prices_stock` (если ее нет)
2. Автоматически удалится таблица `price_tires` (если она существует)

Никаких ручных действий не требуется!

### Ручное применение (опционально)

Если хотите применить миграцию вручную до запуска приложения:

```bash
# Подключитесь к БД
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full"

# Примените миграцию
\i migrations/002_replace_price_tires.sql
```

## Проверка после миграции

```sql
-- Проверить наличие новой таблицы
SELECT COUNT(*) FROM tyres_prices_stock;

-- Проверить, что старая таблица удалена
SELECT COUNT(*) FROM information_schema.tables
WHERE table_name = 'price_tires';  -- должно быть 0

-- Проверить уникальный индекс
SELECT indexname, indexdef FROM pg_indexes
WHERE tablename = 'tyres_prices_stock' AND indexname = 'idx_tyres_prices_stock_unique';
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

**Важно:** Автоматические миграции применятся при старте приложения. В логах вы увидите:

```
INFO: Checking database schema...
INFO: Dropped table price_tires
INFO: Database schema is up to date
```

## Rollback (если понадобится)

Если нужно откатить изменения:

1. **Остановить приложение:**
```bash
docker compose down
```

2. **Восстановить старую таблицу вручную:**
```sql
CREATE TABLE price_tires (
    id SERIAL PRIMARY KEY,
    article TEXT NOT NULL,
    price NUMERIC,
    balance INTEGER,
    store TEXT,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);

CREATE INDEX idx_price_tires_article ON price_tires(article);
CREATE INDEX idx_price_tires_provider ON price_tires(provider);
CREATE INDEX idx_price_tires_created_at ON price_tires(created_at);
CREATE INDEX idx_price_tires_dedup ON price_tires(article, price, balance, store);
```

3. **Откатить код:**
```bash
git checkout <previous-commit>
docker compose build
docker compose up -d
```

## Изменения в коде

### Структура данных
```go
// Старая структура
type PriceTireRow struct {
    Article   string
    Price     float64
    Balance   int
    Store     string
    Provider  string
    EmailDate time.Time
}

// Новая структура
type TyrePriceStockRow struct {
    CAE           string    // article → CAE
    Price         float64
    Stock         int       // balance → Stock
    WarehouseName string    // store → WarehouseName
    Provider      string
    EmailDate     time.Time
}
```

### Функции БД
- `InsertPriceTiresWithEmail()` → `InsertTyrePriceStockWithEmail()`
- `insertTiresInTx()` → `insertTyrePriceStockInTx()`

### Парсер
- `[]db.PriceTireRow` → `[]db.TyrePriceStockRow`
- Обновлены все поля структуры

## FAQ

**Q: Что произойдет с данными в старой таблице price_tires?**
A: Таблица будет удалена вместе со всеми данными. Если нужно сохранить данные, сделайте backup перед миграцией.

**Q: Можно ли перенести данные из price_tires в tyres_prices_stock?**
A: Да, если нужно:
```sql
INSERT INTO tyres_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, created_at)
SELECT article, price, balance, store, provider, email_date, isimport, created_at
FROM price_tires
ON CONFLICT (cae, warehouse_name, provider) DO NOTHING;
```

**Q: Что делать, если миграция не применилась автоматически?**
A: Проверьте логи приложения. Если есть ошибки подключения к БД, примените миграцию вручную из файла `migrations/002_replace_price_tires.sql`.

**Q: Изменится ли формат данных из email?**
A: Нет, парсинг Excel-файлов остался прежним. Изменилось только название полей в БД и логика сохранения (UPSERT вместо append-only).
