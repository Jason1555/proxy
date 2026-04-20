package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger interface {
	Debug(msg string, fields zap.Field)
	Info(msg string, fields zap.Field)
	Warn(msg string, fields zap.Field)
	Error(msg string, fields zap.Field)
	Fatal(msg string, fields zap.Field)

	AccessDenied(ip, url, reason string)
	AccessAllowed(ip, url string)
	RuleAdded(ruleID, ruleType, ip string)
	RuleDeleted(ruleID string)
	ConfigReloaded(rulesCount int)
	CacheHit(ip string)
	CacheMiss(ip string)

	Sync() error
}

type ZapLogger struct {
	logger *zap.Logger
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
		logger: zapLogger,
	}, nil
}

func (l *ZapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

func (l *ZapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *ZapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

func (l *ZapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

func (l *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *ZapLogger) AccessDenied(ip, url, reason string) {
	l.logger.Warn(
		"access_denied",
		zap.String("ip", ip),
		zap.String("url", url),
		zap.String("reason", reason),
	)
}

func (l *ZapLogger) AccessAllowed(ip, url string) {
	l.logger.Info(
		"access_allowed",
		zap.String("ip", ip),
		zap.String("url", url),
	)
}

func (l *ZapLogger) RuleAdded(ruleID, ruleType, ip string) {
	l.logger.Info(
		"rule_added",
		zap.String("rule_id", ruleID),
		zap.String("type", ruleType),
		zap.String("ip", ip),
	)
}

func (l *ZapLogger) RuleDeleted(ruleID string) {
	l.logger.Info(
		"rule_deleted",
		zap.String("rule_id", ruleID),
	)
}

func (l *ZapLogger) ConfigReloaded(rulesCount int) {
	l.logger.Info(
		"config_reloaded",
		zap.Int("rules_count", rulesCount),
	)
}

func (l *ZapLogger) CacheHit(ip string) {
	l.logger.Debug(
		"cache_hit",
		zap.String("ip", ip),
	)
}

func (l *ZapLogger) CacheMiss(ip string) {
	l.logger.Debug(
		"cache_missed",
		zap.String("ip", ip),
	)
}

func (l *ZapLogger) Sync() error {
	return l.logger.Sync()
}
