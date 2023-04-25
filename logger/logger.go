package logger

import "go.uber.org/zap"

var (
	Logger = zap.NewNop()
	Sugar  = Logger.Sugar()
	Cli, _ = zap.NewDevelopment(zap.IncreaseLevel(zap.InfoLevel))
)

func SetLogger(l *zap.Logger) {
	Logger = l
	Sugar = l.Sugar()
}

func SetCliLogger(l *zap.Logger) {
	Cli = l
}
