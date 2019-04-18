package main

import (
	"bufio"
	"fmt"
	"github.com/killingspark/sparkzsdt/decompression"
	"io"
	"os"
	"runtime/pprof"
)

type nullWriter struct{}

func (nw *nullWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func DoDecoding(r io.Reader) {
	var err error
	outfile := &nullWriter{}
	//outfile, err := os.OpenFile("/dev/null", os.O_WRONLY, 777)
	//outfile, err := os.Create("./../testingdata/ubuntu.iso")
	//outfile, err := os.Create("./Hexagon.jpg")
	if err != nil {
		panic(err.Error())
	}

	dec := decompression.NewFrameDecompressor(r, outfile)
	dec.Verbose = true

	err = dec.Decompress()
	if err != nil {
		panic(err.Error())
	}
}

func CompareWithFile(original string, compressed *os.File) {
	origfile, err := os.Open(original)
	if err != nil {
		panic(err.Error())
	}
	origReader := bufio.NewReader(origfile)

	comp, err := decompression.NewFrameReader(compressed)
	if err != nil {
		panic(err.Error())
	}
	//comp.PrintStatus = true
	compReader := bufio.NewReader(comp)

	for i := 0; true; i++ {
		b1, err1 := origReader.ReadByte()
		b2, err2 := compReader.ReadByte()

		if err1 == err2 && err1 == io.EOF {
			println("Ended at same byte")
			println("##################")
			println("###  Success  ####")
			println("##################")
			return
		}
		if err1 != nil {
			panic(err1.Error())
		}
		if err2 != nil {
			panic(err2.Error())
		}

		if b1 != b2 {
			fmt.Printf("%d\t%d\t%d\n", i, b1, b2)
		}
	}
}

func main() {
	cpuprofiling, err := os.Create("./cpu.prof")
	if err != nil {
		panic(err.Error())
	}
	pprof.StartCPUProfile(cpuprofiling)
	defer pprof.StopCPUProfile()

	memprofiling, err := os.Create("./mem.prof")
	if err != nil {
		panic(err.Error())
	}

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
		CompareWithFile(original, file)
	}
	return

	file, err := os.Open("./../testingdata/bachelorarbeit.tar.zst")
	//file, err := os.Open("./../testingdata/ubuntu.zst")
	//file, err := os.Open("./Hexagon.jpg.zst")
	//content, err := ioutil.ReadFile("./ubuntu.zst")
	if err != nil {
		panic(err.Error())
	}

	//buf := bytes.NewBuffer(content)
	//DoDecoding(file)
	//CompareWithFile("./../testingdata/ubuntu-18.04.2-live-server-amd64.iso", file)

	//compares the output of the decompression byte for byte with the original file and produces output similarly
	//to the "cmp" tool. If you want to disable the progress printing, delete the line "comp.PrintStatus = true"
	CompareWithFile("./../testingdata/bachelorarbeit.tar", file)
	pprof.WriteHeapProfile(memprofiling)
}
