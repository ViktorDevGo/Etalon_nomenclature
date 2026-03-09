-- =============================================================================
-- КОМПЛЕКСНОЕ ИСПРАВЛЕНИЕ: Унификация provider + Повторная обработка писем
-- =============================================================================
--
-- Проблема: Отсутствуют диски за 06-08.03.2026 (~10,000 записей)
-- Причина: Письма обработаны ДО добавления disk parser (08.03.2026 16:14)
--
-- Решение:
-- 1. Унифицировать названия provider: "БРИНЕКС" → "ГРУППА БРИНЕКС"
-- 2. Очистить processed_emails за 06-08.03 для повторной обработки
--
-- =============================================================================

\echo ''
\echo '========================================='
\echo 'ШАГ 1: Текущее состояние БД'
\echo '========================================='
\echo ''

\echo 'price_disks - распределение по provider:'
SELECT provider, COUNT(*) as count, MIN(email_date) as first_date, MAX(email_date) as last_date
FROM price_disks
GROUP BY provider
ORDER BY provider;

\echo ''
\echo 'price_tires - распределение по provider:'
SELECT provider, COUNT(*) as count, MIN(email_date) as first_date, MAX(email_date) as last_date
FROM price_tires
GROUP BY provider
ORDER BY provider;

\echo ''
\echo 'processed_emails - письма за 06-09.03:'
SELECT
    DATE(email_date) as date,
    COUNT(*) as emails_count
FROM processed_emails
WHERE DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-09'
GROUP BY DATE(email_date)
ORDER BY date;

\echo ''
\echo '========================================='
\echo 'ШАГ 2: Унификация названия provider'
\echo '========================================='
\echo ''

-- Обновляем price_disks
UPDATE price_disks
SET provider = 'ГРУППА БРИНЕКС'
WHERE provider = 'БРИНЕКС';

\echo '✓ Обновлена таблица price_disks'

-- Обновляем price_tires
UPDATE price_tires
SET provider = 'ГРУППА БРИНЕКС'
WHERE provider = 'БРИНЕКС';

\echo '✓ Обновлена таблица price_tires'

\echo ''
\echo '========================================='
\echo 'ШАГ 3: Очистка processed_emails'
\echo '========================================='
\echo ''

\echo 'Письма которые будут удалены для повторной обработки:'
SELECT
    message_id,
    email_date,
    processed_at,
    EXTRACT(DAY FROM email_date) as day
FROM processed_emails
WHERE DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08'
ORDER BY email_date;

-- Удаляем записи
DELETE FROM processed_emails
WHERE DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08';

\echo '✓ Удалены записи из processed_emails за 06-08.03'
\echo ''
\echo 'Письма будут автоматически обработаны в следующем цикле (макс 1 минута)'

\echo ''
\echo '========================================='
\echo 'ШАГ 4: Проверка результата'
\echo '========================================='
\echo ''

\echo 'price_disks после унификации:'
SELECT provider, COUNT(*) as count, MIN(email_date) as first_date, MAX(email_date) as last_date
FROM price_disks
GROUP BY provider
ORDER BY provider;

\echo ''
\echo 'price_tires после унификации:'
SELECT provider, COUNT(*) as count, MIN(email_date) as first_date, MAX(email_date) as last_date
FROM price_tires
GROUP BY provider
ORDER BY provider;

\echo ''
\echo 'processed_emails после очистки:'
SELECT COUNT(*) as total_emails
FROM processed_emails;

\echo ''
\echo '========================================='
\echo '✅ ГОТОВО!'
\echo '========================================='
\echo ''
\echo 'Следующие шаги:'
\echo '1. Подождите 1-2 минуты для автоматической обработки'
\echo '2. Проверьте логи: docker compose logs -f app'
\echo '3. Проверьте количество дисков (ожидается ~22,700):'
\echo ''
\echo 'SELECT provider, COUNT(*) FROM price_disks GROUP BY provider;'
\echo ''
