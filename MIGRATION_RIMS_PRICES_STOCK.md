# Миграция price_disks → rims_prices_stock + nomenclature_rims

## Что изменилось

### Таблицы БД: `price_disks` → `rims_prices_stock` + `nomenclature_rims`

**Старая таблица (`price_disks`):**
```sql
CREATE TABLE price_disks (
    id SERIAL PRIMARY KEY,
    article TEXT NOT NULL,
    manufacturer TEXT,
    model TEXT,
    width NUMERIC,
    diameter NUMERIC,
    drilling TEXT,
    radius TEXT,
    central_hole TEXT,
    color TEXT,
    price NUMERIC,
    store TEXT,
    balance INTEGER,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);
```

**Новые таблицы:**

1. **`rims_prices_stock`** - хранит цены и остатки дисков от ВСЕХ поставщиков:
```sql
CREATE TABLE rims_prices_stock (
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

2. **`nomenclature_rims`** - хранит номенклатуру дисков ТОЛЬКО для ЗАПАСКА + определенные производители:
```sql
CREATE TABLE nomenclature_rims (
    id SERIAL PRIMARY KEY,
    cae TEXT NOT NULL,
    name TEXT,                      -- полное название диска
    width NUMERIC,
    diameter NUMERIC,
    bolts_count INTEGER,            -- извлекается из drilling ("5" из "5*114.3")
    bolts_spacing NUMERIC,          -- извлекается из drilling ("114.3" из "5*114.3")
    et TEXT,                        -- вылет (radius)
    dia TEXT,                       -- центральное отверстие (central_hole)
    model TEXT,
    brand TEXT,
    color TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);
```

### Маппинг колонок

**price_disks → rims_prices_stock:**

| Старое название | Новое название | Описание |
|----------------|----------------|----------|
| `article` | `cae` | Артикул товара |
| `price` | `price` | Цена (без изменений) |
| `balance` | `stock` | Остаток на складе |
| `store` | `warehouse_name` | Название склада |
| `provider` | `provider` | Поставщик (без изменений) |
| `isimport` | `isimport` | Флаг импорта (без изменений) |

**price_disks → nomenclature_rims (только ЗАПАСКА + COX, FF, Koko, Sakura):**

| Старое название | Новое название | Описание |
|----------------|----------------|----------|
| `article` | `cae` | Артикул товара |
| `manufacturer` | `brand` | Производитель |
| `model` | `model` | Модель (без изменений) |
| `width` | `width` | Ширина (без изменений) |
| `diameter` | `diameter` | Диаметр (без изменений) |
| `drilling` | `bolts_count`, `bolts_spacing` | Сверловка (разбивается на 2 поля) |
| `radius` | `et` | Вылет |
| `central_hole` | `dia` | Центральное отверстие |
| `color` | `color` | Цвет (без изменений) |
| - | `name` | Полное название (генерируется) |

### Изменение логики сохранения

#### rims_prices_stock (все поставщики)

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
INSERT INTO rims_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, created_at)
VALUES ($1, $2, $3, $4, $5, $6, 0, now())
ON CONFLICT (cae, warehouse_name, provider)
DO UPDATE SET
    price = EXCLUDED.price,
    stock = EXCLUDED.stock,
    email_date = EXCLUDED.email_date,
    isimport = 0,
    created_at = now()
WHERE rims_prices_stock.price != EXCLUDED.price
   OR rims_prices_stock.stock != EXCLUDED.stock
```

#### nomenclature_rims (ТОЛЬКО ЗАПАСКА + COX, FF, Koko, Sakura)

**Логика фильтрации:**
- Данные попадают в таблицу **ТОЛЬКО** если:
  - Поставщик = **ЗАПАСКА**
  - Производитель (brand) = **COX**, **FF**, **Koko** или **Sakura**

**Логика сохранения (SKIP):**
- **Уникальный ключ:** `(cae)`
- **Если CAE уже существует** → **SKIP** (ничего не делаем)
- **Если CAE новый** → **INSERT** с `isimport=0`

**SQL запрос (SKIP):**
```sql
INSERT INTO nomenclature_rims (cae, name, width, diameter, bolts_count, bolts_spacing, et, dia, model, brand, color, email_date, isimport, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 0, now())
ON CONFLICT (cae) DO NOTHING
```

### Парсинг сверловки (drilling)

**Формат входных данных:** "5*114.3" или "4*100"

**Извлечение:**
- `bolts_count` = **5** (первая цифра до `*`)
- `bolts_spacing` = **114.3** (цифра после `*`)

**Пример в коде:**
```go
parts := strings.Split(diskData.Drilling, "*")
if len(parts) == 2 {
    boltsCount, _ = strconv.Atoi(parts[0])       // "5"
    boltsSpacing, _ = strconv.ParseFloat(parts[1], 64) // "114.3"
}
```

### Генерация поля `name` в nomenclature_rims

Формат: `"{Brand} {Model} {Width}x{Diameter} {Drilling} {ET} {DIA} {Color}"`

**Пример:**
```
COX D3255 7.0x16 5*114.3 ET35 D66.1 Серебристый
```

## Применение миграции

### Автоматическое применение (рекомендуется)

При следующем запуске приложения:
1. Автоматически создадутся таблицы `rims_prices_stock` и `nomenclature_rims` (если их нет)
2. Автоматически удалится таблица `price_disks` (если она существует)

Никаких ручных действий не требуется!

### Ручное применение (опционально)

Если хотите применить миграцию вручную до запуска приложения:

```bash
# Подключитесь к БД
PGSSLROOTCERT=~/.cloud-certs/root.crt psql \
  "postgresql://gen_user:PASSWORD@c37e696087932476c61fd621.twc1.net:5432/default_db?sslmode=verify-full"

# Примените миграцию
\i migrations/003_replace_price_disks.sql
```

## Проверка после миграции

```sql
-- Проверить наличие новых таблиц
SELECT COUNT(*) FROM rims_prices_stock;
SELECT COUNT(*) FROM nomenclature_rims;

-- Проверить, что старая таблица удалена
SELECT COUNT(*) FROM information_schema.tables
WHERE table_name = 'price_disks';  -- должно быть 0

-- Проверить уникальные индексы
SELECT indexname, indexdef FROM pg_indexes
WHERE tablename IN ('rims_prices_stock', 'nomenclature_rims')
  AND indexname LIKE '%unique%';

-- Проверить данные в nomenclature_rims (должны быть только ЗАПАСКА + определенные производители)
SELECT brand, COUNT(*) FROM nomenclature_rims
GROUP BY brand
ORDER BY COUNT(*) DESC;
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
INFO: Dropped table price_disks
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
CREATE TABLE price_disks (
    id SERIAL PRIMARY KEY,
    article TEXT NOT NULL,
    manufacturer TEXT,
    model TEXT,
    width NUMERIC,
    diameter NUMERIC,
    drilling TEXT,
    radius TEXT,
    central_hole TEXT,
    color TEXT,
    price NUMERIC,
    store TEXT,
    balance INTEGER,
    provider TEXT,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    email_date TIMESTAMP
);

CREATE INDEX idx_price_disks_article ON price_disks(article);
CREATE INDEX idx_price_disks_provider ON price_disks(provider);
CREATE INDEX idx_price_disks_created_at ON price_disks(created_at);
CREATE INDEX idx_price_disks_dedup ON price_disks(article, price, balance, store);
```

3. **Откатить код:**
```bash
git checkout <previous-commit>
docker compose build
docker compose up -d
```

## Изменения в коде

### Структуры данных

```go
// Старая структура (внутренняя для парсинга)
type PriceDiskRow struct {
    Article      string
    Manufacturer string
    Model        string
    Width        float64
    Diameter     float64
    Drilling     string
    Radius       string
    CentralHole  string
    Color        string
    Price        float64
    Store        string
    Balance      int
    Provider     string
    EmailDate    time.Time
}

// Новые структуры (для БД)
type RimPriceStockRow struct {
    CAE           string    // article → CAE
    Price         float64
    Stock         int       // balance → Stock
    WarehouseName string    // store → WarehouseName
    Provider      string
    EmailDate     time.Time
}

type NomenclatureRimRow struct {
    CAE          string
    Name         string    // генерируется из данных диска
    Width        float64
    Diameter     float64
    BoltsCount   int       // из drilling
    BoltsSpacing float64   // из drilling
    ET           string    // radius → ET
    DIA          string    // central_hole → DIA
    Model        string
    Brand        string    // manufacturer → Brand
    Color        string
    EmailDate    time.Time
}
```

### Возвращаемый результат парсера

```go
type ParseResult struct {
    RimPriceRows        []db.RimPriceStockRow
    RimNomenclatureRows []db.NomenclatureRimRow
}

func (p *DiskParser) Parse(...) (*ParseResult, error)
```

### Функции БД

- `InsertPriceDisksWithEmail()` → удалена
- `insertRimPriceStockInTx()` → новая (UPSERT)
- `insertRimNomenclatureInTx()` → новая (SKIP)
- `InsertAllEmailDataWithTransaction()` → обновлена (принимает rim данные)

### Фильтрация данных

**В parser/disk.go:**
```go
func shouldAddToNomenclature(provider, manufacturer string) bool {
    if !strings.Contains(provider, "ЗАПАСКА") {
        return false
    }

    mfg := strings.ToLower(manufacturer)
    allowedManufacturers := []string{"cox", "ff", "koko", "sakura"}
    for _, allowed := range allowedManufacturers {
        if strings.Contains(mfg, allowed) {
            return true
        }
    }
    return false
}
```

## FAQ

**Q: Что произойдет с данными в старой таблице price_disks?**
A: Таблица будет удалена вместе со всеми данными. Если нужно сохранить данные, сделайте backup перед миграцией.

**Q: Можно ли перенести данные из price_disks в новые таблицы?**
A: Да, если нужно:
```sql
-- В rims_prices_stock (все данные)
INSERT INTO rims_prices_stock (cae, price, stock, warehouse_name, provider, email_date, isimport, created_at)
SELECT article, price, balance, store, provider, email_date, isimport, created_at
FROM price_disks
ON CONFLICT (cae, warehouse_name, provider) DO NOTHING;

-- В nomenclature_rims (только ЗАПАСКА + COX, FF, Koko, Sakura)
INSERT INTO nomenclature_rims (cae, name, width, diameter, bolts_count, bolts_spacing, et, dia, model, brand, color, email_date, isimport, created_at)
SELECT
    article,
    CONCAT_WS(' ', manufacturer, model,
        CONCAT(width::text, 'x', diameter::text),
        drilling, radius, central_hole, color),
    width,
    diameter,
    CAST(SPLIT_PART(drilling, '*', 1) AS INTEGER),
    CAST(SPLIT_PART(drilling, '*', 2) AS NUMERIC),
    radius,
    central_hole,
    model,
    manufacturer,
    color,
    email_date,
    isimport,
    created_at
FROM price_disks
WHERE provider LIKE '%ЗАПАСКА%'
  AND LOWER(manufacturer) ~ '(cox|ff|koko|sakura)'
ON CONFLICT (cae) DO NOTHING;
```

**Q: Почему nomenclature_rims только для ЗАПАСКА?**
A: Согласно требованиям проекта, детальная номенклатура дисков нужна только для ЗАПАСКА с определенными производителями (COX, FF, Koko, Sakura). Остальные поставщики хранят только цены и остатки в rims_prices_stock.

**Q: Что делать, если миграция не применилась автоматически?**
A: Проверьте логи приложения. Если есть ошибки подключения к БД, примените миграцию вручную из файла `migrations/003_replace_price_disks.sql`.

**Q: Изменится ли формат данных из email?**
A: Нет, парсинг Excel-файлов остался прежним. Изменилось только:
  - Название полей в БД
  - Логика сохранения (UPSERT для rims_prices_stock, SKIP для nomenclature_rims)
  - Разбиение данных на 2 таблицы
  - Парсинг сверловки на bolts_count и bolts_spacing
  - Фильтрация nomenclature_rims по поставщику и производителю

**Q: Как проверить, что фильтрация работает правильно?**
A: После запуска проверьте:
```sql
-- Все записи в nomenclature_rims должны быть только от ЗАПАСКА
SELECT DISTINCT provider FROM nomenclature_rims;

-- Все записи должны быть только от разрешенных производителей
SELECT DISTINCT brand FROM nomenclature_rims;
```
