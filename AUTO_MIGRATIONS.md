# Автоматические миграции базы данных

## 🎯 Решаемая проблема

Раньше для работы приложения требовалось **вручную применять SQL миграции** через psql:
```bash
psql "CONNECTION_STRING" -f migrations/001_init.sql
```

Это создавало проблемы:
- ❌ Дополнительный шаг при деплое
- ❌ Нужен доступ к psql
- ❌ Ошибки при забытых миграциях
- ❌ Сложности при автоматическом деплое на облачные платформы

## ✅ Решение: Автоматические миграции

Теперь приложение **само проверяет и создает схему БД** при запуске!

### Как это работает

```
Запуск приложения
      ↓
Подключение к PostgreSQL
      ↓
Проверка: есть ли таблицы?
      ↓
   ╔═══╩═══╗
   ↓       ↓
  ДА      НЕТ
   ↓       ↓
Готово   Применение миграций
         ↓
      Создание таблиц
         ↓
      Создание индексов
         ↓
       Готово
```

### Что происходит при старте

1. **Проверка подключения** (`db.PingContext`)
   ```
   INFO: Database connection established successfully
   ```

2. **Проверка схемы** (`checkTablesExist`)
   ```
   INFO: Checking database schema...
   ```

3. **Применение миграций** (если нужно)
   ```
   INFO: Required tables not found, applying migrations...
   INFO: Migrations applied successfully
   ```

4. **Готово к работе**
   ```
   INFO: Processor started
   INFO: Starting email processing cycle
   ```

## 📋 Технические детали

### Что создается

**Таблица `etalon_nomenclature`:**
```sql
CREATE TABLE etalon_nomenclature (
    id SERIAL PRIMARY KEY,
    article TEXT,
    brand TEXT,
    type TEXT,
    size_model TEXT,
    nomenclature TEXT,
    mrc NUMERIC,
    isimport INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now()
);
```

**Индексы для производительности:**
- `idx_etalon_nomenclature_article` - быстрый поиск по артикулу
- `idx_etalon_nomenclature_brand` - быстрый поиск по бренду
- `idx_etalon_nomenclature_isimport` - фильтрация по статусу импорта
- `idx_etalon_nomenclature_created_at` - сортировка по дате

**Таблица `processed_emails`:**
```sql
CREATE TABLE processed_emails (
    id SERIAL PRIMARY KEY,
    message_id TEXT NOT NULL,
    processed_at TIMESTAMP DEFAULT now()
);

CREATE UNIQUE INDEX idx_processed_emails_message_id
ON processed_emails(message_id);
```

### Расположение в коде

**Файл:** `internal/db/postgres.go`

**Ключевые функции:**

1. **`New()`** - создает подключение и вызывает `ensureSchema()`
   ```go
   // Check and apply migrations if needed
   if err := database.ensureSchema(ctx); err != nil {
       return nil, fmt.Errorf("failed to ensure database schema: %w", err)
   }
   ```

2. **`ensureSchema()`** - проверяет схему и применяет миграции
   ```go
   func (d *Database) ensureSchema(ctx context.Context) error {
       exists, err := d.checkTablesExist(ctx)
       if !exists {
           return d.applyMigrations(ctx)
       }
       return nil
   }
   ```

3. **`checkTablesExist()`** - проверяет наличие таблиц
   ```go
   SELECT COUNT(*) FROM information_schema.tables
   WHERE table_schema = 'public'
   AND table_name IN ('etalon_nomenclature', 'processed_emails')
   ```

4. **`applyMigrations()`** - применяет SQL миграции
   ```go
   // Встроенная SQL миграция в константе migrationSQL
   const migrationSQL = `...`
   ```

## 🚀 Преимущества

### Для разработки
- ✅ Не нужно помнить о применении миграций
- ✅ Новые разработчики сразу получают рабочую БД
- ✅ Легко сбросить БД и начать заново

### Для продакшена
- ✅ Простой деплой без дополнительных шагов
- ✅ Работает с любыми облачными платформами
- ✅ Автоматическое восстановление схемы

### Для CI/CD
- ✅ Не нужны отдельные шаги для миграций
- ✅ Меньше точек отказа
- ✅ Быстрее деплой

## 🔒 Безопасность

### Идемпотентность

Миграции используют `CREATE TABLE IF NOT EXISTS` и `CREATE INDEX IF NOT EXISTS`:
- ✅ Можно запускать много раз безопасно
- ✅ Не перезаписывает существующие данные
- ✅ Не удаляет существующие таблицы

### Транзакции

Миграции выполняются в транзакции:
```go
tx, err := d.db.BeginTx(ctx, nil)
defer tx.Rollback()

// Execute migrations...

tx.Commit()
```

Это гарантирует:
- ✅ Атомарность (все или ничего)
- ✅ Откат при ошибках
- ✅ Консистентность данных

## 📝 Логирование

### Успешный запуск (таблицы уже есть)
```
INFO: Database connection established successfully
INFO: Checking database schema...
INFO: Database schema is up to date
INFO: Processor started
```

### Первый запуск (создание таблиц)
```
INFO: Database connection established successfully
INFO: Checking database schema...
INFO: Required tables not found, applying migrations...
DEBUG: Executing migration statement: CREATE TABLE IF NOT EXISTS etalon_nomenclature...
DEBUG: Executing migration statement: CREATE INDEX IF NOT EXISTS...
INFO: Migrations applied successfully
INFO: Processor started
```

### Ошибка подключения к БД
```
ERROR: failed to open database: connection refused
```

### Ошибка миграции
```
ERROR: failed to ensure database schema: failed to apply migrations: ...
```

## 🛠️ Ручное управление (если нужно)

### Проверка таблиц

```bash
psql "YOUR_CONNECTION_STRING" -c "\dt"
```

Должны быть:
- `etalon_nomenclature`
- `processed_emails`

### Удаление таблиц (для полного сброса)

```bash
psql "YOUR_CONNECTION_STRING" <<EOF
DROP TABLE IF EXISTS etalon_nomenclature CASCADE;
DROP TABLE IF EXISTS processed_emails CASCADE;
EOF
```

После этого при следующем запуске приложение создаст таблицы заново.

### Ручное применение миграций

Если нужно по какой-то причине применить миграции вручную:

```bash
psql "YOUR_CONNECTION_STRING" -f migrations/001_init.sql
```

Это безопасно, так как используется `CREATE IF NOT EXISTS`.

## ❓ FAQ

### Что если таблицы уже созданы вручную?

Приложение проверит их наличие и пропустит создание:
```
INFO: Database schema is up to date
```

### Что если схема таблиц устарела?

Текущая версия проверяет только **наличие** таблиц, не их структуру. Если вы изменили структуру таблиц:
1. Либо удалите и пересоздайте таблицы
2. Либо примените миграцию изменений вручную

### Можно ли отключить автоматические миграции?

Да, закомментируйте вызов в `internal/db/postgres.go`:
```go
// if err := database.ensureSchema(ctx); err != nil {
//     return nil, fmt.Errorf("failed to ensure database schema: %w", err)
// }
```

### Что если миграция не удалась?

Приложение **не запустится** и выдаст ошибку:
```
ERROR: failed to ensure database schema: ...
```

Это правильное поведение - лучше не запускаться, чем работать с неправильной схемой БД.

## 🎓 Дополнительные материалы

- **Код миграций:** `internal/db/postgres.go`
- **SQL миграции:** `migrations/001_init.sql` (для справки)
- **Документация:** `README.md` → "Технические детали"

---

**Автор:** Встроено в приложение для упрощения деплоя
**Дата:** 2026-03-05
