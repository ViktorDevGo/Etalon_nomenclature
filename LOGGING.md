# Логирование парсера

## Конфигурация

Парсер использует ротацию логов с автоматической очисткой старых файлов.

### Параметры:
- **Файл логов**: `/var/log/parser/parser.log` (внутри контейнера)
- **Локально**: `./logs/parser.log` (на хосте)
- **Максимальный размер файла**: 50 MB
- **Хранение**: только текущий день (старые логи удаляются автоматически)
- **Формат**: JSON для файла, консольный для stdout

### Уровни логирования:
- **Файл**: DEBUG и выше (все логи)
- **Консоль (Docker)**: INFO и выше (только важные события)

## Просмотр логов

### На сервере:

#### Последние логи из файла:
```bash
tail -f /root/Etalon_nomenclature/logs/parser.log
```

#### Последние 100 строк:
```bash
tail -100 /root/Etalon_nomenclature/logs/parser.log
```

#### Только сегодняшние логи:
```bash
cat /root/Etalon_nomenclature/logs/parser.log
```

#### Поиск по логам:
```bash
# Найти ошибки
grep -i "error" /root/Etalon_nomenclature/logs/parser.log

# Найти письма от БИГМАШИН
grep -i "bigm.pro" /root/Etalon_nomenclature/logs/parser.log

# Найти blacklist события
grep -i "blacklisted" /root/Etalon_nomenclature/logs/parser.log
```

### Через Docker (консоль):
```bash
docker-compose -f /root/Etalon_nomenclature/docker-compose.yml logs -f --tail=50
```

## Ротация логов

### Автоматическая очистка:
- Логи старше **1 дня** удаляются автоматически
- Резервные копии **НЕ создаются** (MaxBackups: 0)
- При достижении 50 MB файл НЕ ротируется, а продолжает расти (чтобы не потерять логи текущего дня)

### Структура логов:

#### JSON формат (в файле):
```json
{
  "level": "INFO",
  "timestamp": "2026-03-07T10:30:45.123+03:00",
  "caller": "service/processor.go:142",
  "msg": "Processing email",
  "from": "m.timoshenkova@bigm.pro",
  "subject": "Прайс-лист"
}
```

#### Консольный формат (Docker):
```
2026-03-07T10:30:45.123+03:00	INFO	service/processor.go:142	Processing email	{"from": "m.timoshenkova@bigm.pro"}
```

## Диагностика проблем

### БИГМАШИН не обрабатывается:
```bash
# Проверить, приходят ли письма
grep "m.timoshenkova@bigm.pro" logs/parser.log

# Проверить фильтр blacklist
grep "blacklisted" logs/parser.log

# Проверить ошибки парсинга
grep -i "error.*price" logs/parser.log

# Проверить вставку в БД
grep "Transaction committed" logs/parser.log
```

### Bitrix24 не фильтруется:
```bash
# Проверить, что фильтр работает
grep "bitrix24" logs/parser.log

# Должно быть:
# "Email from blacklisted domain, skipping"
```

## Локальный доступ к логам

Логи доступны на хосте в директории:
```
/root/Etalon_nomenclature/logs/parser.log
```

Можно скачать на локальную машину:
```bash
scp root@c37e696087932476c61fd621.twc1.net:/root/Etalon_nomenclature/logs/parser.log ./
```
