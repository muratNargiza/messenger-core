package logger

import (
	"context"
	"io"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.SugaredLogger

type zapLogger struct {
	sugared *zap.SugaredLogger
	level   zap.AtomicLevel
}

func init() {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	l, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic(err)
	}
	Log = l.Sugar()

	hlog.SetLogger(&zapLogger{
		sugared: l.Sugar(),
		level:   cfg.Level,
	})
}

func (z *zapLogger) Trace(v ...interface{}) {
	z.sugared.Debug(v...)
}

func (z *zapLogger) Debug(v ...interface{}) {
	z.sugared.Debug(v...)
}

func (z *zapLogger) Info(v ...interface{}) {
	z.sugared.Info(v...)
}

func (z *zapLogger) Notice(v ...interface{}) {
	z.sugared.Info(v...)
}

func (z *zapLogger) Warn(v ...interface{}) {
	z.sugared.Warn(v...)
}

func (z *zapLogger) Error(v ...interface{}) {
	z.sugared.Error(v...)
}

func (z *zapLogger) Fatal(v ...interface{}) {
	z.sugared.Fatal(v...)
}

func (z *zapLogger) Tracef(format string, v ...interface{}) {
	z.sugared.Debugf(format, v...)
}

func (z *zapLogger) Debugf(format string, v ...interface{}) {
	z.sugared.Debugf(format, v...)
}

func (z *zapLogger) Infof(format string, v ...interface{}) {
	z.sugared.Infof(format, v...)
}

func (z *zapLogger) Noticef(format string, v ...interface{}) {
	z.sugared.Infof(format, v...)
}

func (z *zapLogger) Warnf(format string, v ...interface{}) {
	z.sugared.Warnf(format, v...)
}

func (z *zapLogger) Errorf(format string, v ...interface{}) {
	z.sugared.Errorf(format, v...)
}

func (z *zapLogger) Fatalf(format string, v ...interface{}) {
	z.sugared.Fatalf(format, v...)
}

func (z *zapLogger) CtxTracef(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Debugf(format, v...)
}

func (z *zapLogger) CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Debugf(format, v...)
}

func (z *zapLogger) CtxInfof(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Infof(format, v...)
}

func (z *zapLogger) CtxNoticef(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Infof(format, v...)
}

func (z *zapLogger) CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Warnf(format, v...)
}

func (z *zapLogger) CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Errorf(format, v...)
}

func (z *zapLogger) CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	z.sugared.Fatalf(format, v...)
}

func (z *zapLogger) SetLevel(level hlog.Level) {
	var zapLevel zapcore.Level
	switch level {
	case hlog.LevelTrace, hlog.LevelDebug:
		zapLevel = zapcore.DebugLevel
	case hlog.LevelInfo, hlog.LevelNotice:
		zapLevel = zapcore.InfoLevel
	case hlog.LevelWarn:
		zapLevel = zapcore.WarnLevel
	case hlog.LevelError:
		zapLevel = zapcore.ErrorLevel
	case hlog.LevelFatal:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}
	z.level.SetLevel(zapLevel)
}

func (z *zapLogger) SetOutput(writer io.Writer) {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg.EncoderConfig),
		zapcore.AddSync(writer),
		z.level,
	)
	l := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	z.sugared = l.Sugar()
	Log = z.sugared
}

func Debug(v ...interface{}) {
	Log.Debug(v...)
}

func Info(v ...interface{}) {
	Log.Info(v...)
}

func Warn(v ...interface{}) {
	Log.Warn(v...)
}

func Error(v ...interface{}) {
	Log.Error(v...)
}

func Fatal(v ...interface{}) {
	Log.Fatal(v...)
}

func Debugf(format string, v ...interface{}) {
	Log.Debugf(format, v...)
}

func Infof(format string, v ...interface{}) {
	Log.Infof(format, v...)
}

func Warnf(format string, v ...interface{}) {
	Log.Warnf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	Log.Errorf(format, v...)
}

func Fatalf(format string, v ...interface{}) {
	Log.Fatalf(format, v...)
}

func CtxDebugf(ctx context.Context, format string, v ...interface{}) {
	Log.Debugf(format, v...)
}

func CtxInfof(ctx context.Context, format string, v ...interface{}) {
	Log.Infof(format, v...)
}

func CtxWarnf(ctx context.Context, format string, v ...interface{}) {
	Log.Warnf(format, v...)
}

func CtxErrorf(ctx context.Context, format string, v ...interface{}) {
	Log.Errorf(format, v...)
}

func CtxFatalf(ctx context.Context, format string, v ...interface{}) {
	Log.Fatalf(format, v...)
}
