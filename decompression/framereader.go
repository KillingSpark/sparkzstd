package decompression

import (
	"bytes"
	"io"
)

//FrameReader wraps a FrameDecompressor and probides the io.Reader interface
type FrameReader struct {
	fd     *FrameDecompressor
	buffer bytes.Buffer
}

//NewFrameReader creates the necessary buffers and the FrameDecompressor
func NewFrameReader(source io.Reader) (*FrameReader, error) {
	fr := &FrameReader{}
	fr.fd = NewFrameDecompressor(source, &fr.buffer)
	err := fr.fd.CheckMagicnum()
	if err != nil {
		return nil, err
	}

	err = fr.fd.DecodeFrameHeader()
	if err != nil {
		return nil, err
	}
	return fr, nil
}

func (fr *FrameReader) Read(target []byte) (int, error) {
	if fr.fd.CurrentBlock.Header.LastBlock {
		fr.fd.decodebuffer.Flush()
	}

	if fr.buffer.Len() >= len(target) {
		copy(target, fr.buffer.Next(len(target)))
		return len(target), nil
	}

	if fr.fd.CurrentBlock.Header.LastBlock {
		if fr.buffer.Len() > 0 {
			bs := fr.buffer.Bytes()
			fr.buffer.Reset()
			copy(target, bs)
			return len(bs), nil
		}
		return 0, io.EOF
	}

	oldSize := fr.buffer.Len()

	for fr.buffer.Len() == oldSize {
		err := fr.fd.DecodeNextBlock()
		if err != nil {
			return 0, err
		}
		fr.fd.BlockCounter++
	}
	buf := fr.buffer.Next(len(target))
	copy(target, buf)
	return len(buf), nil
}
