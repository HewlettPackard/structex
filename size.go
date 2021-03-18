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
	"errors"
	"fmt"
	"reflect"
)

var (
	CannotDeductSliceLengthError = errors.New("Cannot duduct slice length")
)

type sizer struct {
	size   uint64
	nbits  uint64
	nbytes uint64
}

func (s *sizer) addBits(nbits uint64) error {
	s.nbits += nbits

	s.nbytes += s.nbits / 8
	s.nbits = s.nbits % 8

	return nil
}

func (s *sizer) field(val reflect.Value, tags *tags) error {
	if tags == nil {
		return s.addBits(uint64(val.Type().Bits()))
	}
	return s.addBits(tags.bitfield.nbits)
}

func (s *sizer) layout(val reflect.Value, ref *tagReference) error {
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
				value -= s.nbytes
			}
		case countOf:
			value = uint64(val.Len())
		}
	} else {
		value = getValue(ref.value)
	}

	ref.tags.layout.value = value

	return s.addBits(ref.tags.bitfield.nbits)
}

func (s *sizer) array(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error {
	sz, err := size(arr.Index(0))
	if err != nil {
		return err
	}

	len := uint64(arr.Len())
	if ref != nil && !ref.value.IsZero() {
		len = ref.value.Uint()
	}

	return s.addBits(sz * len * 8)
}

func (s *sizer) slice(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error {
	return s.array(t, arr, tags, ref)
}

/*
Size returns the size of the structure after considering all annotation rules.
Annotation rules are defined in the Decode function.

Size cannot determine the length of Slice types unless a layout annotation
exists (`sizeOf` or `countOf`) that contains a non-zero value.
*/
func Size(s interface{}) (uint64, error) {
	return size(reflect.ValueOf(s))
}

func size(value reflect.Value) (uint64, error) {

	s := sizer{
		size: 0,
	}

	t := newTranscoder(&s)

	if err := t.transcode(value); err != nil {
		return 0, err
	}

	if s.nbits != 0 {
		return 0, fmt.Errorf("Left-over bits in structure definition")
	}

	return s.nbytes, nil
}

// typeSize returns the size of the type t and all nested types.
// Unlike getValueSize, getTypeSize cannot return the size of slices
// as it is only aware of the types (and not values)
func typeSize(t reflect.Type) (uint64, error) {

	switch t.Kind() {
	case reflect.Struct:
		return structTypeSize(t)
	case reflect.Array:
		return typeSize(t.Elem())
	case reflect.Slice:
		return 0, CannotDeductSliceLengthError
	default:
		return uint64(t.Size()), nil
	}
}

func structTypeSize(t reflect.Type) (uint64, error) {
	var bytes uint64 = 0
	var bits uint64 = 0

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		switch f.Type.Kind() {
		case reflect.Array:
			sz, err := typeSize(f.Type)
			if err != nil {
				return 0, err
			}
			bits += sz * 8 * uint64(f.Type.Len())
		case reflect.Slice:
			return 0, CannotDeductSliceLengthError
		default:
			tags := parseFieldTags(f)

			bits += uint64(tags.bitfield.nbits)
		}

		for ; bits >= 8; bits -= 8 {
			bytes++
		}
	}

	if bits != 0 {
		return bytes, CannotDeductSliceLengthError
	}

	return bytes, nil
}
