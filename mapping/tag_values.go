package mapping

import "strings"

func splitTagValues(value string) []string {
	if !strings.Contains(value, ";") {
		return []string{value}
	}
	parts := strings.Split(value, ";")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		values = append(values, trimmed)
	}
	if len(values) == 0 {
		return []string{value}
	}
	return values
}

func mappingValueMatches(values map[Value]struct{}, tagValue string, splitValues bool) bool {
	if !splitValues {
		_, ok := values[Value(tagValue)]
		return ok
	}
	for _, value := range splitTagValues(tagValue) {
		if _, ok := values[Value(value)]; ok {
			return true
		}
	}
	return false
}
