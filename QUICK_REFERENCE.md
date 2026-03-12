# 📋 Быстрая справка: Логика обработки файлов

## 🎯 Главное правило

**Тип файла определяется по НАЗВАНИЮ файла, а не по содержимому!**

---

## 📊 Таблица маршрутизации

| Если filename содержит | → Тип файла | → Таблица БД | Пример |
|---|---|---|---|
| **"МРЦ"** | nomenclature | `etalon_nomenclature` | "МРЦ Лето 2026.xlsx" |
| **"диск"** | disk | `price_disks` | "Диски БРИНЕКС.xlsx" |
| **"прайс"** | price | `price_tires` (+disks для ЗАПАСКА/БРИНЕКС) | "Прайс-лист.xls" |
| *любое другое* | price (default) | `price_tires` | "report.xlsx" |

---

## 🔍 Приоритет проверок

```
1️⃣ "МРЦ"     → nomenclature (ВЫСШИЙ ПРИОРИТЕТ)
2️⃣ "диск"    → disk
3️⃣ "прайс"   → price
4️⃣ default   → price
```

---

## 📂 Примеры файлов

### ✅ Правильные названия:

```
✓ МРЦ Лето 10.03.2026.xlsx          → etalon_nomenclature
✓ МРЦ Зима 2026.xlsx                → etalon_nomenclature
✓ Прайс-лист ЗАПАСКА.xls            → price_tires + price_disks
✓ Прайс БИГМАШИН.xls                → price_tires
✓ Диски БРИНЕКС.xlsx                → price_disks
✓ Автодиски прайс.xlsx              → price_disks (приоритет "диск"!)
```

### ⚠️ Особые случаи:

```
! report_2026.xlsx                  → price_tires (default)
! data.xls                          → price_tires (default)
! МРЦ и прайс диски.xlsx            → etalon_nomenclature ("МРЦ" = приоритет 1!)
```

---

## 🚀 Полный процесс обработки

```
┌──────────────┐
│   📬 EMAIL   │
│              │
│ From: pna@   │
│ sibzapaska.ru│
│              │
│ Subject:     │
│ "МРЦ Лето"   │
│              │
│ Attachment:  │
│ МРЦ.xlsx     │
└──────┬───────┘
       │
       │ 1. Детектор типа файла (по filename)
       │
       ▼
┌──────────────┐
│   ДЕТЕКТОР   │
│              │
│ "МРЦ.xlsx"   │
│   contains   │
│    "мрц"?    │
│     YES ✓    │
│              │
│ FileType:    │
│ nomenclature │
└──────┬───────┘
       │
       │ 2. Выбор парсера
       │
       ▼
┌──────────────┐
│  EXCEL       │
│  PARSER      │
│              │
│ Парсит:      │
│ • Артикул    │
│ • Марка      │
│ • Размер     │
│ • МРЦ        │
│              │
│ Находит:     │
│ • Заголовки  │
│ • 8,035 строк│
└──────┬───────┘
       │
       │ 3. Сохранение в БД
       │
       ▼
┌──────────────┐
│   DATABASE   │
│              │
│ etalon_      │
│ nomenclature │
│              │
│ INSERT       │
│ 8,035 rows   │
│              │
│ Дедупликация:│
│ по article   │
│ за сегодня   │
└──────┬───────┘
       │
       │ 4. Маркировка
       │
       ▼
┌──────────────┐
│ processed_   │
│ emails       │
│              │
│ MessageID ✓  │
└──────────────┘
```

---

## 🎭 Поставщики (определяются по email)

| Email отправителя | Поставщик | Особенности |
|---|---|---|
| **@sibzapaska.ru** | ЗАПАСКА | Прайс содержит шины + диски |
| **@brinex.ru** | ГРУППА БРИНЕКС | Прайс содержит шины + диски |
| **@bigm.pro** | БИГМАШИН | Только шины |
| другие | НЕИЗВЕСТНЫЙ | Только шины |

---

## 💾 Дедупликация

### `etalon_nomenclature`:
```
Условие дубля: article + сегодняшний день
Действие: Удалить старый, вставить новый
```

### `price_tires` и `price_disks`:
```
Условие дубля: article + price + balance + store
Действие: Пропустить, если точный дубль
```

---

## ⚡ Быстрая проверка

**Хотите понять, куда попадет файл?**

1. Посмотрите на название файла
2. Найдите ключевое слово:
   - **МРЦ** → `etalon_nomenclature` ✅
   - **диск** → `price_disks`
   - **прайс** → `price_tires` (+ disks для ЗАПАСКА/БРИНЕКС)
   - **нет ключевых слов** → `price_tires` (по умолчанию)

---

## 📞 Код для справки

### Детектор (detector.go):
```go
// 1. "МРЦ" → nomenclature
if strings.Contains(normalized, "мрц") {
    return FileTypeNomenclature
}

// 2. "диск" → disk
if strings.Contains(normalized, "диск") {
    return FileTypeDisk
}

// 3. "прайс" → price
if strings.Contains(normalized, "прайс") {
    return FileTypePrice
}

// 4. default → price
return FileTypePrice
```

### Поставщик (detector.go):
```go
switch {
case strings.Contains(email, "sibzapaska.ru"):
    return ProviderZapaska

case strings.Contains(email, "brinex.ru"):
    return ProviderBrinex

case strings.Contains(email, "bigm.pro"):
    return ProviderBigMachine

default:
    return ProviderUnknown
}
```

---

## ✅ Чек-лист проверки нового файла

- [ ] Проверил название файла
- [ ] Определил тип по ключевому слову
- [ ] Проверил email отправителя (поставщик)
- [ ] Знаю, в какую таблицу попадут данные
- [ ] Понимаю условие дедупликации

**Готово!** 🎉
