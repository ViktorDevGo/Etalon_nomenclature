-- Унификация названия поставщика "БРИНЕКС" → "ГРУППА БРИНЕКС"

-- Проверим текущее состояние
SELECT 'price_disks' as table_name, provider, COUNT(*) as count
FROM price_disks
GROUP BY provider
ORDER BY provider;

SELECT 'price_tires' as table_name, provider, COUNT(*) as count
FROM price_tires
GROUP BY provider
ORDER BY provider;

-- Обновим price_disks
UPDATE price_disks
SET provider = 'ГРУППА БРИНЕКС'
WHERE provider = 'БРИНЕКС';

-- Обновим price_tires
UPDATE price_tires
SET provider = 'ГРУППА БРИНЕКС'
WHERE provider = 'БРИНЕКС';

-- Проверим результат
SELECT 'После обновления - price_disks' as info, provider, COUNT(*) as count
FROM price_disks
GROUP BY provider
ORDER BY provider;

SELECT 'После обновления - price_tires' as info, provider, COUNT(*) as count
FROM price_tires
GROUP BY provider
ORDER BY provider;
