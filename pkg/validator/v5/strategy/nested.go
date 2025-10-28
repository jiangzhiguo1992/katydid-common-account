package strategy

import (
	"fmt"
	v5 "katydid-common-account/pkg/validator/v5"
	"katydid-common-account/pkg/validator/v5/core"
	error2 "katydid-common-account/pkg/validator/v5/error"
	"reflect"
)

// NestedStrategy 嵌套验证策略
// 职责：递归验证嵌套的结构体
type NestedStrategy struct {
	engine   *v5.ValidatorEngine
	maxDepth int
}

// NewNestedStrategy 创建嵌套验证策略
func NewNestedStrategy(engine *v5.ValidatorEngine, maxDepth int) *NestedStrategy {
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
func (s *NestedStrategy) Validate(target any, ctx *v5.ValidationContext) error {
	// 获取反射值
	val := reflect.ValueOf(target)
	if !val.IsValid() {
		return nil
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
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
			if ctx.Depth >= s.maxDepth {
				ctx.AddError(
					error2.NewFieldError("Struct", "max_depth").
						WithMessage(fmt.Sprintf("maximum validation depth of %d exceeded", s.maxDepth)),
				)
				break
			}

			// 创建子上下文，保持深度和上下文信息
			subCtx := v5.NewValidationContext(ctx.Scene, ctx.MaxErrors).
				WithContext(ctx.Context).
				WithErrors(ctx.errors)
			subCtx.Depth = ctx.Depth + 1
			subCtx.Metadata = ctx.Metadata

			// 使用子上下文进行递归验证
			fieldValue := field.Interface()
			if err := s.engine.validateWithContext(fieldValue, subCtx); err != nil {
				// 如果返回错误，直接中断
				return err
			}

			// 将子上下文的错误添加到父上下文
			if subCtx.HasErrors() {
				ctx.AddErrors(subCtx.GetErrors())
			}
		}
	}

	return nil
}
