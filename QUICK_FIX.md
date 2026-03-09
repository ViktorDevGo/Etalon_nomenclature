# Быстрое исправление: Восстановление дисков за 06-08.03

## 🎯 Что случилось?

Disk parser был добавлен **08.03.2026 в 16:14**, а письма за 06-08.03 были обработаны **ДО** этого.
Результат: шины записались, диски — нет (парсера еще не было).

## ⚡ Быстрое решение (1 команда)

```bash
ssh u2827889@62.122.170.171 << 'EOF'
cd etalon-nomenclature
docker compose exec -T app sh -c "PGPASSWORD='<3B;hH5EFDH' psql -h c37e696087932476c61fd621.twc1.net -U gen_user -d default_db -f /app/fix_all.sql"
EOF
```

Эта команда:
1. ✅ Унифицирует названия: "БРИНЕКС" → "ГРУППА БРИНЕКС"
2. ✅ Очистит processed_emails за 06-08.03
3. ✅ Письма автоматически перепарсятся (макс 1 минута)

## 📊 Проверка результата (через 2 минуты)

```bash
ssh u2827889@62.122.170.171 << 'EOF'
cd etalon-nomenclature
docker compose exec -T app sh -c "PGPASSWORD='<3B;hH5EFDH' psql -h c37e696087932476c61fd621.twc1.net -U gen_user -d default_db -c 'SELECT provider, COUNT(*) as count, MIN(DATE(email_date)) as first, MAX(DATE(email_date)) as last FROM price_disks GROUP BY provider;'"
EOF
```

**Ожидаемый результат:**
```
      provider      | count | first      | last
--------------------+-------+------------+------------
 ГРУППА БРИНЕКС     | ~20000| 2026-03-06 | 2026-03-09
 ЗАПАСКА            | ~2500 | 2026-03-06 | 2026-03-09
```

## 📋 Детали

Подробное описание проблемы и решения: `FIX_MISSING_DISKS.md`

## ⚠️ Примечание

Код работает правильно! Проблема была только в том, что старые письма были обработаны до добавления disk parser.
