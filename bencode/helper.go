package bencode

import (
	"fmt"
	"reflect"
)

// set 设置 ref 最终指向的值为 val，无论 ret 是否是指针
// 要求 val 和 ref 最终指向值类型相同或可转化
// 你甚至可以使用 **int 初始化  ****int
func set(ref reflect.Value, val any) error {
	refEl := elem(ref)
	vv, ok := val.(reflect.Value)
	if !ok {
		vv = reflect.ValueOf(val)
	}
	vvel := elem(vv)

	if !vvel.CanConvert(refEl.Type()) {
		return fmt.Errorf("can not convert %s to %s", vv.Type(), ref.Type())
	}

	refEl.Set(vvel.Convert(refEl.Type()))
	return nil
}

// zero 初始化 v 为零值，如果是多层指针，则会递归初始化
func zero(v reflect.Value) {
	if v.Kind() != reflect.Pointer {
		v.Set(reflect.Zero(v.Type()))
		return
	}

	if v.IsNil() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	zero(v.Elem())
}

// elem 获取 ref 最终指向的值，如果是多层指针，则会递归获取最底层的值
func elem(ref reflect.Value) reflect.Value {
	el := ref
	for {
		if el.Kind() != reflect.Pointer {
			break
		}

		if el.IsNil() {
			zero(el)
		}
		el = el.Elem()
	}
	return el
}
