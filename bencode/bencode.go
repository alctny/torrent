package bencode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

type BenType uint8

const (
	BenNone BenType = iota // 树的根结点，允许 Value 为空
	BenInt
	BenStr
	BenLst
	BenDir
)

var (
	ErrType = errors.New("type error")
	// TODO : 这些方法并不能帮助排查错误，需要优化
	ErrInt        = errors.New("type error, not int")
	ErrStr        = errors.New("type error, not string")
	ErrLst        = errors.New("type error, not list")
	ErrDic        = errors.New("type error, not dict")
	ErrColon      = errors.New("need ':', but not")
	ErrUnknowByte = errors.New("unknow byte")
)

type benObject struct {
	_type  BenType
	_value any
}

// TODO: 兼容负数，浮点数
// decodeInt 解析 Int
func decodeInt(data io.Reader) (int64, error) {
	reader := bufio.NewReader(data)
	first, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}

	if first != 'i' {
		return 0, ErrInt
	}

	var res int64
	var flag int64 = 1
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, errors.Join(ErrInt, err)
		}

		switch b {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			res = res*10 + int64(b-'0')
		case '-':
			if res != 0 {
				return 0, ErrInt
			}
			flag = -1
		case 'e':
			return res * flag, nil
		default:
			return 0, ErrInt
		}
	}
}

// decodeString 解析 string
func decodeString(data io.Reader) (string, error) {
	reader := bufio.NewReader(data)
	len := 0
	var b byte
	var err error

	for {
		b, err = reader.ReadByte()
		if err != nil {
			return "", ErrStr
		}
		if !(b >= '0' && b <= '9') {
			if b != ':' {
				return "", ErrColon
			}
			break
		}
		len = len*10 + int(b-'0')
	}
	if len == 0 {
		return "", nil
	}

	strBuf := make([]byte, len)

	// reader.Read 虽然效率高，但存在一个问题，如果字符串长度超过缓冲区，需要二次读取
	// reader.ReadByte() 虽然读取效率低一些，但是代码会更加简洁易懂
	for i := 0; i < len; i++ {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		strBuf[i] = b
	}

	return string(strBuf), nil
}

func decodeList(buf io.Reader) ([]benObject, error) {
	reader := bufio.NewReader(buf)
	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if b != 'l' {
		return nil, ErrLst
	}

	res := []benObject{}
	for {
		first, err := reader.Peek(1)
		if err != nil {
			return nil, err
		}

		switch first[0] {
		case 'e':
			reader.ReadByte()
			return res, nil

		case 'i':
			i, err := decodeInt(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, benObject{BenInt, i})

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			bstr, err := decodeString(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, benObject{BenStr, bstr})

		case 'l':
			lis, err := decodeList(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, benObject{BenLst, lis})

		case 'd':
			dic, err := decodeDict(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, benObject{BenDir, dic})

		default:
			return nil, ErrUnknowByte
		}

	}

}

func decodeDict(buf io.Reader) (map[string]benObject, error) {
	reader := bufio.NewReader(buf)
	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if b != 'd' {
		return nil, ErrDic
	}

	res := map[string]benObject{}
	for {
		first, err := reader.Peek(1)
		if err != nil {
			return nil, err
		}

		if first[0] == 'e' {
			reader.ReadByte()
			return res, nil
		}

		// parser key
		key, err := decodeString(reader)
		if err != nil {
			return nil, err
		}
		if _, ok := res[key]; ok {
			return nil, fmt.Errorf("dupile key: %s", key)
		}

		// parser value
		first, err = reader.Peek(1)
		if err != nil {
			return nil, err
		}

		switch first[0] {
		case 'i':
			i, err := decodeInt(reader)
			if err != nil {
				return nil, err
			}
			res[key] = benObject{BenInt, i}

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			s, err := decodeString(reader)
			if err != nil {
				return nil, err
			}
			res[key] = benObject{BenStr, s}

		case 'l':
			lis, err := decodeList(reader)
			if err != nil {
				return nil, err
			}
			res[key] = benObject{BenLst, lis}

		case 'd':
			dic, err := decodeDict(reader)
			if err != nil {
				return nil, err
			}
			res[key] = benObject{BenDir, dic}

		case 'e':
			return nil, fmt.Errorf("direct has key but no value")

		default:
			return nil, ErrUnknowByte
		}
	}

}

// TODO: 把 io.Reader 改为 bufio.Reader，因为是私有函数，并非对外提供的接口
// 不需要考虑兼容 io.Reader 和做无意义的转化 && 封装
func parser(buf io.Reader) (*benObject, error) {
	reader := bufio.NewReader(buf)
	first, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}

	var res *benObject

	switch first[0] {
	case 'i':
		var i int64
		i, err = decodeInt(reader)
		res = &benObject{BenInt, i}

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		var s string
		s, err = decodeString(reader)
		res = &benObject{BenStr, s}

	case 'l':
		var lis []benObject
		lis, err = decodeList(reader)
		res = &benObject{BenLst, lis}

	case 'd':
		var dis map[string]benObject
		dis, err = decodeDict(reader)
		res = &benObject{BenDir, dis}

	default:
		return nil, ErrType
	}

	return res, err
}

// encodeInt 编码 int
func encodeInt(w *bytes.Buffer, i int64) error {
	err := w.WriteByte('i')
	if err != nil {
		return err
	}
	_, err = w.WriteString(fmt.Sprint(i))
	if err != nil {
		return err
	}
	return w.WriteByte('e')
}

// encodeString 编码 string
func encodeString(bw *bytes.Buffer, s string) error {
	length := len(s)
	if length == 0 {
		return nil
	}
	_, err := bw.WriteString(fmt.Sprint((length)))
	if err != nil {
		return err
	}
	bw.WriteByte(':')
	_, err = bw.WriteString(s)
	return err
}
