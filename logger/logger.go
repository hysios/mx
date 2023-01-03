package logger

import "go.uber.org/zap"

var (
	Logger = zap.NewNop()
	Cli, _ = zap.NewDevelopment(zap.IncreaseLevel(zap.InfoLevel))
)

func SetLogger(l *zap.Logger) {
	Logger = l
}

func SetCliLogger(l *zap.Logger) {
	Cli = l
}
