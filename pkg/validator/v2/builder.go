package v2

import (
	"github.com/go-playground/validator/v10"
)

// ============================================================================
// 构建器实现 - 建造者模式：流式API构建复杂验证器
// ============================================================================

// validatorBuilder 验证器构建器
type validatorBuilder struct {
	cache          CacheManager
	pool           ValidatorPool
	strategy       ValidationStrategy
	errorFormatter ErrorFormatter
	tagName        string
	validate       *validator.Validate
	customFuncs    map[string]validator.Func
}

// NewValidatorBuilder 创建验证器构建器
func NewValidatorBuilder() ValidatorBuilder {
	return &validatorBuilder{
		validate:    validator.New(),
		tagName:     "validate",
		customFuncs: make(map[string]validator.Func),
	}
}

// WithCache 设置缓存
func (b *validatorBuilder) WithCache(cache CacheManager) ValidatorBuilder {
	b.cache = cache
	return b
}

// WithPool 设置对象池
func (b *validatorBuilder) WithPool(pool ValidatorPool) ValidatorBuilder {
	b.pool = pool
	return b
}

// WithStrategy 设置验证策略
func (b *validatorBuilder) WithStrategy(strategy ValidationStrategy) ValidatorBuilder {
	b.strategy = strategy
	return b
}

// WithErrorFormatter 设置错误格式化器
func (b *validatorBuilder) WithErrorFormatter(formatter ErrorFormatter) ValidatorBuilder {
	b.errorFormatter = formatter
	return b
}

// WithTagName 设置标签名称
func (b *validatorBuilder) WithTagName(tagName string) ValidatorBuilder {
	b.tagName = tagName
	return b
}

// RegisterCustomValidation 注册自定义验证函数
func (b *validatorBuilder) RegisterCustomValidation(tag string, fn validator.Func) ValidatorBuilder {
	b.customFuncs[tag] = fn
	return b
}

// Build 构建验证器
func (b *validatorBuilder) Build() (Validator, error) {
	// 注册所有自定义验证函数
	for tag, fn := range b.customFuncs {
		if err := b.validate.RegisterValidation(tag, fn); err != nil {
			return nil, err
		}
	}

	// 设置标签名称
	if b.tagName != "" {
		b.validate.SetTagName(b.tagName)
	}

	// 创建验证器实例
	v := &defaultValidator{
		validate:       b.validate,
		cache:          b.cache,
		pool:           b.pool,
		strategy:       b.strategy,
		errorFormatter: b.errorFormatter,
		tagName:        b.tagName,
		useCache:       b.cache != nil,
		usePool:        b.pool != nil,
	}

	return v, nil
}

// ============================================================================
// 预配置的构建器工厂函数 - 简化常用场景
// ============================================================================

// NewDefaultValidator 创建默认验证器（带缓存和对象池）
func NewDefaultValidator() (Validator, error) {
	return NewValidatorBuilder().
		WithCache(NewCacheManager()).
		WithPool(NewValidatorPool()).
		WithStrategy(NewDefaultStrategy()).
		Build()
}

// NewSimpleValidator 创建简单验证器（无缓存和对象池）
func NewSimpleValidator() (Validator, error) {
	return NewValidatorBuilder().
		WithStrategy(NewDefaultStrategy()).
		Build()
}

// NewPerformanceValidator 创建高性能验证器（LRU缓存 + 对象池）
func NewPerformanceValidator(cacheSize int) (Validator, error) {
	return NewValidatorBuilder().
		WithCache(NewLRUCacheManager(cacheSize)).
		WithPool(NewValidatorPool()).
		WithStrategy(NewDefaultStrategy()).
		Build()
}

// NewFailFastValidator 创建快速失败验证器
func NewFailFastValidator() (Validator, error) {
	return NewValidatorBuilder().
		WithStrategy(NewFailFastStrategy()).
		Build()
}

// NewPartialValidator 创建部分字段验证器
func NewPartialValidator(fields ...string) (Validator, error) {
	return NewValidatorBuilder().
		WithStrategy(NewPartialStrategy(fields...)).
		Build()
}
