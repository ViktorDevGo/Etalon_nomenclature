-- Очистка processed_emails для повторной обработки писем за 06-08.03.2026
-- Причина: письма были обработаны ДО добавления disk parser (08.03.2026 16:14)
-- Шины записались, но диски не парсились (парсера еще не было)

-- 1. Проверим какие письма будут удалены
SELECT
    message_id,
    email_date,
    processed_at
FROM processed_emails
WHERE DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08'
ORDER BY email_date;

-- 2. Посчитаем количество
SELECT COUNT(*) as emails_to_reprocess
FROM processed_emails
WHERE DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08';

-- 3. Удалим записи для повторной обработки
DELETE FROM processed_emails
WHERE DATE(email_date) BETWEEN '2026-03-06' AND '2026-03-08';

-- 4. Проверим результат
SELECT COUNT(*) as remaining_emails
FROM processed_emails;

-- Примечание: После выполнения этого скрипта нужно дождаться следующего цикла
-- обработки (макс 1 минута), и письма будут перепарсены с дисками!
