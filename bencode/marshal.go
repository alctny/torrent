package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"unicode"
)

var (
	ErrUnsupportedType = errors.New("unsupported type")
)

// TODO: support type any
// Marshal marshal any type to bencode bytes
func Marshal(a any) ([]byte, error) {
	ref := elem(reflect.ValueOf(a))
	buf := bytes.NewBuffer(nil)

	err := marshal(buf, ref)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func marshal(buf *bytes.Buffer, ref reflect.Value) error {
	switch ref.Kind() {

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return encodeInt(buf, ref.Int())

	case reflect.String:
		return encodeString(buf, ref.String())

	case reflect.Slice, reflect.Array:
		return marshalList(buf, ref)

	case reflect.Map:
		return marshalDict(buf, ref)

	case reflect.Struct:
		return marshalStruct(buf, ref)

	default:
		fmt.Println(ref.CanInt(), ref.CanUint(), ref.CanConvert(reflect.TypeOf("")))
		return ErrUnsupportedType
	}
}

func marshalList(buf *bytes.Buffer, ref reflect.Value) error {
	refEl := elem(ref)
	length := refEl.Len()
	err := buf.WriteByte('l')
	if err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		iel := elem(refEl.Index(i))
		err := marshal(buf, iel)
		if err != nil {
			return err
		}
	}

	return buf.WriteByte('e')
}

func marshalDict(buf *bytes.Buffer, ref reflect.Value) error {
	refEl := elem(ref)
	length := refEl.Len()
	err := buf.WriteByte('d')
	if err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		k := refEl.MapKeys()[i]
		v := refEl.MapIndex(k)
		vel := elem(v)
		if vel.Kind() == reflect.String && vel.IsZero() {
			continue
		}

		err := encodeString(buf, k.String())
		if err != nil {
			return err
		}
		err = marshal(buf, elem(v))
		if err != nil {
			return err
		}
	}

	return buf.WriteByte('e')
}

func marshalStruct(buf *bytes.Buffer, ref reflect.Value) error {
	refEl := elem(ref)
	length := refEl.NumField()
	err := buf.WriteByte('d')
	if err != nil {
		return err
	}

	for i := 0; i < length; i++ {
		fieldName := refEl.Type().Field(i).Name
		if unicode.IsLower(rune(fieldName[0])) {
			continue
		}
		tag := refEl.Type().Field(i).Tag.Get("bencode")
		if tag == "-" {
			continue
		}
		if tag == "" {
			tag = fieldName
		}

		err = encodeString(buf, tag)
		if err != nil {
			return err
		}
		field := elem(refEl.Field(i))
		err = marshal(buf, field)
		if err != nil {
			return err
		}
	}

	return buf.WriteByte('e')
}
