package gunfish

import (
	"bytes"
	"fmt"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

var (
	baseTimestamp time.Time
)

func init() {
	baseTimestamp = time.Now()
}

func miniTS() int {
	return int(time.Since(baseTimestamp) / time.Second)
}

// LtsvFormatter is ltsv format for logrus
type LtsvFormatter struct {
	DisableTimestamp bool
	TimestampFormat  string
	DisableSorting   bool
}

// Format entry
func (f *LtsvFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}

	if !f.DisableSorting {
		sort.Strings(keys)
	}

	b := &bytes.Buffer{}

	prefixFieldClashes(entry.Data)

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.RFC3339
	}

	f.appendKeyValue(b, "level", entry.Level.String())

	if entry.Message != "" {
		f.appendKeyValue(b, "msg", entry.Message)
	}

	for _, key := range keys {
		f.appendKeyValue(b, key, entry.Data[key])
	}

	if !f.DisableTimestamp {
		f.appendKeyValue(b, "time", entry.Time.Format(timestampFormat))
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func needsQuoting(text string) bool {
	for _, ch := range text {
		if !((ch >= 'a' && ch <= 'z') ||
			(ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') ||
			ch == '-' || ch == '.') {
			return false
		}
	}
	return true
}

func (f *LtsvFormatter) appendKeyValue(b *bytes.Buffer, key string, value interface{}) {

	b.WriteString(key)
	b.WriteByte(':')

	switch value := value.(type) {
	case string:
		if needsQuoting(value) {
			b.WriteString(value)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	case int, int64, int32:
		fmt.Fprintf(b, "%d", value)
	case float64, float32:
		fmt.Fprintf(b, "%f", value)
	case error:
		errmsg := value.Error()
		if needsQuoting(errmsg) {
			b.WriteString(errmsg)
		} else {
			fmt.Fprintf(b, "%q", value)
		}
	default:
		fmt.Fprintf(b, "\"%v\"", value)
	}

	b.WriteByte('\t')
}

func prefixFieldClashes(data logrus.Fields) {
	if _, ok := data["time"]; ok {
		data["fields.time"] = data["time"]
	}

	if _, ok := data["msg"]; ok {
		data["fields.msg"] = data["msg"]
	}

	if _, ok := data["level"]; ok {
		data["fields.level"] = data["level"]
	}
}
