package bencode

import (
	"bufio"
	"bytes"
	"errors"
)

var (
	ErrNotDictionary = errors.New("not a dictionary")
)

// TDOO: 允许使用 key1.key2.key3 形式获取嵌套数据
// GetRaw 通过 key 获取原始数据
func GetRaw(data []byte, key string) ([]byte, error) {
	reader := bufio.NewReader(bytes.NewReader(data))
	first, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}
	if first != 'd' {
		return nil, ErrNotDictionary
	}

	for {
		k, err := decodeString(reader)
		if err != nil {
			return nil, err
		}
		if k != key {
			scans(reader, nil)
			continue
		}

		buf := bytes.NewBuffer(nil)
		err = scans(reader, buf)
		return buf.Bytes(), err

	}
}

func scans(r *bufio.Reader, w *bytes.Buffer) error {
	b, err := r.Peek(1)
	if err != nil {
		return err
	}

	switch b[0] {
	case 'i':
		return scansInt(r, w)
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return scansString(r, w)
	case 'l', 'd':
		return scansDL(r, w)
	default:
		return errors.New("invalid data")
	}
}

func scansInt(r *bufio.Reader, w *bytes.Buffer) error {
	var b byte
	var err error

	for {
		b, err = r.ReadByte()
		if err != nil {
			return err
		}

		if w != nil {
			w.WriteByte(b)
		}

		if b == 'e' {
			return nil
		}
	}
}

func scansString(r *bufio.Reader, w *bytes.Buffer) error {
	len := 0
	for {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		if w != nil {
			w.WriteByte(b)
		}
		if b == ':' {
			break
		}
		len = len*10 + int(b-'0')
	}

	for i := 0; i < len; i++ {
		b, err := r.ReadByte()
		if err != nil {
			return err
		}
		if w != nil {
			w.WriteByte(b)
		}
	}

	return nil
}

func scansDL(r *bufio.Reader, w *bytes.Buffer) error {
	b, err := r.ReadByte()
	if err != nil {
		return err
	}
	if w != nil {
		w.WriteByte(b)
	}
	for {
		err = scans(r, w)
		if err != nil {
			return err
		}

		if b == 'd' {
			err = scans(r, w)
			if err != nil {
				return err
			}
		}

		b, err := r.Peek(1)
		if err != nil {
			return err
		}

		if b[0] == 'e' {
			bb, err := r.ReadByte()
			if w != nil {
				w.WriteByte(bb)
			}
			return err
		}
	}
}
