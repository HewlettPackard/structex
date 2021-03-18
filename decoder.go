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
	"io"
	"math"
	"reflect"
)

type decoder struct {
	reader      io.ByteReader
	currentByte uint8
	byteOffset  uint64
	bitOffset   uint64
}

func (d *decoder) read(nbits uint64) (uint64, error) {

	if nbits == 0 {
		return 0, fmt.Errorf("Unsupported zero bit operation")
	}

	if nbits < 8 {
		if d.bitOffset == 0 {
			b, err := d.reader.ReadByte()
			if err != nil {
				return 0, err
			}
			d.currentByte = b
		}

		if nbits > uint64(8-d.bitOffset) {
			return 0, fmt.Errorf("Insufficient bit count for reading")
		}

		mask := uint(math.Pow(2, float64(nbits)) - 1)
		value := uint(d.currentByte>>d.bitOffset) & mask

		d.bitOffset += nbits
		if d.bitOffset >= 8 {
			d.bitOffset = 0
		}

		return uint64(value), nil
	}

	if nbits%8 != 0 {
		return 0, fmt.Errorf("Unsupported bit span of %d bits", nbits)
	}

	var value uint64 = 0
	for i := uint64(0); i < nbits; i += 8 {
		b, err := d.reader.ReadByte()
		if err != nil {
			return 0, err
		}

		value |= uint64(b) << i
	}

	return value, nil
}

func (d *decoder) readValue(value reflect.Value, tags *tags) (uint64, error) {

	if !value.CanSet() {
		return 0, fmt.Errorf("Field of type %s cannot be set. Make sure it is exported.",
			value.Type().Kind().String())
	}

	nbits := uint64(0)
	if value.Kind() == reflect.Bool {
		nbits = 1
	} else {
		nbits = uint64(value.Type().Bits())
	}

	if tags != nil {
		if nbits < tags.bitfield.nbits {
			return 0, fmt.Errorf("Field value of type %s has bitfield definition with %d bits, exceeding field size of %d bits.",
				value.Type().Kind().String(),
				tags.bitfield.nbits,
				nbits)
		}
		nbits = tags.bitfield.nbits
	}

	v, err := d.read(nbits)
	if err != nil {
		return 0, err
	}

	switch value.Kind() {
	case reflect.Bool:
		value.SetBool(v == 1)
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint:
		value.SetUint(v)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int:
		value.SetInt(int64(v))
	default:
		return 0, fmt.Errorf("Unsupported read type %s", value.Kind().String())
	}

	return v, nil
}

func (d *decoder) field(val reflect.Value, tags *tags) error {
	_, err := d.readValue(val, tags)
	return err
}

func (d *decoder) layout(val reflect.Value, ref *tagReference) error {
	value, err := d.readValue(ref.value, ref.tags)
	ref.tags.layout.value = value

	return err
}

func (d *decoder) array(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error {
	isStruct := arr.Type().Elem().Kind() == reflect.Struct
	for j := 0; j < arr.Len(); j++ {

		if isStruct { // Recurse down into the struct
			if err := t.transcode(arr.Index(j)); err != nil {
				return err
			}
		} else {
			if _, err := d.readValue(arr.Index(j), nil); err != nil {
				if err == io.EOF && tags != nil && tags.truncate {
					return nil
				}

				return err
			}
		}
	}

	return nil
}

func (d *decoder) slice(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error {
	length := uint64(arr.Len())

	if ref != nil {
		switch ref.tags.layout.format {
		case sizeOf:
			sz, err := typeSize(arr.Type().Elem())
			if err != nil {
				return err
			}

			if ref.tags.layout.value%sz != 0 {
				return fmt.Errorf("Slice with size %d of slice is a non-multiple of structure size %d",
					ref.tags.layout.value,
					sz)
			}

			length = ref.tags.layout.value / sz
		case countOf:
			length = ref.tags.layout.value
		default:
			return fmt.Errorf("Slice size cannot be determined. Did you miss a field tag?")
		}

		arr.Set(reflect.MakeSlice(arr.Type(), int(length), int(length)))
	}

	for j := 0; j < arr.Len(); j++ {
		if err := t.transcode(arr.Index(j)); err != nil {
			if err == io.EOF && tags != nil && tags.truncate {
				return nil
			}

			return err
		}
	}

	return nil
}

/*
Decode reads data from a ByteReader into provided annotated structure.

Deserialization occurs according to the annotations in the structure which
take several options:

Bitfields:
	Bitfields define a structure field with an explicit size in bits. They are
	analogous to bit fields in the C specification. Un

	`bitfield:[size][,reserved]`

	size       Specifies the size, in bits, of the field.

	reserved   Optional modifier that specifies the field contains reserved
	           bits and should be encoded as zeros.

Dynamic Layouts:
	Many industry standards support dynamically sized return fields where the
	data layout is self described by other fields. To support such formats
	two annotations are provided.

	`sizeOf:"[Field][,relative]"`

	Field		Specifies that the field describes the size of Field within the
				structure.

				During decoding, if field is non-zero, the field's value is
				used to limit the number elements in the array or slice of
				name Field.

	relative	Optional modifier that specifies the field value describing
				the size of Field is relative to the field offset within
				the structure. This is often used in T10.org documentation

	`countOf:"[Field]"`

	Field		Specifies that the field describes the count of elements in
				Field.

				During decoding, if field is non-zero, the field's value is
				used to limit the number elements in the array or slice of
				name Field.

Alignment:
	Annotations can specified the byte-alignment requirement for structure
	fields. Analogous to the alignas specifier in C. Can only be applied
	to non-bitfield structure fields.

	`align:"[value]"`

	value		An integer value specifying the byte alignment of the field.
				Invalid non-zero alignments panic.

*/
func Decode(reader io.ByteReader, s interface{}) error {

	d := decoder{
		reader:      reader,
		currentByte: 0,
		byteOffset:  0,
		bitOffset:   0,
	}

	t := newTranscoder(&d)

	return t.transcode(reflect.ValueOf(s))
}

// DecodeByteBuffer takes a raw byte buffer and unpacks the buffer into
// the provided structure. Unused bytes do not cause an error.
func DecodeByteBuffer(b *bytes.Buffer, s interface{}) error {
	reader := byteBufferReader{
		buffer: b,
	}

	return Decode(&reader, s)
}
