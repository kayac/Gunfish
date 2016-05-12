package gunfish

import (
	"fmt"
	"runtime"

	"github.com/Sirupsen/logrus"
)

// LogWithFields wraps logrus's WithFields
func LogWithFields(fields map[string]interface{}) *logrus.Entry {
	_, file, line, _ := runtime.Caller(1)

	fields["file"] = file
	fields["line"] = fmt.Sprintf("%d", line)

	return logrus.WithFields(fields)
}
