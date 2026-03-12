# Логика обработки файлов из почты

## 📬 Общая схема обработки

```
Email → IMAP → Вложения → Детектор типа → Парсер → Таблица БД
```

---

## 1️⃣ Получение писем из почты

### Файл: `internal/imap/client.go`

```go
// Подключение к почтовому ящику
mailbox: zakupki@etalon-shina.ru
host: mail.hosting.reg.ru:993
```

### Логика поиска писем:

```go
// Ищем письма за последние 3 дня
since := time.Now().AddDate(0, 0, -3)

// Фильтруем по отправителю
allowedSenders := []string{"pna@sibzapaska.ru"}

// Проверяем, обработано ли письмо ранее
isProcessed := db.IsEmailProcessed(messageID)
if isProcessed {
    skip // Пропускаем уже обработанные
}
```

**Результат:** Список писем с вложениями

---

## 2️⃣ Извлечение вложений из писем

### Файл: `internal/imap/client.go` (функция `FetchMessages`)

```go
type Attachment struct {
    Filename string  // Название файла (например: "МРЦ Лето 10.03.2026.xlsx")
    Content  []byte  // Содержимое файла
    Size     int64   // Размер в байтах
}

type Email struct {
    MessageID   string       // <008101dcb125$...@sibzapaska.ru>
    Subject     string       // Тема письма
    From        string       // pna@sibzapaska.ru
    Date        time.Time    // Дата письма
    Attachments []Attachment // Список вложений
}
```

**Результат:** Email со списком вложений (каждое вложение имеет название и содержимое)

---

## 3️⃣ Определение типа файла по названию

### Файл: `internal/parser/detector.go` (функция `DetectFileType`)

### Приоритет проверок:

```go
func DetectFileType(filename string) FileType {
    normalized := strings.ToLower(filename) // Приводим к нижнему регистру

    // 1️⃣ ПРИОРИТЕТ 1: Проверяем "МРЦ"
    if strings.Contains(normalized, "мрц") {
        return FileTypeNomenclature // → etalon_nomenclature
    }

    // 2️⃣ ПРИОРИТЕТ 2: Проверяем "диск"
    if strings.Contains(normalized, "диск") {
        return FileTypeDisk // → price_disks
    }

    // 3️⃣ ПРИОРИТЕТ 3: Проверяем "прайс"
    if strings.Contains(normalized, "прайс") {
        return FileTypePrice // → price_tires
    }

    // 4️⃣ ПО УМОЛЧАНИЮ: price
    return FileTypePrice // → price_tires
}
```

### Примеры определения типа:

| Название файла | Содержит | Тип файла | Таблица БД |
|---|---|---|---|
| **МРЦ Лето 10.03.2026.xlsx** | "мрц" | nomenclature | `etalon_nomenclature` |
| **МРЦ Зима 2026.xlsx** | "мрц" | nomenclature | `etalon_nomenclature` |
| **Прайс-лист ЗАПАСКА.xlsx** | "прайс" | price | `price_tires` |
| **Прайс БИГМАШИН.xls** | "прайс" | price | `price_tires` |
| **Диски БРИНЕКС.xlsx** | "диск" | disk | `price_disks` |
| **Автодиски прайс.xlsx** | "диск" (приоритет!) | disk | `price_disks` |
| **report_2026.xlsx** | - | price (default) | `price_tires` |

---

## 4️⃣ Определение поставщика по email

### Файл: `internal/parser/detector.go` (функция `DetectProvider`)

```go
func DetectProvider(emailFrom string) Provider {
    emailLower := strings.ToLower(emailFrom)

    switch {
    case strings.Contains(emailLower, "bigm.pro"):
        return ProviderBigMachine  // "БИГМАШИН"

    case strings.Contains(emailLower, "sibzapaska.ru"):
        return ProviderZapaska     // "ЗАПАСКА"

    case strings.Contains(emailLower, "brinex.ru"):
        return ProviderBrinex      // "ГРУППА БРИНЕКС"

    default:
        return ProviderUnknown     // "НЕИЗВЕСТНЫЙ"
    }
}
```

### Примеры:

| Email отправителя | Поставщик |
|---|---|
| pna@sibzapaska.ru | ЗАПАСКА |
| manager@bigm.pro | БИГМАШИН |
| sales@brinex.ru | ГРУППА БРИНЕКС |
| other@example.com | НЕИЗВЕСТНЫЙ |

---

## 5️⃣ Маршрутизация файлов по парсерам

### Файл: `internal/service/processor.go` (функция `processEmail`)

```go
for _, attachment := range email.Attachments {
    // Шаг 1: Определяем тип файла по названию
    fileType := detector.DetectFileType(attachment.Filename)

    // Шаг 2: Определяем поставщика по email
    provider := detector.DetectProvider(email.From)

    // Шаг 3: Маршрутизация по типу файла
    switch fileType {

    case FileTypeNomenclature:
        // 📋 НОМЕНКЛАТУРА → etalon_nomenclature
        rows := excelParser.Parse(attachment.Content, attachment.Filename, email.Date)
        db.InsertNomenclatureWithEmail(rows, email.MessageID)

    case FileTypePrice:
        // 💰 ПРАЙС → price_tires + price_disks (для ЗАПАСКА/БРИНЕКС)

        // Парсим шины
        tireRows := priceParser.Parse(attachment.Content, attachment.Filename, provider, email.Date)
        allPriceRows = append(allPriceRows, tireRows...)

        // Для ЗАПАСКА и БРИНЕКС парсим также диски из того же файла
        if provider == ProviderZapaska || provider == ProviderBrinex {
            diskRows := diskParser.Parse(attachment.Content, attachment.Filename, provider, email.Date)
            allDiskRows = append(allDiskRows, diskRows...)
        }

    case FileTypeDisk:
        // 💿 ДИСКИ → price_disks
        diskRows := diskParser.Parse(attachment.Content, attachment.Filename, provider, email.Date)
        allDiskRows = append(allDiskRows, diskRows...)
    }
}

// Сохраняем все данные одной транзакцией
db.InsertAllEmailDataWithTransaction(
    nomenclatureRows,  // → etalon_nomenclature
    tireRows,          // → price_tires
    diskRows,          // → price_disks
    email.MessageID    // → processed_emails (в конце)
)
```

---

## 6️⃣ Полная схема обработки с примерами

### Пример 1: Файл "МРЦ Лето 10.03.2026.xlsx" от pna@sibzapaska.ru

```
1. Email получен → MessageID: <008101dcb125$...@sibzapaska.ru>
2. Вложение найдено → Filename: "МРЦ Лето 10.03.2026.xlsx"
3. Детектор типа → Contains "мрц" → FileTypeNomenclature
4. Детектор поставщика → "sibzapaska.ru" → ProviderZapaska
5. Парсер → excelParser.Parse()
   - Лист "Прайс Лист1" → 4,055 строк
   - Лист "Прайс Лист2" → 250 строк
   - Всего: 8,035 строк
6. Таблица → etalon_nomenclature
7. Маркер → processed_emails (messageID)
```

**Результат:** 8,035 записей в `etalon_nomenclature`

---

### Пример 2: Файл "Прайс-лист ЗАПАСКА.xls" от pna@sibzapaska.ru

```
1. Email получен → MessageID: <...@sibzapaska.ru>
2. Вложение найдено → Filename: "Прайс-лист ЗАПАСКА.xls"
3. Детектор типа → Contains "прайс" → FileTypePrice
4. Детектор поставщика → "sibzapaska.ru" → ProviderZapaska
5. Парсер шин → priceParser.Parse()
   - Секция "Шины" → 2,500 строк
6. Таблица → price_tires (2,500 строк)
7. Парсер дисков → diskParser.Parse() (т.к. provider = ЗАПАСКА)
   - Секция "Диски" → 800 строк
8. Таблица → price_disks (800 строк)
9. Маркер → processed_emails (messageID)
```

**Результат:**
- 2,500 записей в `price_tires`
- 800 записей в `price_disks`

---

### Пример 3: Файл "Диски БРИНЕКС.xlsx" от sales@brinex.ru

```
1. Email получен → MessageID: <...@brinex.ru>
2. Вложение найдено → Filename: "Диски БРИНЕКС.xlsx"
3. Детектор типа → Contains "диск" → FileTypeDisk
4. Детектор поставщика → "brinex.ru" → ProviderBrinex
5. Парсер дисков → diskParser.Parse()
   - Лист "Автодиски" → 1,200 строк
6. Таблица → price_disks (1,200 строк)
7. Маркер → processed_emails (messageID)
```

**Результат:** 1,200 записей в `price_disks`

---

## 7️⃣ Дедупликация при вставке

### Таблица `etalon_nomenclature`:

```sql
-- Удаляем дубликаты для СЕГОДНЯ (одинаковые артикулы)
DELETE FROM etalon_nomenclature
WHERE article = ANY($1) AND DATE(created_at) = CURRENT_DATE;

-- Вставляем новые данные
INSERT INTO etalon_nomenclature (article, brand, type, size_model, nomenclature, mrc, email_date)
VALUES (...);
```

**Логика:** Для каждого артикула храним ТОЛЬКО последнюю запись за сегодня.

---

### Таблицы `price_tires` и `price_disks`:

```sql
WITH new_data AS (
    SELECT * FROM unnest($1::text[], $2::numeric[], $3::integer[], $4::text[], ...)
    AS t(article, price, balance, store, ...)
)
INSERT INTO price_tires (article, price, balance, store, provider, email_date)
SELECT article, price, balance, store, provider, email_date
FROM new_data nd
WHERE NOT EXISTS (
    SELECT 1 FROM price_tires pt
    WHERE pt.article = nd.article
      AND pt.price = nd.price
      AND pt.balance = nd.balance
      AND pt.store = nd.store
);
```

**Логика:** Пропускаем ТОЧНЫЕ дубли (article + price + balance + store), добавляем только изменения.

---

## 8️⃣ Финальная маркировка

### Таблица `processed_emails`:

```sql
-- Помечаем email как обработанный ТОЛЬКО если ВСЕ данные успешно сохранены
INSERT INTO processed_emails (message_id, email_date)
VALUES ($1, $2)
ON CONFLICT (message_id) DO NOTHING;
```

**Логика:** Если хотя бы одна таблица не сохранилась → откат транзакции → email НЕ помечен → при следующем запуске обработается снова.

---

## 9️⃣ Алгоритм принятия решений (блок-схема)

```
┌─────────────────────────┐
│  Получено письмо        │
│  с вложением            │
└───────────┬─────────────┘
            │
            ▼
┌─────────────────────────┐
│ Извлечь filename        │
│ (например: "МРЦ Лето.xlsx") │
└───────────┬─────────────┘
            │
            ▼
    ┌───────────────┐
    │ filename      │
    │ contains?     │
    └───┬───────────┘
        │
        ├─── "мрц" ───────────────────┐
        │                             │
        ├─── "диск" ──────────────┐   │
        │                         │   │
        ├─── "прайс" ─────────┐   │   │
        │                     │   │   │
        └─── default ─────┐   │   │   │
                          │   │   │   │
                          ▼   ▼   ▼   ▼
                    ┌─────────────────────┐
                    │   Тип файла         │
                    ├─────────────────────┤
                    │ price | disk | nom  │
                    └──────┬──────────────┘
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
        ▼                  ▼                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ priceParser  │   │ diskParser   │   │ excelParser  │
└──────┬───────┘   └──────┬───────┘   └──────┬───────┘
       │                  │                  │
       ▼                  ▼                  ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ price_tires  │   │ price_disks  │   │ etalon_      │
│              │   │              │   │ nomenclature │
└──────────────┘   └──────────────┘   └──────────────┘
```

---

## 🔟 Таблица соответствий

| Ключевое слово в filename | FileType | Парсер | Таблица БД | Дедупликация |
|---|---|---|---|---|
| **мрц** | nomenclature | excelParser | `etalon_nomenclature` | По article за день |
| **прайс** | price | priceParser | `price_tires` | По (article, price, balance, store) |
| **прайс** (ЗАПАСКА/БРИНЕКС) | price | priceParser + diskParser | `price_tires` + `price_disks` | По (article, price, balance, store) |
| **диск** | disk | diskParser | `price_disks` | По (article, price, balance, store) |
| *(любое другое)* | price (default) | priceParser | `price_tires` | По (article, price, balance, store) |

---

## Итог

**Вся логика определения таблицы зависит от:**
1. **Названия файла** (детектор ищет ключевые слова)
2. **Email отправителя** (определяет поставщика)
3. **Типа файла** (nomenclature/price/disk)

**Гарантии:**
- ✅ Email обрабатывается только 1 раз (проверка `processed_emails`)
- ✅ Дубликаты пропускаются (дедупликация при INSERT)
- ✅ Все данные сохраняются атомарно (транзакция)
- ✅ При ошибке данные не теряются (rollback + email не помечен)
