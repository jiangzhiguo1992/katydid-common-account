package strategy

import (
	v5 "katydid-common-account/pkg/validator/v5"
	"katydid-common-account/pkg/validator/v5/context"
	"katydid-common-account/pkg/validator/v5/core"
	"katydid-common-account/pkg/validator/v5/err"
	"reflect"
)

// NestedStrategy 嵌套验证策略
// 职责：递归验证嵌套的结构体
type NestedStrategy struct {
	engine   *v5.ValidatorEngine
	maxDepth int8
}

// NewNestedStrategy 创建嵌套验证策略
func NewNestedStrategy(engine *v5.ValidatorEngine, maxDepth int8) core.IValidationStrategy {
	return &NestedStrategy{
		engine:   engine,
		maxDepth: maxDepth,
	}
}

// Type 策略类型
func (s *NestedStrategy) Type() core.StrategyType {
	return core.StrategyTypeNested
}

// Priority 优先级
func (s *NestedStrategy) Priority() int8 {
	return 20
}

// Validate 执行嵌套验证
func (s *NestedStrategy) Validate(target any, ctx core.IValidationContext) {
	// 获取反射值
	val := reflect.ValueOf(target)
	if !val.IsValid() {
		return
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	// 遍历所有字段
	typ := val.Type()
	numField := val.NumField()

	for i := 0; i < numField; i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过不可访问的字段
		if !field.CanInterface() || !field.IsValid() {
			continue
		}

		// 跳过 nil 指针
		if field.Kind() == reflect.Ptr && field.IsNil() {
			continue
		}

		// 只处理匿名（嵌入）结构体字段
		fieldKind := field.Kind()
		if fieldKind == reflect.Ptr && !field.IsNil() {
			fieldKind = field.Elem().Kind()
		}

		// 只处理匿名（嵌入）的结构体字段
		if fieldKind == reflect.Struct && fieldType.Anonymous {
			// 超过最大深度，记录错误并停止验证
			if ctx.Depth() >= s.maxDepth {
				ctx.AddError(err.NewFieldError("Struct", "max_depth"))
				break
			}

			// 创建子上下文，保持深度和上下文信息
			subCtx := context.NewValidationContext(
				ctx.Scene(),
				ctx.MaxErrors(),
				context.WithContext(ctx.Context()),
				context.WithDepth(ctx.Depth()+1),
				context.WithErrors(ctx.Errors()),
				context.WithMetadata(ctx.Metadata()),
			)

			// 使用子上下文进行递归验证
			fieldValue := field.Interface()
			if e := s.engine.ValidateWithContext(fieldValue, subCtx); e != nil {
				// 如果返回错误，直接中断
				ctx.AddError(err.NewFieldErrorWithMessage(e.Error()))
				return
			}

			// 将子上下文的错误添加到父上下文
			if subCtx.HasErrors() {
				ctx.AddErrors(subCtx.Errors())
			}
		}
	}

	return
}
