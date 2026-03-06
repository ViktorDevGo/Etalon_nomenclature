package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Симуляция текущей логики parseFloat
func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}

	fmt.Printf("  Исходное значение: '%s'\n", s)

	// Remove spaces
	s = strings.ReplaceAll(s, " ", "")
	fmt.Printf("  После удаления пробелов: '%s'\n", s)

	// Count dots and commas to determine format
	dotCount := strings.Count(s, ".")
	commaCount := strings.Count(s, ",")
	fmt.Printf("  Точек: %d, Запятых: %d\n", dotCount, commaCount)

	if dotCount > 1 {
		// Multiple dots: "4.124.00" - dots are thousand separators except the last one
		parts := strings.Split(s, ".")
		fmt.Printf("  Части (split by '.'): %v\n", parts)
		if len(parts) > 0 {
			wholePart := strings.Join(parts[:len(parts)-1], "")
			decimalPart := parts[len(parts)-1]
			s = wholePart + "." + decimalPart
			fmt.Printf("  Преобразовано в: '%s'\n", s)
		}
	} else if commaCount > 1 {
		parts := strings.Split(s, ",")
		fmt.Printf("  Части (split by ','): %v\n", parts)
		if len(parts) > 0 {
			wholePart := strings.Join(parts[:len(parts)-1], "")
			decimalPart := parts[len(parts)-1]
			s = wholePart + "." + decimalPart
			fmt.Printf("  Преобразовано в: '%s'\n", s)
		}
	} else if dotCount == 1 && commaCount == 1 {
		dotPos := strings.Index(s, ".")
		commaPos := strings.Index(s, ",")
		fmt.Printf("  Позиция точки: %d, запятой: %d\n", dotPos, commaPos)
		if dotPos < commaPos {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
			fmt.Printf("  Европейский формат, преобразовано в: '%s'\n", s)
		} else {
			s = strings.ReplaceAll(s, ",", "")
			fmt.Printf("  US формат, преобразовано в: '%s'\n", s)
		}
	} else if commaCount == 1 {
		s = strings.ReplaceAll(s, ",", ".")
		fmt.Printf("  Русский формат, преобразовано в: '%s'\n", s)
	}

	result, err := strconv.ParseFloat(s, 64)
	fmt.Printf("  Итоговое число: %.2f\n", result)
	return result, err
}

func main() {
	fmt.Println("🔬 ДИАГНОСТИКА ПАРСИНГА ЦЕН")
	fmt.Println("=" + string(make([]byte, 70)))
	fmt.Println()

	// Тестовые случаи на основе проблемных значений
	testCases := []string{
		"11.09.11.01.09.31", // Потенциально дата/время
		"16.11.75.90.000",   // Другой паттерн
		"4.124.00",          // Нормальная цена с разделителями
		"1234,56",           // Русский формат
		"1,234.56",          // US формат
		"1.234,56",          // Европейский формат
		"5000",              // Простое число
	}

	for i, tc := range testCases {
		fmt.Printf("Тест #%d:\n", i+1)
		_, err := parseFloat(tc)
		if err != nil {
			fmt.Printf("  ❌ ОШИБКА: %v\n", err)
		}
		fmt.Println()
	}

	fmt.Println("💡 ВЫВОД:")
	fmt.Println("Если в Excel попадают даты в формате '11.09.11.01.09.31',")
	fmt.Println("они интерпретируются как цены с множественными разделителями тысяч,")
	fmt.Println("что приводит к огромным числам типа 110911010931.")
	fmt.Println()
	fmt.Println("РЕШЕНИЕ: Нужно добавить валидацию результата или детектировать даты.")
}
