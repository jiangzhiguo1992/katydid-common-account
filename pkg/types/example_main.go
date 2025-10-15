package types

import (
	"fmt"
)

func main() {
	// 创建 Extras
	e := NewExtras()

	// 设置各种类型的值
	e.Set("name", "张三")
	e.Set("age", 25)
	e.Set("vip", true)
	e.Set("score", 98.5)

	// 读取值
	if name, ok := e.GetString("name"); ok {
		fmt.Printf("姓名: %s\n", name)
	}

	if age, ok := e.GetInt("age"); ok {
		fmt.Printf("年龄: %d\n", age)
	}

	if vip, ok := e.GetBool("vip"); ok {
		fmt.Printf("VIP: %v\n", vip)
	}

	if score, ok := e.GetFloat64("score"); ok {
		fmt.Printf("分数: %.1f\n", score)
	}

	fmt.Printf("\n总共 %d 个字段\n", e.Len())
	fmt.Printf("所有键: %v\n", e.Keys())

	fmt.Println("\n✅ Extras 测试成功！")
}
