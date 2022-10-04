package utility

import (
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NopCoreLogger returns a logger option as no-op core logger.
func NopCoreLogger() []zap.Option {
	nopCore := zap.WrapCore(func(zapcore.Core) zapcore.Core {
		return zapcore.NewNopCore()
	})

	return []zap.Option{nopCore}
}

// CustomLoggerOptions
// Reference: https://pkg.go.dev/go.uber.org/zap#example-package-AdvancedConfiguration
func CustomLoggerOptions() []zap.Option {
	// Defines level-handling
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.WarnLevel
	})

	core := zapcore.NewCore(zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()), zapcore.AddSync(io.Discard), highPriority)
	highPriorityLogger := zap.WrapCore(func(zapcore.Core) zapcore.Core { return core })

	return []zap.Option{
		highPriorityLogger,
		zap.WithCaller(true),
	}
}
