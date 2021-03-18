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
	"io"
	"math"
	"reflect"
)

type encoder struct {
	writer      io.ByteWriter
	currentByte uint8
	byteOffset  uint64
	bitOffset   uint64
}

func (e *encoder) write(value uint64, nbits uint64) error {

	if nbits > 1 && value > uint64(math.Pow(2, float64(nbits)))-1 {
		return fmt.Errorf("Value %d will overflow bitfield of %d bits", value, nbits)
	}

	if nbits < 8 {
		shift := e.bitOffset
		if shift+nbits > 8 {
			return fmt.Errorf("Bitfield not allowed to overflow single byte")
		}

		e.currentByte |= uint8(value << shift)
		e.bitOffset += nbits

		if e.bitOffset == 8 {
			e.writeByte(e.currentByte)
			e.bitOffset = 0
			e.currentByte = 0
		}

		return nil
	}

	if e.bitOffset != 0 {
		return fmt.Errorf("Cannot span multi-byte bitfield")
	}

	if nbits%8 != 0 {
		return fmt.Errorf("Can only write 8-bit byte multiples")
	}

	for b := uint64(0); b < nbits/8; b++ {
		if err := e.writeByte(uint8(value >> (b * 8))); err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) writeByte(value uint8) error {
	if err := e.writer.WriteByte(value); err != nil {
		return err
	}

	e.byteOffset++
	return nil
}

func (e *encoder) field(val reflect.Value, tags *tags) error {
	v := getValue(val)
	if tags == nil {
		return e.write(v, uint64(val.Type().Bits()))
	}

	return e.write(v, tags.bitfield.nbits)
}

func (e *encoder) layout(val reflect.Value, ref *tagReference) error {
	value := uint64(0)

	if ref.value.IsZero() {
		switch ref.tags.layout.format {
		case sizeOf:
			sz, err := size(val.Index(0))
			if err != nil {
				return err
			}

			value = uint64(val.Len()) * sz
			if ref.tags.layout.relative {
				value -= e.byteOffset
			}

		case countOf:
			value = uint64(val.Len())
		}
	} else {
		value = getValue(ref.value)
	}

	return e.write(value, ref.tags.bitfield.nbits)
}

func (e *encoder) array(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error {
	l := arr.Len()
	if ref != nil && !ref.value.IsZero() {
		l = int(ref.value.Uint())
	}

	for i := 0; i < l; i++ {
		if err := t.transcode(arr.Index(i)); err != nil {
			return err
		}
	}

	return nil
}

func (e *encoder) slice(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error {
	return e.array(t, arr, tags, ref)
}

/*
Encode serializes the data structure defined by 's' into the available
io.ByteWriter stream. Annotation rules are as defined in the Decode
function.
*/
func Encode(writer io.ByteWriter, s interface{}) error {

	e := encoder{
		writer:      writer,
		currentByte: 0,
		byteOffset:  0,
		bitOffset:   0,
	}

	t := newTranscoder(&e)

	return t.transcode(reflect.ValueOf(s))
}

/*
EncodeByteBuffer serializes the provided data structure 's' into a new byte
buffer. Bytes are packed according to the annotation rules defined for 's'.
*/
func EncodeByteBuffer(s interface{}) ([]byte, error) {
	buf := NewBuffer(s)
	if buf == nil {
		return nil, fmt.Errorf("Could not allocate byte buffer")
	}

	if err := Encode(buf, s); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
