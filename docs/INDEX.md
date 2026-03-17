# Документация Etalon Nomenclature

## 📚 Структура документации

### 🚀 Быстрый старт
- [README.md](../README.md) - Основная информация и быстрый старт
- [DEPLOYMENT.md](../DEPLOYMENT.md) - Инструкции по деплою на продакшн
- [QUICK_REFERENCE.md](../QUICK_REFERENCE.md) - Быстрая справка

### 📋 Рабочие процессы
- [PRODUCTION_CHECKLIST.md](../PRODUCTION_CHECKLIST.md) - Чеклист перед деплоем
- [FILE_PROCESSING_LOGIC.md](../FILE_PROCESSING_LOGIC.md) - Логика обработки файлов
- [DEDUPLICATION_LOGIC.md](../DEDUPLICATION_LOGIC.md) - Логика дедупликации

### 🔄 Миграции БД
- [MIGRATION_TYRES_PRICES_STOCK.md](../MIGRATION_TYRES_PRICES_STOCK.md) - Миграция price_tires → tyres_prices_stock
- [MIGRATION_RIMS_PRICES_STOCK.md](../MIGRATION_RIMS_PRICES_STOCK.md) - Миграция price_disks → rims_prices_stock + nomenclature_rims
- [MIGRATION_RENAME_TABLE.md](../MIGRATION_RENAME_TABLE.md) - Переименование etalon_nomenclature → mrc_etalon
- [MIGRATION_ADD_ISIMPORT_1C.md](../MIGRATION_ADD_ISIMPORT_1C.md) - Добавление колонки isimport_1С

### 🗄️ SQL Миграции
- [001_init.sql](../migrations/001_init.sql) - Инициализация схемы БД
- [002_replace_price_tires.sql](../migrations/002_replace_price_tires.sql) - Замена price_tires
- [003_replace_price_disks.sql](../migrations/003_replace_price_disks.sql) - Замена price_disks
- [004_rename_etalon_nomenclature.sql](../migrations/004_rename_etalon_nomenclature.sql) - Переименование таблицы
- [005_add_isimport_1C.sql](../migrations/005_add_isimport_1C.sql) - Добавление isimport_1С

---

## 🗂️ Структура БД

### Основные таблицы

#### 1. `mrc_etalon` - Номенклатура с МРЦ
Хранит номенклатуру шин с минимальной розничной ценой.

**Источник:** Email от ЗАПАСКА с Excel файлами (колонка "МРЦ")

**Логика:** Append-only с дедупликацией по (article, mrc)

**Поля:**
- `id` - Первичный ключ
- `article` - Артикул
- `brand` - Бренд
- `type` - Тип (Ш - шины)
- `size_model` - Размер/модель
- `nomenclature` - Полное название (генерируется)
- `mrc` - МРЦ (минимальная розничная цена)
- `isimport` - Флаг импорта (0 = новая запись)
- `isimport_1С` - Флаг импорта в 1С (0 = не импортировано, 1 = импортировано)
- `created_at` - Дата создания
- `email_date` - Дата из email

#### 2. `tyres_prices_stock` - Цены и остатки шин
Хранит актуальные цены и остатки шин от всех поставщиков.

**Источник:** Email от поставщиков с прайс-листами

**Логика:** UPSERT по (cae, warehouse_name, provider)

**Поля:**
- `id` - Первичный ключ
- `cae` - Артикул (article)
- `price` - Цена
- `stock` - Остаток (balance)
- `warehouse_name` - Склад (store)
- `provider` - Поставщик
- `isimport` - Флаг импорта (0 при вставке/обновлении)
- `created_at` - Дата создания/обновления
- `email_date` - Дата из email

#### 3. `rims_prices_stock` - Цены и остатки дисков
Хранит актуальные цены и остатки дисков от всех поставщиков.

**Источник:** Email от поставщиков с прайс-листами

**Логика:** UPSERT по (cae, warehouse_name, provider)

**Поля:** Аналогичны `tyres_prices_stock`

#### 4. `nomenclature_rims` - Номенклатура дисков
Хранит детальную номенклатуру дисков ТОЛЬКО от ЗАПАСКА с определенными производителями.

**Источник:** Email от ЗАПАСКА

**Фильтры:**
- Поставщик: ЗАПАСКА
- Производители: COX, FF, Koko, Sakura

**Логика:** SKIP (ON CONFLICT DO NOTHING) по (cae)

**Поля:**
- `id` - Первичный ключ
- `cae` - Артикул
- `name` - Полное название (генерируется)
- `width` - Ширина
- `diameter` - Диаметр
- `bolts_count` - Количество болтов (из drilling: "5" из "5*114.3")
- `bolts_spacing` - Разболтовка (из drilling: "114.3" из "5*114.3")
- `et` - Вылет
- `dia` - Центральное отверстие
- `model` - Модель
- `brand` - Бренд
- `color` - Цвет
- `isimport` - Флаг импорта
- `created_at` - Дата создания
- `email_date` - Дата из email

#### 5. `processed_emails` - Обработанные письма
Отслеживает обработанные письма для предотвращения повторной обработки.

**Поля:**
- `id` - Первичный ключ
- `message_id` - Message-ID письма (уникальный)
- `email_date` - Дата письма
- `processed_at` - Дата обработки

---

## 🔧 Технические детали

### Поставщики
- **ЗАПАСКА** (pna@sibzapaska.ru) - номенклатура МРЦ, прайсы шин/дисков, номенклатура дисков
- **БИГМАШИН** - прайсы шин/дисков
- **БРИНЕКС** - прайсы шин/дисков

### Типы файлов
- **Номенклатура** - содержит колонку "МРЦ" → таблица `mrc_etalon`
- **Прайсы шин** - содержит данные о ценах/остатках шин → таблица `tyres_prices_stock`
- **Прайсы дисков** - содержит данные о ценах/остатках дисков → таблица `rims_prices_stock`
- **Номенклатура дисков** - детальные характеристики дисков ЗАПАСКА → таблица `nomenclature_rims`

### Логика обработки

#### Append-only (mrc_etalon)
- Вставка только уникальных записей
- Дедупликация по (article, mrc)
- Нет UPDATE, нет DELETE

#### UPSERT (tyres_prices_stock, rims_prices_stock)
- Уникальный ключ: (cae, warehouse_name, provider)
- При совпадении всех полей → SKIP
- При изменении price/stock → UPDATE + isimport=0
- При новой записи → INSERT + isimport=0

#### SKIP (nomenclature_rims)
- Уникальный ключ: (cae)
- При существующем CAE → SKIP (ничего не делаем)
- При новом CAE → INSERT + isimport=0
- Фильтрация: только ЗАПАСКА + COX/FF/Koko/Sakura

---

## 📞 Поддержка

При возникновении проблем:
1. Проверьте логи: `docker compose logs -f app`
2. Проверьте статус: `docker compose ps`
3. Проверьте БД: см. [QUICK_REFERENCE.md](../QUICK_REFERENCE.md)

---

**Версия документации:** 1.0
**Дата обновления:** 17.03.2026
