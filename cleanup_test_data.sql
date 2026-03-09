-- Удалить тестовые записи из processed_emails
DELETE FROM processed_emails 
WHERE message_id LIKE 'test-%';

-- Показать результат
SELECT message_id FROM processed_emails ORDER BY processed_at DESC;
