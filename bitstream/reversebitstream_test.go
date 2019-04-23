package bitstream

import (
	"testing"
)

func TestReverseBitStream(t *testing.T) {
	var data [256]byte
	for idx := range data {
		data[idx] = byte(idx)
	}

	bs := NewReversebitstream(data[:])

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
		if byte(x) != data[len(data)-1-i] {
			t.Errorf("Mööp %d", x)
			return
		}
	}

	//half byte aligned

	bs = NewReversebitstream(data[:])

	//first byte aligned just for safety
	for i := 0; i < len(data)*2; i++ {
		x, err := bs.Read(4)
		if err != nil {
			t.Error(err.Error())
			return
		}
		if i%2 == 0 {
			if byte(x) != (data[len(data)-(i/2)-1] >> 4) {
				t.Errorf("Mööp %d", x)
				return
			}
		} else {
			if byte(x) != (data[len(data)-(i/2)-1] & 0xF) {
				t.Errorf("Mööp %d", x)
				return
			}
		}
	}

	//read 8*5bit = 40 bit = 5 byte and a full byte to check
	bs = NewReversebitstream(data[:])

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
		if byte(x) != data[len(data)-index-1] {
			t.Errorf("X: %d, Should be %d", x, data[len(data)-index-1])
		}
	}

	//read 8*3bit = 24 bit = 3 byte and a full byte to check
	bs = NewReversebitstream(data[:])

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
		if byte(x) != data[len(data)-index-1] {
			t.Errorf("X: %d, Should be %d", x, data[len(data)-index-1])
		}
	}

	//read 6+4*3+6bit = 24 bit = 3 byte and a full byte to check
	bs = NewReversebitstream(data[:])

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
		if byte(x) != data[len(data)-index-1] {
			t.Errorf("X: %d, Should be %d", x, data[len(data)-index-1])
		}
	}

	//read 3*7+3bit = 24 bit = 3 byte and a full byte to check
	bs = NewReversebitstream(data[:])

	for i := 0; i < len(data)/4; i++ {
		for j := 0; j < 3; j++ {
			_, err := bs.Read(7)
			if err != nil {
				t.Error(err.Error())
				return
			}
		}
		_, err := bs.Read(3)
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
		if byte(x) != data[len(data)-index-1] {
			t.Errorf("X: %d, Should be %d", x, data[len(data)-index-1])
		}
	}
}

func TestEdges(t *testing.T) {
	data := []byte{64, 58, 169, 224}
	rbs := NewReversebitstream(data)

	x, _ := rbs.Read(3)
	if x != 7 {
		panic(x)
	}
	x, _ = rbs.Read(4)
	if x != 0 {
		panic(x)
	}
	x, _ = rbs.Read(4)
	if x != 5 {
		panic(x)
	}

	//first byte and 3 bit

	x, _ = rbs.Read(1)
	if x != 0 {
		panic("DD")
	}
	x, _ = rbs.Read(3)
	if x != 4 {
		panic("DD")
	}
	x, _ = rbs.Read(5)
	if x != 19 {
		panic("DD")
	}

	//second byte and 4 bit

	x, _ = rbs.Read(1)
	if x != 1 {
		panic("DD")
	}
	x, _ = rbs.Read(3)
	if x != 2 {
		panic("DD")
	}

	//third byte

	x, _ = rbs.Read(3)
	if x != 2 {
		panic("DD")
	}
	x, _ = rbs.Read(0)
	if x != 0 {
		panic("DD")
	}
	x, _ = rbs.Read(4)
	if x != 0 {
		panic("DD")
	}
}
