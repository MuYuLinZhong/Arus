package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// L 是 zap.Logger 的实例,全局构成;
var L *zap.Logger

func Init(mode string) {
	var cfg zap.Config
	//mode 只能是 debug 或 release
	if mode == "release" {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	}
	// 时间格式化;
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// 输出到 stdout,不以文件进行保存
	cfg.OutputPaths = []string{"stdout"}

	var err error
	// L进行全局实示例构建
	L, err = cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic("failed to init logger: " + err.Error())
	}
}

func Info(msg string, fields ...zap.Field)  { L.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { L.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { L.Error(msg, fields...) }
func Debug(msg string, fields ...zap.Field) { L.Debug(msg, fields...) }

func Fatal(msg string, fields ...zap.Field) {
	L.Fatal(msg, fields...)
	os.Exit(1)
}

// 刷写日志到文件或者输出;
func Sync() { _ = L.Sync() }
