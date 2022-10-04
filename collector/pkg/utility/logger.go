package utility

import (
	"github.com/tiqqe/go-logger"
)

type KeyValue struct {
	K string
	V interface{}
}

func (kv KeyValue) KV() (string, interface{}) { return kv.K, kv.V }

func CreateEntry(message, code string, err error, extras ...KeyValue) *logger.LogEntry {
	entry := &logger.LogEntry{
		Message:   message,
		ErrorCode: code,
	}

	if err != nil {
		entry.ErrorMessage = err.Error()
	}

	if len(extras) > 0 {
		for _, e := range extras {
			entry.SetKey(e.KV())
		}
	}

	return entry
}

// LogError creates an entry of error message using the logger.Error function
func LogError(err error, code string, message string, kv ...KeyValue) {
	kv = append(kv, KeyValue{K: "Integration Flow", V: "Lambda Layer Collector"})
	logger.Error(CreateEntry(message, code, err, kv...))
}
