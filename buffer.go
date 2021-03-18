/*
Copyright 2021 Hewlett Packard Enterprise Development LP

Permission is hereby granted, free of charge, to any person obtaining a 
copy of this software and associated documentation files (the "Software"), 
to deal in the Software without restriction, including without limitation 
the rights to use, copy, modify, merge, publish, distribute, sublicense, 
and/or sell copies of the Software, and to permit persons to whom the 
Software is furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in 
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR 
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, 
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.

IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, 
DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR 
OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE 
USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package structex

import (
	"bytes"
	"fmt"
)

/*
Buffer forms the basis of the storage area used for Pack and Unpack operations.
*/
type Buffer struct {
	bytes  []byte
	offset int
}

/*
Bytes returns the raw bytes forming the basis of the Buffer
*/
func (buf *Buffer) Bytes() []byte {
	return buf.bytes
}

/*
WriteByte implements the io.Writer requirement for the Buffer
*/
func (buf *Buffer) WriteByte(b byte) error {
	if buf.offset >= len(buf.bytes) {
		return fmt.Errorf("Write buffer overrun")
	}

	buf.bytes[buf.offset] = b
	buf.offset++

	return nil
}

/*
ReadByte implements the io.Reader requirement of the Buffer
*/
func (buf *Buffer) ReadByte() (byte, error) {
	if buf.offset >= len(buf.bytes) {
		return 0, fmt.Errorf("Read buffer overrun")
	}

	b := buf.bytes[buf.offset]
	buf.offset++

	return b, nil
}

/*
NewBuffer returns a new Buffer for making Pack and Unpack operations
ahead of the file Write and after file Read operations, respectfully.

The general pattern is to declare an annotated structure and fill in
the desired fields. For example

    type Test struct {
	    Param1 uint32 `bitfield:"5"`
	    Param2 uint32 `bitfield:"3"`
    }

would declare variable test with parameters

	var test Test
	test.Param1 = 3
	test.Param2 = 1

A buffer is allocated of appropriate size using the NewBuffer function

	b := structex.NewBuffer(test)

for which the structure is packed into

	structex.Pack(b, test)

The Buffer can then be used for a File Write operations

	f, err := os.Open("path/to/file")
	nbytes, err := f.Write(b.Bytes())

or used for File Read operations

	nbytes, err := f.Read(b.Bytes())

for which the Buffer can feed into Unpack to decode the available bytes

	var unpacked = new (Test)
	err := structex.Unpack(b, unpacked)
*/
func NewBuffer(s interface{}) *Buffer {
	var buf Buffer

	if s != nil {
		size, err := Size(s)
		if err != nil {
			return nil
		}
		buf.bytes = make([]byte, size)
	}

	buf.offset = 0

	return &buf
}

/*
Reset will clear the buffer for reuse
*/
func (buf *Buffer) Reset() {
	for i := range buf.bytes {
		buf.bytes[i] = 0
	}
	buf.offset = 0
}

/*
DebugDump will print the Buffer in raw byte format, 16 bytes
per line. Useful for debugging structure packing and unpacking.
*/
func (buf *Buffer) DebugDump() {
	var offset = 0
	for offset = 0; offset < buf.offset; offset += 16 {
		fmt.Printf("%08x: ", offset)
		for i := 0; i < 16; i++ {
			if offset+i < buf.offset {
				fmt.Printf("%02x ", buf.bytes[i+offset])
			} else {
				fmt.Printf("-- ")
			}
		}
		fmt.Printf("\n")
	}
}

type byteBufferReader struct {
	buffer *bytes.Buffer
}

/*
ReadByte implements the io.Reader requirement of the Buffer
*/
func (b *byteBufferReader) ReadByte() (byte, error) {
	return b.buffer.ReadByte()
}
