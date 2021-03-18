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

type alignment int64

type tags struct {
	bitfield  bitfield
	layout    layout
	alignment alignment
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
func parseFieldTags(sf reflect.StructField) (tag tags) {
	tag = tags{
		bitfield:  bitfield{0, false},
		layout:    layout{none, "", false, 0},
		alignment: -1,
	}

	if t := sf.Tag.Get("bitfield"); len(t) != 0 {
		if nbs := strings.Split(t, ",")[0]; len(nbs) != 0 {
			nbits, err := strconv.ParseInt(nbs, 0, int(sf.Type.Bits()))
			if err != nil {
				panic(&TaggingError{string(sf.Tag), sf.Type.Kind()})
			}
			tag.bitfield.nbits = uint64(nbits)
		}

		tag.bitfield.reserved = strings.Contains(t, "reserved")
	} else {
		tag.bitfield.nbits = uint64(sf.Type.Bits())
	}

	if t := sf.Tag.Get("sizeOf"); len(t) != 0 {
		tag.layout.format = sizeOf
		tag.layout.name = strings.Split(t, ",")[0]
		tag.layout.relative = strings.Contains(t, "relative")
	}

	if t := sf.Tag.Get("countOf"); len(t) != 0 {
		tag.layout.format = countOf
		tag.layout.name = t
	}

	if t := sf.Tag.Get("align"); len(t) != 0 {
		align, err := strconv.ParseInt(t, 0, 64)
		if err != nil {
			panic(&TaggingError{string(sf.Tag), sf.Type.Kind()})
		}
		tag.alignment = alignment(align)
	}

	return
}

func (t *tags) print() {
	fmt.Printf("Bitfield: Bits: %d Reserved: %t\n", t.bitfield.nbits, t.bitfield.reserved)
	fmt.Printf("Layout: Type: %d Field: %s Relative %t\n", t.layout.format, t.layout.name, t.layout.relative)
	fmt.Printf("Alignment: %d\n", t.alignment)
	fmt.Println("")
}
