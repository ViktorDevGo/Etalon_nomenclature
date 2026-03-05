# Contributing to Etalon Nomenclature

Спасибо за интерес к проекту! Мы приветствуем любой вклад.

## Как внести вклад

### Reporting Bugs

Если вы нашли баг:
1. Проверьте, что баг еще не был зарегистрирован в Issues
2. Создайте новый Issue с подробным описанием:
   - Шаги для воспроизведения
   - Ожидаемое поведение
   - Фактическое поведение
   - Версия Go, ОС
   - Логи (если применимо)

### Suggesting Features

Для предложения новых функций:
1. Проверьте, что функция еще не предложена
2. Создайте Issue с описанием:
   - Проблема, которую решает функция
   - Предлагаемое решение
   - Альтернативы (если есть)

### Code Contributions

1. **Fork репозитория**
   ```bash
   git clone https://github.com/your-username/etalon-nomenclature.git
   cd etalon-nomenclature
   ```

2. **Создайте ветку для изменений**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Внесите изменения**
   - Следуйте code style проекта
   - Добавьте тесты для новой функциональности
   - Обновите документацию при необходимости

4. **Запустите тесты**
   ```bash
   make test
   make lint
   ```

5. **Commit изменений**
   ```bash
   git add .
   git commit -m "feat: add new feature"
   ```

   Следуйте [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` — новая функция
   - `fix:` — исправление бага
   - `docs:` — изменения в документации
   - `test:` — добавление тестов
   - `refactor:` — рефакторинг кода
   - `chore:` — обновление зависимостей, etc.

6. **Push в ваш fork**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Создайте Pull Request**
   - Опишите изменения
   - Ссылайтесь на связанные Issues
   - Дождитесь review

## Code Style

### Go Code

- Следуйте [Effective Go](https://golang.org/doc/effective_go.html)
- Используйте `gofmt` для форматирования
- Запускайте `golangci-lint` перед commit

```bash
# Форматирование
gofmt -w .

# Линтинг
golangci-lint run ./...
```

### Комментарии

- Документируйте публичные функции и типы
- Используйте godoc формат
- Комментарии на русском или английском

Пример:
```go
// ProcessEmail обрабатывает email с Excel вложениями
// и сохраняет данные в базу данных
func ProcessEmail(ctx context.Context, email Email) error {
    // ...
}
```

### Тесты

- Пишите unit тесты для новой функциональности
- Используйте table-driven tests
- Стремитесь к coverage > 70%

Пример:
```go
func TestParser_parseFloat(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    float64
        wantErr bool
    }{
        {
            name:    "simple number",
            input:   "100",
            want:    100,
            wantErr: false,
        },
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Development Setup

### Требования

- Go 1.22+
- Docker и Docker Compose
- PostgreSQL (для интеграционных тестов)
- Make

### Локальная разработка

1. **Клонирование**
   ```bash
   git clone https://github.com/prokoleso/etalon-nomenclature.git
   cd etalon-nomenclature
   ```

2. **Установка зависимостей**
   ```bash
   go mod download
   ```

3. **Копирование конфигурации**
   ```bash
   cp config.example.yaml config.yaml
   # Отредактируйте config.yaml
   ```

4. **Запуск**
   ```bash
   make run
   ```

5. **Тесты**
   ```bash
   make test
   make test-coverage
   ```

### Docker разработка

```bash
# Сборка
make docker-build

# Запуск
make docker-up

# Логи
make docker-logs

# Остановка
make docker-down
```

## Pull Request Process

1. Убедитесь, что все тесты проходят
2. Обновите README.md и другую документацию
3. Обновите CHANGELOG.md
4. PR должен быть reviewed минимум одним мейнтейнером
5. После одобрения, PR будет смержен

## Versioning

Мы используем [SemVer](http://semver.org/) для версионирования:

- **MAJOR** — несовместимые изменения API
- **MINOR** — новая функциональность (обратно совместимая)
- **PATCH** — исправления багов (обратно совместимые)

## License

Внося вклад в проект, вы соглашаетесь с тем, что ваш код будет под [MIT License](LICENSE).

## Questions?

Если у вас есть вопросы:
- Откройте Issue с вопросом
- Свяжитесь с мейнтейнерами

Спасибо за ваш вклад! 🎉
