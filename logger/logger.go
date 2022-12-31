package logger

import "go.uber.org/zap"

var Logger = zap.NewNop()

func SetLogger(l *zap.Logger) {
	Logger = l
}
