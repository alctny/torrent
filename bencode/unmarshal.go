package bencode

import (
	"bytes"
	"errors"
	"reflect"
	"unicode"
)

// TODO:
// 在所有 unmarshal 中应该是先创建一个和目标数据类型一致的 reflect.Value
// 然后把这个值 Set 到目标变量上

var ErrNotPtr = errors.New("not a pointer or nil")

// TODO: 类型不匹配的时候不要直接 panic，而是返回一个 error
func Unmarshal(data []byte, res any) error {
	rv := reflect.ValueOf(res)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return ErrNotPtr
	}

	reader := bytes.NewReader(data)
	bo, err := parser(reader)
	if err != nil {
		return err
	}

	switch bo._type {
	case BenInt, BenStr:
		return set(rv, bo._value)

	case BenLst:
		return unmarshalList(bo._value.([]benObject), rv.Elem())

	case BenDir:
		return unmarshalDir(bo._value.(map[string]benObject), rv.Elem())

	default:
		return errors.New("unknown type")
	}

}

// unmarshalList 反序列化列表类型的 BenObject
func unmarshalList(bens []benObject, ref reflect.Value) error {
	refEl := elem(ref)
	typ := refEl.Type()
	if refEl.Kind() == reflect.Slice || refEl.Kind() == reflect.Array {
		typ = refEl.Type().Elem()
	}

	newSlice := reflect.MakeSlice(reflect.SliceOf(typ), len(bens), len(bens))
	var err error
	for i, v := range bens {
		el := reflect.New(typ)
		switch v._type {
		case BenInt, BenStr:
			err = set(el, v._value)
		case BenLst:
			err = unmarshalList(v._value.([]benObject), el)
		case BenDir:
			err = unmarshalDir(v._value.(map[string]benObject), el)
		}
		if err != nil {
			return err
		}
		err = set(newSlice.Index(i), el)
		if err != nil {
			return err
		}
	}

	err = set(ref, newSlice)
	if err != nil {
		return ErrColon
	}
	return nil
}

// unmarshalDir 反序列化字典类型的 BenObject
func unmarshalDir(bens map[string]benObject, ref reflect.Value) error {
	el := elem(ref)
	switch el.Kind() {
	case reflect.Map, reflect.Interface:
		return unmarshalMap(bens, el)
	case reflect.Struct:
		return unmarshalStruct(bens, el)
	default:
		return errors.New("type error: not a map or struct")
	}
}

// unmarshalMap 反序列化字典类型的 BenObject 到 map/any
func unmarshalMap(bens map[string]benObject, ref reflect.Value) error {
	refEl := elem(ref)
	keyTyp := reflect.TypeOf("")
	valTyp := ref.Type()
	if ref.Kind() == reflect.Map {
		valTyp = refEl.Type().Elem()
	}

	newMap := reflect.MakeMap(reflect.MapOf(keyTyp, valTyp))
	var err error
	for key, benv := range bens {
		val := reflect.New(valTyp)
		switch benv._type {
		case BenInt, BenStr:
			err = set(val, benv._value)
		case BenLst:
			err = unmarshalList(benv._value.([]benObject), val)
		case BenDir:
			err = unmarshalDir(benv._value.(map[string]benObject), val)
		}
		if err != nil {
			return err
		}
		newMap.SetMapIndex(reflect.ValueOf(key), val)
	}
	err = set(refEl, newMap)
	if err != nil {
		return err
	}

	return nil
}

// unmarshalStruct 反序列化字典类型的 BenObject 到 struct
func unmarshalStruct(bens map[string]benObject, ref reflect.Value) error {
	el := elem(ref)

	var err error
	for i := 0; i < el.NumField(); i++ {
		tag := el.Type().Field(i).Tag.Get("bencode")
		if tag == "" {
			tag = el.Type().Field(i).Name
		}
		if tag == "-" || unicode.IsLower(rune(el.Type().Field(i).Name[0])) {
			continue
		}

		v, ok := bens[tag]
		if !ok {
			continue
		}
		switch v._type {
		case BenInt, BenStr:
			err = set(el.Field(i), v._value)
		case BenLst:
			err = unmarshalList(v._value.([]benObject), el.Field(i))
		case BenDir:
			err = unmarshalDir(v._value.(map[string]benObject), el.Field(i))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
