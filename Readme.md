# Sparkzstd
This is a decompressor for the Zstandard compression format [Original Documentation](https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md)

It is working, tested on a lot (1000) of decodecorpus files (generated with the tool from the original zstd authors: https://github.com/facebook/zstd/tree/dev/tests). A few samples are in this repo for anyone who might be wanting to work on this and might  need something to do regression tests.


## What are the goals of this project
Well mainly I had some time on my hands and wanted to write something that might be useful to someone out there.
The goal was to provide a io.Reader compatible API for reading zstd encoded data from a provided io.Reader.

The original goal has been reached. Now I will maybe work on some optimizations. Some parts can be parallelized and some parts can 
probably be written better. (Some clean up on eg. exported types and functions might be nice)

Checksums should be supported, and maybe resetting the reader/decompressor so it can read a new frame without having to allocate a new one.

## How do I use this?

You can use this a library in your own project. In cmd/* are different programs that make use of this library.

### Library usage
There are two things this libary primarly provides to users. 

Firstly an io.Reader compatible "FrameReader". It is created by calling "NewFrameReader(r)" which accepts any io.Reader. This reader can be for example a file that contains a zstd-Frame, or a tcp connection that receives a zstd frame.

Secondly a FrameDecoder which acts a kind of pipe from a "source" io.Reader which writes the decoded zstd-frame into a "target" io.Writer.
This is used by the framereader which uses a bytes.Buffer as "target" from which it serves the Read() calls.

### cmd/* programs and building
Currently there is only cmd/sparkzstd which is used for testing (see below) decompression against original files. It can be built by 
doing 
```
cd cmd/sparkzstd
go build . 
```

I plan to implement an equivalent of `zstd -d` so you can use it as a drop in if you want.


## If you want to help test this
I'd love some others to test this library. The workflow I use is:
1. Since this is only a decompressor you still need the original zstd to compress your test file
2. Compress any file you want (for example tar (without compression) some directories you have and compress the result with zstd)
3. Use the main.go in cmd/sparkzstd to compare the output of the reader with the original file

The current main.go takes as arguments one or more paths pointing to .zst files. It expects the original file to have the same path but without the extension .zst. It will compare the decoding-output for each input file with the content of the original and give a summary of the findings.

The main.go for now just takes a list of pathes to .zst files and checks if the decompressed output matches byte for byte the content of the file with the same path but without .zst extension. 
eg: `go run cmd/sparkzstd/main.go ../testdata/pi.txt.zst`
checks the decompression of ../testdata/pi.txt.zst against ../testdata/pi.txt

If you'd like I would be glad to add your results to the list below.

## Where do I find stuff
1. Frame/Block/Literals/Sequences and their decoding is in /structure (Some HeaderDecoding is happening in the /decompression/framedecompressor.go)
2. Actual decompression aka. SequenceExecution is in /decompression/sequence_execution.go and /decompression/ringbuffer.go
3. FSE related stuff like predefined tables etc. are in /fse/predefined
4. Helpers for operations that need to read bits out of a bitstream or a reversed bitstream are located in /bitstream

## What is still missing
Generally all concepts of the Format have been implemented and are working (to a degree, some subtle bugs are still there) except dictionary support.
1. Dictionary parsing
2. Checksum calculation
1. Good benchmarks
2. Better doc
3. More bugs (I do have some unit tests and did some manual testing but you know...)

## Tracking of tests I did as of yet
I am testing this on a few files right know
### Working
1. A simple .jpg which I cant upload here for copyright reasons. This already decodes correctly but it is also just 36kb big
1. (FIXED. Does now decode correctly) A ubuntu 18.04.2-live-server-amd64.iso with md5sum: fcbcc756a1aa5314d52e882067c4ca6a. This decodes almost correctly. The result has the correct length but differs in about 400 bytes in different locations
1. Tested all files from the Canterbury corpus from here http://corpus.canterbury.ac.nz/descriptions/#cantrbry . They decompress correctly
1. Tested all (the one pi.txt) files from the Miscellaneous corpus from here http://corpus.canterbury.ac.nz/descriptions/#misc . They decompress correctly
1. A bigger file that klauspost (see https://github.com/klauspost/compress/tree/zstd-decoder/zstd) uses to test his implementation decodes correctly. It had an edge case that I didnt account for. So thanks to Klaus for unveiling that bug!
1. All of the files in decodecourpus_files decode correctly
1. (FIXED. Does now decode correctly) Another larger file (tar archive of some parts of my $HOME which I cant upload here) wont decompress. (Probably) At some point the decoder doesnt read the correct amount of bytes (which is unlikely because I check in many places for correctness of amounts read/decoded etc). It finds a block with the "reserved" block type 3. I tested just discarding the block but that just fails at the next block.

### Not working

## Other Libaries
1. Another pure go implementation (that got finished around the same time as mine): https://github.com/klauspost/compress/tree/master/zstd. Sadly I didnt find the project before I Started on this one.
2. A wuff implementation is WIP here (but wuff doesnt generate Go correctly yet) https://github.com/mvdan/zstd
3. A cgo binding to zsdt can be found here (which is needed if you want to compress stuff and not just decompress): https://github.com/DataDog/zstd 

