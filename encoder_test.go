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
	"fmt"
	"testing"
)

const (
	BufferSize = 512
)

type testWriter struct {
	bytes  [BufferSize]byte
	nbytes int
}

func (tw *testWriter) WriteByte(b byte) error {
	if tw.nbytes >= BufferSize {
		return fmt.Errorf("Byte buffer overflow")
	}

	tw.bytes[tw.nbytes] = b
	tw.nbytes++

	return nil
}

func (tw *testWriter) getByte(i int) byte {
	return tw.bytes[i]
}

func (tw *testWriter) getSize() int {
	return tw.nbytes
}

func packAndTest(t *testing.T, s interface{}, testFunc func(t *testing.T, tw *testWriter)) {
	var tw = &testWriter{}
	if err := Encode(tw, s); err != nil {
		t.Error(err)
	}

	testFunc(t, tw)
}

func TestSimpleEncoder(t *testing.T) {
	s := struct {
		A uint16
		B uint16
	}{
		0x0001, 0xFFEE,
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getByte(0) != 0x01 {
			t.Errorf("Simple pack failure byte 0: Expected: %x Actual: %x", 0x01, tw.getByte(0))
		}
		if tw.getByte(1) != 0x00 {
			t.Errorf("Simple pack failure byte 1: Expected: %x Actual: %x", 0x00, tw.getByte(1))
		}
		if tw.getByte(2) != 0xEE {
			t.Errorf("Simple pack failure byte 2: Expected: %x Actual: %x", 0xEE, tw.getByte(2))
		}
		if tw.getByte(3) != 0xFF {
			t.Errorf("Simple pack failure byte 3: Expected: %x Actual: %x", 0xFF, tw.getByte(3))
		}
	})
}

func TestBitfieldEncoder(t *testing.T) {

	s := struct {
		A int `bitfield:"3"`
		B int `bitfield:"4"`
		C int `bitfield:"1"`
		D int `bitfield:"12"`
		E int `bitfield:"4"`
	}{
		0x7, 0x8, 0x1, 0x0FFF, 0x1,
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getByte(0) != 0xC7 {
			t.Errorf("Invalid bitfield: Expected: %#02x Actual: %#02x", 0xC7, tw.getByte(0))
		}
		if tw.getByte(1) != 0xFF {
			t.Errorf("Invalid bitfield: Expected: %#02x Actual: %#02x", 0xFF, tw.getByte(1))
		}
		if tw.getByte(2) != 0x1F {
			t.Errorf("Invalid bitfield: Expected: %#02x Actual: %#02x", 0x1F, tw.getByte(2))
		}
	})
}

func TestNestingEncoder(t *testing.T) {
	type Nested struct {
		M uint8
		N uint8
	}

	type S struct {
		A uint8
		B uint8
		C Nested
	}

	s := S{
		A: 0x00,
		B: 0x01,
		C: Nested{
			M: 0x02,
			N: 0x03,
		},
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		for i := 0; i < 4; i++ {
			if tw.getByte(i) != uint8(i) {
				t.Errorf("Invalid byte at offset %d: Expected: %#02x Actual: %#02x", i, i, tw.getByte(i))
			}
		}
	})

}

func TestByteArrayEncoder(t *testing.T) {
	s := struct {
		Size uint8
		Data []byte
	}{
		Size: 4,
		Data: []byte{0x00, 0x01, 0x02, 0x03},
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getByte(0) != 4 {
			t.Errorf("Size Encoding Incorrect: Expected: %d Actual: %d", 4, tw.getByte(0))
		}
		for i := 0; i < 4; i++ {
			if tw.getByte(i+1) != byte(i) {
				t.Errorf("Byte Index %d Incorrect: Expected: %#02x Actual: %#02x", i, i, tw.getByte(i+1))
			}
		}
	})
}
func TestArrayEncoder(t *testing.T) {
	type T struct {
		A uint8
		B uint8
	}

	s := struct {
		Count uint8 `countOf:"Ts"`
		Size  uint8 `sizeOf:"Ts"`
		Ts    [2]T
	}{
		Count: 0x00,
		Size:  0x00,
		Ts: [2]T{
			{A: 0x01, B: 0x02},
			{A: 0x03, B: 0x04},
		},
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getByte(0) != 2 {
			t.Errorf("Invalid countOf: Expected: %d Actual: %d", 2, tw.getByte(0))
		}
		if tw.getByte(1) != 4 {
			t.Errorf("Invalid sizeOf: Expected: %d Actual: %d", 4, tw.getByte(1))
		}

		expected := []uint8{0x01, 0x02, 0x03, 0x04}
		actual := []uint8{tw.getByte(2), tw.getByte(3), tw.getByte(4), tw.getByte(5)}

		for i := 0; i < 4; i++ {
			if expected[i] != actual[i] {
				t.Errorf("Invalid array pack: Index: %d Expected: %#02x Actual: %#02x", i, expected[i], actual[i])
			}
		}
	})
}

func TestSliceEncoder(t *testing.T) {

	ts := [6]uint8{0xA, 0xB, 0xC, 0xD, 0xE, 0xF}

	s := struct {
		Count uint8 `countOf:"Ts"`
		Size  uint8 `sizeOf:"Ts"`
		Ts    []uint8
	}{
		Count: 0,
		Size:  0,
		Ts:    ts[2:4],
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getByte(0) != 2 {
			t.Errorf("Invalid countOf: Expected: %d Actual: %d", 2, tw.getByte(0))
		}

		if tw.getByte(1) != 2 {
			t.Errorf("Invalid sizeOf: Expected: %d Actual: %d", 2, tw.getByte(1))
		}

		// Check the slice contents
		expected := []uint8{0x0C, 0x0D}
		actual := []uint8{tw.getByte(2), tw.getByte(3)}

		for i := 0; i < len(expected); i++ {
			if expected[i] != actual[i] {
				t.Errorf("Invalid slice pack: Index: %d Expected: %#02x Actual: %#02x", i, expected[i], actual[i])
			}
		}
	})
}

func TestArrayTruncate(t *testing.T) {
	s := struct {
		Size  uint32 `sizeOf:"Array"`
		Array [BufferSize - 4]byte
	}{
		4,
		[BufferSize - 4]byte{0x00, 0x01, 0x02, 0x03, 0xFF /*Shouldn't be decoded*/},
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getSize() != 8 {
			t.Errorf("Invalid size of encoded buffer: Expected: %d Actual: %d", 8, tw.getSize())
		}
		if tw.getByte(0) != 4 {
			t.Errorf("Invalid sizeOf: Expected: %d Actual: %d", 4, tw.getByte(0))
		}
		for i := 0; i < 4; i++ {
			if tw.getByte(4+i) != byte(i) {
				t.Errorf("Invalid array byte: Expected: %#02x Actual: %#02x", i, tw.getByte(4+i))
			}
		}
	})
}

func TestAlignment(t *testing.T) {
	s := struct {
		Pad uint8
		Aligned uint32 `align:"4"`
	} {
		0x00, 0xFF,
	}

	packAndTest(t, s, func(t *testing.T, tw *testWriter) {
		if tw.getSize() != 8 {
			t.Errorf("Invalid size of encoded buffer: Expected: %d Actual: %d", 8, tw.getSize())
		}

		if tw.getByte(4) != 0xFF {
			t.Errorf("Invalid aligned field: Expected %#02x Actual: %#02x", 0xFF, tw.getByte(4))
		}
	})
}
