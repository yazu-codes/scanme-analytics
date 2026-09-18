package util

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// ParseJSONFileToHumanReadable reads a JSON file and converts it into an
// indented, human-readable text representation suitable for LLM prompts.
func ParseJSONFileToHumanReadable(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var sb strings.Builder
	writeReadable(&sb, parsed, 0)
	return sb.String(), nil
}

// ParseJSONToHumanReadable does the same, but from an already-loaded
// struct/object instead of a file (useful when you already have the data
// in memory, e.g. a model.Menu you just fetched).
func ParseJSONToHumanReadable(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal value: %w", err)
	}

	var parsed any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	var sb strings.Builder
	writeReadable(&sb, parsed, 0)
	return sb.String(), nil
}

func writeReadable(sb *strings.Builder, v any, indent int) {
	prefix := strings.Repeat("  ", indent)

	switch val := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			label := humanizeKey(k)
			child := val[k]

			switch child.(type) {
			case map[string]any, []any:
				sb.WriteString(fmt.Sprintf("%s%s:\n", prefix, label))
				writeReadable(sb, child, indent+1)
			default:
				sb.WriteString(fmt.Sprintf("%s%s: %s\n", prefix, label, formatScalar(child)))
			}
		}

	case []any:
		for i, item := range val {
			switch item.(type) {
			case map[string]any, []any:
				sb.WriteString(fmt.Sprintf("%s- Item %d:\n", prefix, i+1))
				writeReadable(sb, item, indent+1)
			default:
				sb.WriteString(fmt.Sprintf("%s- %s\n", prefix, formatScalar(item)))
			}
		}

	default:
		sb.WriteString(fmt.Sprintf("%s%s\n", prefix, formatScalar(val)))
	}
}

func formatScalar(v any) string {
	if v == nil {
		return "N/A"
	}
	switch val := v.(type) {
	case string:
		if val == "" {
			return "N/A"
		}
		return val
	case float64:
		// avoid ugly float formatting for whole numbers (e.g. prices, IDs)
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

// humanizeKey turns "picture_url" into "Picture Url", "menu_owner_name" into "Menu Owner Name"
func humanizeKey(key string) string {
	parts := strings.Split(key, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}

func JsonStringToArray(menuIds string) ([]int64, error) {
	var menuIDs []int64

	if err := json.Unmarshal(
		[]byte(menuIds),
		&menuIDs,
	); err != nil {
		return nil, err
	}

	return menuIDs, nil
}
