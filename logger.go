package edgex

//
// Author: 陈哈哈 yoojiachen@gmail.com
//

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ZapLoggerConfig = zap.Config{
	Level:       zap.NewAtomicLevelAt(zap.DebugLevel),
	Development: false,
	Encoding:    "console",
	EncoderConfig: zapcore.EncoderConfig{
		// Keys can be anything except the empty string.
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		MessageKey:     "M",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	},
	OutputPaths:      []string{"stdout"},
	ErrorOutputPaths: []string{"stderr"},
}

var ZapLogger = NewZapLogger()
var ZapSugarLogger = NewZapSugarLogger()

var log = ZapSugarLogger

func ZapLoggerConfig() zap.Config {
	return _ZapLoggerConfig
}

func NewZapLogger() *zap.Logger {
	logger, _ := _ZapLoggerConfig.Build()
	return logger
}

func NewZapSugarLogger() *zap.SugaredLogger {
	return ZapLogger.Sugar()
}
