package bencode

import (
	"bytes"
	"fmt"
	"testing"
)

func TestParser(t *testing.T) {
	data := []byte{100, 56, 58, 99, 111, 109, 112, 108, 101, 116, 101, 105, 51, 101, 49, 48, 58, 100, 111, 119, 110, 108, 111, 97, 100, 101, 100, 105, 51, 50, 101, 49, 48, 58, 105, 110, 99, 111, 109, 112, 108, 101, 116, 101, 105, 53, 101, 56, 58, 105, 110, 116, 101, 114, 118, 97, 108, 105, 49, 56, 49, 49, 101, 49, 50, 58, 109, 105, 110, 32, 105, 110, 116, 101, 114, 118, 97, 108, 105, 54, 48, 101, 53, 58, 112, 101, 101, 114, 115, 52, 56, 58, 124, 225, 94, 99, 144, 125, 139, 162, 86, 191, 19, 136, 157, 254, 20, 199, 19, 136, 172, 104, 88, 226, 19, 136, 182, 84, 181, 252, 88, 92, 14, 191, 222, 73, 160, 204, 111, 199, 250, 13, 105, 228, 111, 250, 73, 147, 198, 166, 54, 58, 112, 101, 101, 114, 115, 54, 48, 58, 101}

	d, err := parser(bytes.NewReader(data))
	if err != nil {
		t.Error(err)
	}
	fmt.Println(d)
}
