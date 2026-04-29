package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Debugf(msg string, args ...any)
	Infof(msg string, args ...any)
	Warnf(msg string, args ...any)
	Errorf(msg string, args ...any)
	Fatalf(msg string, args ...any)
}

type ZapLogger struct {
	logger *zap.SugaredLogger
}

func NewZapLogger(config *Config) (*ZapLogger, error) {
	zapConfig := zap.NewProductionConfig()

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, err
	}
	zapConfig.Level = zap.NewAtomicLevelAt(level)

	zapConfig.Encoding = config.Encoding
	zapConfig.OutputPaths = config.OutputPaths
	zapConfig.ErrorOutputPaths = config.ErrorOutputPaths

	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapConfig.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	zapConfig.InitialFields = config.InitialFields

	zapLogger, err := zapConfig.Build(
		zap.WithCaller(config.WithCaller),
		zap.AddStacktrace(zap.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return &ZapLogger{
		logger: zapLogger.Sugar(),
	}, nil
}

func (l *ZapLogger) Debugf(msg string, args ...any) {
	l.logger.Debugf(msg, args...)
}

func (l *ZapLogger) Infof(msg string, args ...any) {
	l.logger.Infof(msg, args...)
}

func (l *ZapLogger) Warnf(msg string, args ...any) {
	l.logger.Warnf(msg, args...)
}

func (l *ZapLogger) Errorf(msg string, args ...any) {
	l.logger.Errorf(msg, args...)
}

func (l *ZapLogger) Fatalf(msg string, args ...any) {
	l.logger.Fatalf(msg, args...)
}
