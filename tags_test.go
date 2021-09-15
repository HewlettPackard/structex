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
	"reflect"
	"testing"
)

func testTags(t *testing.T, s interface{}, i int, test func(t tags) bool) {
	val := reflect.ValueOf(s)
	typ := val.Type()
	tags := parseFieldTags(typ.Field(i))
	if !test(tags) {
		t.Errorf("Test on field %d '%s' failed: tags %+v", i, typ.Field(i).Name, tags)
	}
}

func TestBareTags(t *testing.T) {
	s := struct {
		A int `bitfield:"3"`
		B int `bitfield:"3,reserved"`
		C int `countOf:"D"`
		D int `sizeOf:"E"`
		E int `sizeOf:"F,relative"`
		G int `align:"8"`
		H int `truncate:""`
		I int `big:""`
		J int `little:""`
	}{
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
	}

	testTags(t, s, 0, func(t tags) bool { return t.bitfield.nbits == 3 })
	testTags(t, s, 1, func(t tags) bool { return t.bitfield.nbits == 3 && t.bitfield.reserved })
	testTags(t, s, 2, func(t tags) bool { return t.layout.name == "D" && t.layout.format == countOf })
	testTags(t, s, 3, func(t tags) bool { return t.layout.name == "E" && t.layout.format == sizeOf })
	testTags(t, s, 4, func(t tags) bool { return t.layout.name == "F" && t.layout.format == sizeOf && t.layout.relative })
	testTags(t, s, 5, func(t tags) bool { return t.alignment == 8 })
	testTags(t, s, 6, func(t tags) bool { return t.truncate == true })
	testTags(t, s, 7, func(t tags) bool { return t.endian == big })
	testTags(t, s, 8, func(t tags) bool { return t.endian == little })
}

func TestFullTags(t *testing.T) {
	s := struct {
		A int `structex:"bitfield='3'"`
		B int `structex:"bitfield='3,reserved'"`
		C int `structex:"countOf='D'"`
		D int `structex:"sizeOf='E'"`
		E int `structex:"sizeOf='F,relative'"`
		G int `structex:"align='8'"`
		H int `structex:"truncate"`
		I int `structex:"big"`
		J int `structex:"little"`
	}{
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
		0,
	}

	testTags(t, s, 0, func(t tags) bool { return t.bitfield.nbits == 3 })
	testTags(t, s, 1, func(t tags) bool { return t.bitfield.nbits == 3 && t.bitfield.reserved })
	testTags(t, s, 2, func(t tags) bool { return t.layout.name == "D" && t.layout.format == countOf })
	testTags(t, s, 3, func(t tags) bool { return t.layout.name == "E" && t.layout.format == sizeOf })
	testTags(t, s, 4, func(t tags) bool { return t.layout.name == "F" && t.layout.format == sizeOf && t.layout.relative })
	testTags(t, s, 5, func(t tags) bool { return t.alignment == 8 })
	testTags(t, s, 6, func(t tags) bool { return t.truncate == true })
	testTags(t, s, 7, func(t tags) bool { return t.endian == big })
	testTags(t, s, 8, func(t tags) bool { return t.endian == little })
}
