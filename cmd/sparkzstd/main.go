package main

import (
	"bufio"
	"fmt"
	"github.com/killingspark/sparkzsdt/decompression"
	"io"
	"os"
	//"runtime/pprof"
	"time"
)

type nullWriter struct {
	N uint64
}

func (nw *nullWriter) Write(data []byte) (int, error) {
	nw.N += uint64(len(data))
	return len(data), nil
}

func DoDecoding(r io.Reader) int64 {
	var err error
	outfile := &nullWriter{}
	//outfile, err := os.OpenFile("/dev/null", os.O_WRONLY, 777)
	//outfile, err := os.Create("./../testingdata/ubuntu.iso")
	//outfile, err := os.Create("./Hexagon.jpg")
	if err != nil {
		panic(err.Error())
	}

	dec := decompression.NewFrameDecompressor(r, outfile)
	//dec.Verbose = true

	err = dec.Decompress()
	if err != nil {
		panic(err.Error())
	}
	return int64(outfile.N)
}

var diffs []string

func CompareWithFile(original string, compressed *os.File) int64 {
	origfile, err := os.Open(original)
	if err != nil {
		panic(err.Error())
	}
	origReader := bufio.NewReader(origfile)

	comp, err := decompression.NewFrameReader(compressed)
	if err != nil {
		panic(err.Error())
	}
	comp.PrintStatus = true
	compReader := bufio.NewReader(comp)

	differences := false

	lastIndexDiff := false
	i := 0
	for i = 0; true; i++ {
		b1, err1 := origReader.ReadByte()
		b2, err2 := compReader.ReadByte()

		if err1 == err2 && err1 == io.EOF {
			println("Ended at same byte")
			println("##################")
			if differences {
				println("###  Diffs    ####")
				diffs = append(diffs, original)
			} else {
				println("###  Success  ####")
			}
			println("##################")
			break
		}
		if err1 != nil {
			diffs = append(diffs, original+" Err: "+err1.Error())
			break
		}
		if err2 != nil {
			diffs = append(diffs, original+" Err: "+err2.Error())
			break
		}

		if b1 != b2 {
			if !lastIndexDiff {
				println("")
			}
			fmt.Printf("%d\t%d\t%d\n", i, b1, b2)
			differences = true
			lastIndexDiff = true
		} else {
			lastIndexDiff = false
		}
	}

	return int64(i)
}

func main() {
	//cpuprofiling, err := os.Create("./cpu.prof")
	//if err != nil {
	//	panic(err.Error())
	//}
	//pprof.StartCPUProfile(cpuprofiling)
	//defer pprof.StopCPUProfile()
	//
	//memprofiling, err := os.Create("./mem.prof")
	//if err != nil {
	//	panic(err.Error())
	//}

	seconds := float64(0)
	bytes := int64(0)
	for i := 1; i < len(os.Args); i++ {
		println(os.Args[i])
		file, err := os.Open(os.Args[i])
		if err != nil {
			println("Error:")
			println(err.Error())
			println("Skipping")
			continue
		}
		//compare with same file but with ".zst" cut off
		original := (os.Args[i])[:len(os.Args[i])-4]
		print("Comparing with: ")
		println(original)
		startT := time.Now()
		bytes += CompareWithFile(original, file)
		//bytes += DoDecoding(file)
		timeUsed := time.Now().Sub(startT)

		seconds += timeUsed.Seconds()
	}

	if len(diffs) > 0 {
		println("Found diffs in files: ")
		for _, f := range diffs {
			println(f)
		}
		println("###########")
	} else {
		println("")
		println("Found no diffs in any files! Good job you!")
	}

	bytesPerSecond := float64(bytes) / seconds
	print("\n\n Average detected Speed: ")

	if bytesPerSecond > 1000000 {
		print(int(bytesPerSecond / 1000000))
		println(" MB/s")
	} else {
		if bytesPerSecond > 1000 {
			print(int(bytesPerSecond / 1000))
			println(" KB/s")
		} else {
			print(int(bytesPerSecond))
			println(" B/s")
		}
	}

	//pprof.WriteHeapProfile(memprofiling)
	return
}
