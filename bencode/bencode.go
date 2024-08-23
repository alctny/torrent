package bencode

import (
	"bufio"
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

type BenObject struct {
	Type  BenType
	Value any
}

// // Int returns the integer value of the BenObject.
// func (b *BenObject) Int() (int64, error) {
// 	if b.Type != BenInt {
// 		return 0, ErrType
// 	}
// 	return b.Value.(int64), nil
// }

// // String returns the string value of the BenObject.
// func (b *BenObject) String() (string, error) {
// 	if b.Type != BenStr {
// 		return "", ErrType
// 	}
// 	return b.Value.(string), nil
// }

// func (b *BenObject) Array() ([]BenObject, error) {
// 	if b.Type != BenLst {
// 		return nil, ErrType
// 	}
// 	return b.Value.([]BenObject), nil
// }

// func (b *BenObject) Dictory() (map[string]BenObject, error) {
// 	if b.Type != BenDir {
// 		return nil, ErrType
// 	}
// 	return b.Value.(map[string]BenObject), nil
// }

type BenTree struct {
	Type  BenType
	Value *BenObject
	Nodes []*BenObject
}

// TODO: 兼容负数，浮点数
// DecodeInt 解析 Int
func DecodeInt(data io.Reader) (int64, error) {
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
			return 0, ErrInt
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

// DecodeString 解析 string
func DecodeString(data io.Reader) (string, error) {
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

	strBuf := make([]byte, len)

	// err = readN(data, strBuf)
	// if err != nil {
	// 	return "", err
	// }
	// return string(strBuf), nil

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

func DecodeList(buf io.Reader) ([]BenObject, error) {
	reader := bufio.NewReader(buf)
	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if b != 'l' {
		return nil, ErrLst
	}

	res := []BenObject{}
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
			i, err := DecodeInt(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, BenObject{BenInt, i})

		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			bstr, err := DecodeString(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, BenObject{BenStr, bstr})

		case 'l':
			lis, err := DecodeList(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, BenObject{BenLst, lis})

		case 'd':
			dic, err := DecodeDict(reader)
			if err != nil {
				return nil, err
			}
			res = append(res, BenObject{BenDir, dic})

		default:
			return nil, ErrUnknowByte
		}

	}

}

func DecodeDict(buf io.Reader) (map[string]BenObject, error) {
	reader := bufio.NewReader(buf)
	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if b != 'd' {
		return nil, ErrDic
	}

	res := map[string]BenObject{}
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
		key, err := DecodeString(reader)
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
			i, err := DecodeInt(reader)
			if err != nil {
				return nil, err
			}
			res[key] = BenObject{BenInt, i}

		case '1', '2', '3', '4', '5', '6', '7', '8', '9':
			s, err := DecodeString(reader)
			if err != nil {
				return nil, err
			}
			res[key] = BenObject{BenStr, s}

		case 'l':
			lis, err := DecodeList(reader)
			if err != nil {
				return nil, err
			}
			res[key] = BenObject{BenLst, lis}

		case 'd':
			dic, err := DecodeDict(reader)
			if err != nil {
				return nil, err
			}
			res[key] = BenObject{BenDir, dic}

		case 'e':
			return nil, fmt.Errorf("direct has key but no value")

		default:
			return nil, ErrUnknowByte
		}
	}

}

func Parser(buf io.Reader) (*BenObject, error) {
	reader := bufio.NewReader(buf)
	first, err := reader.Peek(1)
	if err != nil {
		return nil, err
	}

	var res *BenObject

	switch first[0] {
	case 'i':
		i, err := DecodeInt(reader)
		if err != nil {
			return nil, err
		}
		res = &BenObject{BenInt, i}

	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		s, err := DecodeString(reader)
		if err != nil {
			return nil, err
		}
		res = &BenObject{BenStr, s}

	case 'l':
		lis, err := DecodeList(reader)
		if err != nil {
			return nil, err
		}
		res = &BenObject{BenLst, lis}

	case 'd':
		dis, err := DecodeDict(reader)
		if err != nil {
			return nil, err
		}
		res = &BenObject{BenDir, dis}

	default:
		return nil, ErrType
	}

	return res, nil
}
