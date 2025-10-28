package plugin

import (
	"log"

	"katydid-common-account/pkg/validator/v6/core"
)

// LoggingPlugin 日志插件
// 职责：记录验证过程
// 设计模式：插件模式
type LoggingPlugin struct {
	enabled bool
	logger  Logger
}

// Logger 日志接口（依赖倒置）
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// defaultLogger 默认日志实现
type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...any) {
	log.Printf("[INFO] "+msg, args...)
}

func (l *defaultLogger) Error(msg string, args ...any) {
	log.Printf("[ERROR] "+msg, args...)
}

// NewLoggingPlugin 创建日志插件
func NewLoggingPlugin() *LoggingPlugin {
	return &LoggingPlugin{
		enabled: true,
		logger:  &defaultLogger{},
	}
}

// WithLogger 设置日志器
func (p *LoggingPlugin) WithLogger(logger Logger) *LoggingPlugin {
	p.logger = logger
	return p
}

// Name 插件名称
func (p *LoggingPlugin) Name() string {
	return "LoggingPlugin"
}

// Init 初始化插件
func (p *LoggingPlugin) Init(config map[string]any) error {
	if config == nil {
		return nil
	}

	// 读取配置
	if enabled, ok := config["enabled"].(bool); ok {
		p.enabled = enabled
	}

	return nil
}

// BeforeValidate 验证前钩子
func (p *LoggingPlugin) BeforeValidate(ctx core.ValidationContext) error {
	if !p.enabled {
		return nil
	}

	req := ctx.Request()
	p.logger.Info("开始验证", "scene", req.Scene, "target", req.Target)
	return nil
}

// AfterValidate 验证后钩子
func (p *LoggingPlugin) AfterValidate(ctx core.ValidationContext) error {
	if !p.enabled {
		return nil
	}

	errorCount := ctx.ErrorCollector().Count()
	if errorCount > 0 {
		p.logger.Error("验证失败", "errorCount", errorCount)
	} else {
		p.logger.Info("验证成功")
	}

	return nil
}

// Enabled 是否启用
func (p *LoggingPlugin) Enabled() bool {
	return p.enabled
}
