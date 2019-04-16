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
	err := fr.fd.DecodeFrameHeader()
	if err != nil {
		return nil, err
	}
	return fr, nil
}

func (fr *FrameReader) Read(target []byte) (int, error) {
	if fr.buffer.Len() <= len(target) {
		copy(target, fr.buffer.Next(len(target)))
		return len(target), nil
	}

	oldSize := fr.buffer.Len()

	for fr.buffer.Len() == oldSize {
		err := fr.fd.DecodeNextBlockHeader()
		if err != nil {
			return 0, err
		}
		err = fr.fd.DecodeNextBlockContent()
		if err != nil {
			return 0, err
		}
	}
	buf := fr.buffer.Next(len(target))
	copy(target, buf)
	return len(buf), nil
}
