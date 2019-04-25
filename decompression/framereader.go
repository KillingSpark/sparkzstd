package decompression

import (
	"bytes"
	"io"
)

//FrameReader wraps a FrameDecompressor and probides the io.Reader interface
type FrameReader struct {
	fd          *FrameDecompressor
	buffer      bytes.Buffer
	PrintStatus bool
	readTotal   int64
}

//NewFrameReader creates the necessary buffers and the FrameDecompressor
func NewFrameReader(source io.Reader) (*FrameReader, error) {
	fr := &FrameReader{}
	fr.fd = NewFrameDecompressor(source, &fr.buffer)
	//fr.fd.Verbose = true
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
	read, err := fr.readFromBuffer(target)
	newTotal := fr.readTotal + int64(read)
	if fr.PrintStatus {
		if fr.fd.frame.Header.FrameContentSize > 0 {
			oldprcnt := uint64(fr.readTotal*100) / fr.fd.frame.Header.FrameContentSize
			newprcnt := uint64(newTotal*100) / fr.fd.frame.Header.FrameContentSize
			if oldprcnt != newprcnt {
				print("Read bytes: ")
				print(newTotal)
				print(": ")
				print(newprcnt)
				println("%")
			}
		} else {
			print("Read bytes: ")
			println(fr.readTotal)
		}
	}
	fr.readTotal += int64(read)
	return read, err
}

func (fr *FrameReader) readFromBuffer(target []byte) (int, error) {
	if fr.fd.CurrentBlock.Header.LastBlock {
		fr.fd.decodebuffer.Flush()
	}

	if fr.buffer.Len() >= len(target) {
		copy(target, fr.buffer.Next(len(target)))
		return len(target), nil
	}

	if fr.fd.CurrentBlock.Header.LastBlock {
		fr.fd.decodebuffer.Flush()

		if fr.buffer.Len() > 0 {
			bs := fr.buffer.Bytes()
			fr.buffer.Reset()
			copy(target, bs)
			return len(bs), nil
		}
		return 0, io.EOF
	}

	oldSize := fr.buffer.Len()

	//decode until the first time the amount of output changes
	for fr.buffer.Len() == oldSize && !fr.fd.CurrentBlock.Header.LastBlock {
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
