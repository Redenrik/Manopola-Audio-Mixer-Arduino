package runtime

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
)

// Fields captures structured log key/value attributes.
type Fields map[string]any

// Log emits a deterministic structured runtime log entry.
func Log(event string, fields Fields) {
	log.Print(StructuredMessage(event, fields))
}

// StructuredMessage returns a deterministic key/value log message with sorted field keys.
func StructuredMessage(event string, fields Fields) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("event=")
	b.WriteString(event)
	for _, k := range keys {
		b.WriteByte(' ')
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(formatValue(fields[k]))
	}
	return b.String()
}

func formatValue(v any) string {
	switch value := v.(type) {
	case string:
		return strconv.Quote(value)
	case error:
		return strconv.Quote(value.Error())
	default:
		return fmt.Sprint(v)
	}
}
