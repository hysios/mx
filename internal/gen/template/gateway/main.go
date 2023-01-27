package main

import (
	"github.com/hysios/mx"
	"github.com/hysios/mx/logger"
	"go.uber.org/zap"
)

func main() {
	(&mx.Gateway{}).Serve(":8080")
}

func init() {
	cfg, _ := zap.NewDevelopment()
	logger.SetLogger(cfg)
}
