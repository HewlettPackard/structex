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
	"reflect"
	"strconv"
	"strings"
)

type endian int

const (
	little endian = 0
	big    endian = 1
)

type bitfield struct {
	nbits    uint64
	reserved bool
}

const (
	none = iota
	sizeOf
	countOf
)

type layout struct {
	format   int
	name     string
	relative bool
	value    uint64
}

type alignment uint64

type tags struct {
	endian    endian
	bitfield  bitfield
	layout    layout
	alignment alignment
	truncate  bool
}

// A TaggingError occurs when the pack/unpack routines have
// detected the structure field annotation does not match
// what is expected in the data stream.
type TaggingError struct {
	tag  string
	kind reflect.Kind
}

func (e *TaggingError) Error() string {
	return fmt.Sprintf("Invalid tag '%s' for %s", e.tag, e.kind.String())
}

/*
Method which parses the field tags and returns a series of informative
structures defined by the the structure extension values.
*/
func parseFieldTags(sf reflect.StructField) tags {
	t := tags{
		endian:    little,
		bitfield:  bitfield{0, false},
		layout:    layout{none, "", false, 0},
		alignment: 0,
		truncate:  false,
	}

	// Always encode the size of the field, regardless of tags
	switch sf.Type.Kind() {
	case reflect.Array, reflect.Slice, reflect.Struct, reflect.Ptr:
		break
	case reflect.Bool:
		t.bitfield.nbits = 1
	default:
		t.bitfield.nbits = uint64(sf.Type.Bits())
	}

	if s, ok := sf.Tag.Lookup("structex"); ok {
		t.parseString(sf, s, parseOptions{sep: ',', quote: '\'', assign: '='})
	} else {
		t.parseString(sf, string(sf.Tag), parseOptions{sep: ' ', quote: '"', assign: ':'})
	}

	return t
}

type parseOptions struct {
	sep    rune
	quote  rune
	assign rune
}

// Full tag format i.e. `structex:"bitfield='4,reserved',sizeof='Array'"`
// Bare tag format i.e. `bitfield:"3,reserved" sizeOf:"Array"`
func (t *tags) parseString(sf reflect.StructField, tagString string, opts parseOptions) {
	if len(tagString) == 0 {
		return
	}

	key := []rune{}
	val := []rune{}

	inKey := true
	inVal := false

	addKey := func(r rune) {
		key = append(key, r)
	}

	addVal := func(r rune) {
		val = append(val, r)
	}

	runes := []rune(tagString)
	for idx, r := range runes {

		switch r {
		case opts.assign:
			if inKey {
				inKey = false
			}
		case opts.quote:
			if inVal {
				inVal = false
				goto ADDTAG
			} else {
				inVal = true
			}
		case opts.sep, ',':
			if !inVal {
				inKey = true
			} else {
				addVal(r)
			}
		default:
			if inKey {
				addKey(r)
				if idx == len(runes)-1 {
					goto ADDTAG
				}
			} else if inVal {
				addVal(r)
			}
		}

		continue

	ADDTAG:
		t.add(sf, string(key), string(val))
		key = []rune{}
		val = []rune{}

	}
}

func (t *tags) add(sf reflect.StructField, key string, val string) {
	switch strings.ToLower(key) {
	case "little":
		t.endian = little

	case "big":
		t.endian = big

	case "bitfield":
		if nbs := strings.Split(val, ",")[0]; len(nbs) != 0 {
			nbits, err := strconv.ParseInt(nbs, 0, int(sf.Type.Bits()))
			if err != nil {
				panic(&TaggingError{string(sf.Tag), sf.Type.Kind()})
			}
			t.bitfield.nbits = uint64(nbits)
		}

		t.bitfield.reserved = strings.Contains(val, "reserved")

	case "sizeof":
		t.layout.format = sizeOf
		t.layout.name = strings.Split(val, ",")[0]
		t.layout.relative = strings.Contains(val, "relative")

	case "countof":
		t.layout.format = countOf
		t.layout.name = val

	case "truncate":
		t.truncate = true

	case "align":
		align, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			panic(&TaggingError{string(sf.Tag), sf.Type.Kind()})
		}
		t.alignment = alignment(align)
	}
}

func (t *tags) parseBitfield(sf reflect.StructField, s string, opts parseOptions) {

}

func (t *tags) print() {
	fmt.Printf("Bitfield: Bits: %d Reserved: %t\n", t.bitfield.nbits, t.bitfield.reserved)
	fmt.Printf("Layout: Type: %d Field: %s Relative %t\n", t.layout.format, t.layout.name, t.layout.relative)
	fmt.Printf("Alignment: %d\n", t.alignment)
	fmt.Println("")
}
