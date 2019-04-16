package decompression_test

import (
	"bytes"
	"github.com/killingspark/sparkzsdt/decompression"
	"io/ioutil"
	"testing"
)

type nullWriter struct{}

func (nw *nullWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func BenchmarkDecompress(b *testing.B) {
	f, err := ioutil.ReadFile("../cmd/testingdata/ubuntu.zst")
	if err != nil {
		panic(err.Error())
	}
	content := bytes.NewBuffer(f)

	outfile := &nullWriter{}

	//outfile, err := os.OpenFile("/dev/null", os.O_WRONLY, 777)
	if err != nil {
		panic(err.Error())
	}

	dec := decompression.NewFrameDecompressor(content, outfile)
	err = dec.Decompress()
}
