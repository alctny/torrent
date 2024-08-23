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
	bo, err := Parser(reader)
	if err != nil {
		return err
	}

	switch bo.Type {
	case BenInt, BenStr:
		return set(rv, bo.Value)

	case BenLst:
		return unmarshalList(bo.Value.([]BenObject), rv.Elem())

	case BenDir:
		return unmarshalDir(bo.Value.(map[string]BenObject), rv.Elem())

	default:
		return errors.New("unknown type")
	}

}

// unmarshalList 反序列化列表类型的 BenObject
func unmarshalList(bens []BenObject, ref reflect.Value) error {
	refEl := elem(ref)
	typ := refEl.Type()
	if refEl.Kind() == reflect.Slice {
		typ = refEl.Type().Elem()
	}

	newSlice := reflect.MakeSlice(reflect.SliceOf(typ), len(bens), len(bens))
	var err error
	for i, v := range bens {
		el := reflect.New(typ)
		switch v.Type {
		case BenInt, BenStr:
			err = set(el, v.Value)
		case BenLst:
			err = unmarshalList(v.Value.([]BenObject), el)
		case BenDir:
			err = unmarshalDir(v.Value.(map[string]BenObject), el)
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
func unmarshalDir(bens map[string]BenObject, ref reflect.Value) error {
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
func unmarshalMap(bens map[string]BenObject, ref reflect.Value) error {
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
		switch benv.Type {
		case BenInt, BenStr:
			err = set(val, benv.Value)
		case BenLst:
			err = unmarshalList(benv.Value.([]BenObject), val)
		case BenDir:
			err = unmarshalDir(benv.Value.(map[string]BenObject), val)
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
func unmarshalStruct(bens map[string]BenObject, ref reflect.Value) error {
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
		switch v.Type {
		case BenInt, BenStr:
			err = set(el.Field(i), v.Value)
		case BenLst:
			err = unmarshalList(v.Value.([]BenObject), el.Field(i))
		case BenDir:
			err = unmarshalDir(v.Value.(map[string]BenObject), el.Field(i))
		}
		if err != nil {
			return err
		}
	}
	return nil
}
