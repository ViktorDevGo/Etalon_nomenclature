#!/bin/bash

# Скрипт очистки проекта от временных и тестовых файлов
# Использование: ./cleanup_project.sh

set -e

echo "╔═══════════════════════════════════════════════════════════════════════╗"
echo "║           ОЧИСТКА ПРОЕКТА ОТ ВРЕМЕННЫХ ФАЙЛОВ                         ║"
echo "╚═══════════════════════════════════════════════════════════════════════╝"
echo ""

# Проверка, что мы в корне проекта
if [ ! -f "go.mod" ]; then
    echo "❌ Ошибка: запустите скрипт из корня проекта"
    exit 1
fi

echo "📁 Текущая директория: $(pwd)"
echo ""

# Спрашиваем подтверждение
read -p "❓ Удалить временные файлы? Это действие необратимо! (y/N): " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Отменено пользователем"
    exit 0
fi

echo ""
echo "🗑️  Удаление временных файлов..."
echo ""

# Счетчик удаленных файлов
COUNT=0

# Функция для безопасного удаления
safe_remove() {
    if [ -e "$1" ]; then
        rm -rf "$1"
        echo "  ✓ Удалено: $1"
        ((COUNT++))
    fi
}

# 1. Временные Go файлы в корне
echo "📦 Временные Go файлы:"
safe_remove "analyze_excel.go"
safe_remove "check_excel_sheets.go"
safe_remove "clear_processed.go"
safe_remove "clear_tables.go"
safe_remove "clear_tables.sql"
safe_remove "decode_subject.go"
safe_remove "test_deduplication.go"
safe_remove "test_detector.go"
safe_remove "test_parser.go"

# 2. Скомпилированные бинарники
echo ""
echo "🔧 Скомпилированные бинарники:"
safe_remove "app"
safe_remove "bin"

# 3. Логи
echo ""
echo "📝 Логи:"
safe_remove "app.log"

# 4. Директория scripts (вся)
echo ""
echo "📜 Директория scripts:"
if [ -d "scripts" ]; then
    echo "  ℹ️  Найдено $(find scripts -name "*.go" | wc -l) скриптов"
    safe_remove "scripts"
fi

# 5. Устаревшая документация
echo ""
echo "📄 Устаревшая документация:"
safe_remove "ANALYTICS_REPORT.md"
safe_remove "ANALYTICS_REPORT.txt"
safe_remove "AUTO_MIGRATIONS.md"
safe_remove "BUILD_VERIFICATION.md"
safe_remove "CHANGELOG.md"
safe_remove "CONTRIBUTING.md"
safe_remove "DEPLOY_ENV_SETUP.md"
safe_remove "DOCKER_ENV_MIGRATION.md"
safe_remove "INDEX.md"
safe_remove "LOGGING.md"
safe_remove "PROJECT_SUMMARY.md"
safe_remove "QUICKSTART.md"
safe_remove "STRUCTURE.md"

# 6. Опционально: .claude (настройки Claude)
echo ""
read -p "❓ Удалить .claude/ директорию? (y/N): " -n 1 -r
echo ""
if [[ $REPLY =~ ^[Yy]$ ]]; then
    safe_remove ".claude"
fi

echo ""
echo "═══════════════════════════════════════════════════════════════════════"
echo "✅ Очистка завершена!"
echo "📊 Удалено файлов/директорий: $COUNT"
echo ""
echo "⚠️  ВНИМАНИЕ: Не забудьте удалить файлы с секретами вручную:"
echo "   - config.yaml (содержит пароли БД и email)"
echo "   - .env (содержит переменные окружения)"
echo ""
echo "💡 Эти файлы должны быть только на production сервере!"
echo "   Они уже в .gitignore и не попадут в репозиторий."
echo ""
echo "═══════════════════════════════════════════════════════════════════════"
