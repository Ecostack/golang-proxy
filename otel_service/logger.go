package otel_service

import (
	"context"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"http-proxy/config"
)

var Logger *otelzap.Logger

func msgFieldsToMsg(msg string, fields ...zapcore.Field) (string, []zapcore.Field) {
	result := msg
	// Build the message by appending the fields

	nonStringFields := make([]zapcore.Field, 0)

	for _, field := range fields {
		if field.Type == zapcore.StringType {
			result += ", " + field.Key + "=" + field.String
		} else {
			nonStringFields = append(nonStringFields, field)
		}
	}
	return result, nonStringFields
}

func Info(ctx context.Context, logger *otelzap.Logger, msg string, fields ...zapcore.Field) {
	result, leftOverFields := msgFieldsToMsg(msg, fields...)
	logger.Ctx(ctx).Info(result, leftOverFields...)
}

func Warn(ctx context.Context, logger *otelzap.Logger, msg string, fields ...zapcore.Field) {
	result, leftOverFields := msgFieldsToMsg(msg, fields...)
	logger.Ctx(ctx).Warn(result, leftOverFields...)
}

func Error(ctx context.Context, logger *otelzap.Logger, msg string, fields ...zapcore.Field) {
	result, leftOverFields := msgFieldsToMsg(msg, fields...)
	logger.Ctx(ctx).Error(result, leftOverFields...)
}

func Fatal(ctx context.Context, logger *otelzap.Logger, msg string, fields ...zapcore.Field) {
	result, leftOverFields := msgFieldsToMsg(msg, fields...)
	logger.Ctx(ctx).Fatal(result, leftOverFields...)
}

func InitLogger() {
	var logger *zap.Logger

	if config.Production {
		logger = zap.Must(zap.NewProduction())
	} else {
		logger = zap.Must(zap.NewDevelopment())
	}

	zapLogger := otelzap.New(logger, otelzap.WithMinLevel(logger.Level()), otelzap.WithCallerDepth(1))

	Logger = zapLogger

	otelzap.ReplaceGlobals(zapLogger)
}
