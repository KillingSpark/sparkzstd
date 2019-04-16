package bitstream

import (
	"bufio"
	"bytes"
	"testing"
)

func TestBitStream(t *testing.T) {
	var data [256]byte
	for idx := range data {
		data[idx] = byte(idx)
	}

	src := bufio.NewReader(bytes.NewReader(data[:]))
	bs := NewBitstream(src)

	//for i := 0; i < len(data)*8; i++ {
	//	x, err := bs.Read(1)
	//	if err != nil {
	//		panic(err.Error())
	//	}
	//	if i%8 == 0 {
	//		print("\n")
	//	}
	//	print(x)
	//}
	//panic("a")

	//first byte aligned just for safety
	for i := 0; i < len(data); i++ {
		x, err := bs.Read(8)
		if err != nil {
			t.Error(err.Error())
			return
		}
		if byte(x) != data[i] {
			t.Errorf("Mööp %d", x)
			return
		}
	}

	//half byte aligned

	src = bufio.NewReader(bytes.NewReader(data[:]))
	bs = NewBitstream(src)

	//first byte aligned just for safety
	for i := 0; i < len(data)*2; i++ {
		x, err := bs.Read(4)
		if err != nil {
			t.Error(err.Error())
			return
		}
		if i%2 == 0 {
			if byte(x) != (data[i/2] & 0xF) {
				t.Errorf("Mööp %d", x)
				return
			}
		} else {
			if byte(x) != (data[i/2] >> 4) {
				t.Errorf("Mööp %d", x)
				return
			}
		}
	}

	//read 8*5bit = 40 bit = 5 byte and a full byte to check
	src = bufio.NewReader(bytes.NewReader(data[:]))
	bs = NewBitstream(src)

	for i := 0; i < len(data)/9; i++ {
		for j := 0; j < 8; j++ {
			_, err := bs.Read(5)
			if err != nil {
				t.Error(err.Error())
				return
			}
		}
		x, err := bs.Read(8)
		if err != nil {
			t.Error(err.Error())
			return
		}

		index := ((i + 1) * 6) - 1
		if byte(x) != data[index] {
			t.Errorf("X: %d, Should be %d", x, data[index])
		}
	}

	//read 8*3bit = 24 bit = 3 byte and a full byte to check
	src = bufio.NewReader(bytes.NewReader(data[:]))
	bs = NewBitstream(src)

	for i := 0; i < len(data)/4; i++ {
		for j := 0; j < 8; j++ {
			_, err := bs.Read(3)
			if err != nil {
				t.Error(err.Error())
				return
			}
		}
		x, err := bs.Read(8)
		if err != nil {
			t.Error(err.Error())
			return
		}

		index := ((i + 1) * 4) - 1
		if byte(x) != data[index] {
			t.Errorf("X: %d, Should be %d", x, data[index])
		}
	}

	//read 6+4*3+6bit = 24 bit = 3 byte and a full byte to check
	src = bufio.NewReader(bytes.NewReader(data[:]))
	bs = NewBitstream(src)

	for i := 0; i < len(data)/4; i++ {
		_, err := bs.Read(6)
		if err != nil {
			t.Error(err.Error())
			return
		}
		for j := 0; j < 4; j++ {
			_, err := bs.Read(3)
			if err != nil {
				t.Error(err.Error())
				return
			}
		}
		_, err = bs.Read(6)
		if err != nil {
			t.Error(err.Error())
			return
		}
		x, err := bs.Read(8)
		if err != nil {
			t.Error(err.Error())
			return
		}

		index := ((i + 1) * 4) - 1
		if byte(x) != data[index] {
			t.Errorf("X: %d, Should be %d", x, data[index])
		}
	}
}
