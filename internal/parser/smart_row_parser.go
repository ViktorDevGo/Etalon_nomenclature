package parser

import (
	"regexp"
	"strings"
	"unicode"
)

// smartFindArticle пытается найти артикул в строке, используя эвристики
// Артикул обычно:
// - Короткий (< 30 символов)
// - Содержит буквы и цифры
// - НЕ начинается с "Автошина", "А/ш" и т.д.
// - НЕ является ценой (не содержит только цифры и пробелы)
func smartFindArticle(cols []string, mapping *priceColumnMapping) (string, bool) {
	// Сначала пробуем стандартную колонку
	if mapping.article >= 0 && mapping.article < len(cols) {
		article := strings.TrimSpace(cols[mapping.article])
		if isLikelyArticle(article) {
			return article, true
		}
	}

	// Если не подошло, ищем в соседних колонках (0-3)
	for i := 0; i < 4 && i < len(cols); i++ {
		article := strings.TrimSpace(cols[i])
		if isLikelyArticle(article) {
			return article, true
		}
	}

	return "", false
}

// smartFindPrice пытается найти цену в строке
// Цена обычно:
// - Число (с пробелами или без)
// - В разумном диапазоне (100-500000)
// - НЕ слишком длинное число (не артикул)
func smartFindPrice(cols []string, mapping *priceColumnMapping, priceParser *PriceParser) (float64, bool) {
	// Сначала пробуем стандартную колонку
	if mapping.price >= 0 && mapping.price < len(cols) {
		priceStr := strings.TrimSpace(cols[mapping.price])
		if price, err := priceParser.parseFloat(priceStr); err == nil && price > 0 && price < 500000 {
			return price, true
		}
	}

	// Ищем в колонках 4-7 (обычно там цены)
	for i := 4; i < 8 && i < len(cols); i++ {
		priceStr := strings.TrimSpace(cols[i])
		if price, err := priceParser.parseFloat(priceStr); err == nil && price > 0 && price < 500000 {
			// Проверяем, что это не артикул (артикулы могут быть чисто числовые)
			if !isLikelyArticle(priceStr) {
				return price, true
			}
		}
	}

	return 0, false
}

// isLikelyArticle проверяет, похожа ли строка на артикул
func isLikelyArticle(s string) bool {
	s = strings.TrimSpace(s)

	// Пустая строка
	if s == "" {
		return false
	}

	// Слишком длинная (вероятно номенклатура)
	if len(s) > 30 {
		return false
	}

	// Начинается с номенклатурных префиксов
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "автошина") ||
		strings.HasPrefix(lower, "а/ш") ||
		strings.HasPrefix(lower, "на r") ||
		strings.HasPrefix(lower, "шина") {
		return false
	}

	// Содержит "б/у"
	if strings.Contains(lower, "б/у") {
		return false
	}

	// Слишком много пробелов (номенклатура)
	if strings.Count(s, " ") > 2 {
		return false
	}

	// Только цифры с 1-2 пробелами (вероятно цена типа "2 370")
	digitsAndSpaces := true
	spaceCount := 0
	for _, r := range s {
		if r == ' ' {
			spaceCount++
		} else if r < '0' || r > '9' {
			digitsAndSpaces = false
			break
		}
	}
	if digitsAndSpaces && spaceCount <= 2 && len(strings.ReplaceAll(s, " ", "")) < 6 {
		return false // Это цена
	}

	// Должен содержать хотя бы одну букву ИЛИ быть длинным числом (>= 6 цифр)
	hasLetter := false
	digitCount := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			digitCount++
		}
	}

	// Валидный артикул: есть буквы ИЛИ длинное число
	return hasLetter || digitCount >= 6
}

// isLikelyNomenclature проверяет, похожа ли строка на номенклатуру
func isLikelyNomenclature(s string) bool {
	s = strings.TrimSpace(s)
	lower := strings.ToLower(s)

	// Начинается с типичных префиксов
	if strings.HasPrefix(lower, "автошина") ||
		strings.HasPrefix(lower, "а/ш") ||
		strings.HasPrefix(lower, "шина") {
		return true
	}

	// Длинная строка с пробелами (более 40 символов)
	if len(s) > 40 && strings.Count(s, " ") > 3 {
		return true
	}

	// Содержит типичные паттерны номенклатуры
	nomenclaturePatterns := []string{" r1", " r2", "winter", "summer", "ice", "snow"}
	matchCount := 0
	for _, pattern := range nomenclaturePatterns {
		if strings.Contains(lower, pattern) {
			matchCount++
		}
	}

	return matchCount >= 2
}

// Regexp для проверки размера шины (например, "195/65", "205/55")
var sizePattern = regexp.MustCompile(`^\d{3}/\d{2}$`)

// isLikelySize проверяет, похожа ли строка на размер шины
func isLikelySize(s string) bool {
	s = strings.TrimSpace(s)
	return sizePattern.MatchString(s)
}
