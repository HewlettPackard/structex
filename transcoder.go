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
)

type tagReference struct {
	value reflect.Value // Value of field tagged with `sizeOf` or `countOf`.
	tags  *tags         // The tag attributes of the field tagged with `sizeOf` or `countOf`.
}

type stack struct {
	len  int
	vals []reflect.Value
}

type handler interface {
	field(val reflect.Value, tags *tags) error
	layout(val reflect.Value, ref *tagReference) error
	array(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error
	slice(t *transcoder, arr reflect.Value, tags *tags, ref *tagReference) error
}

type transcoder struct {
	handler   handler
	fieldMap  map[string]*tagReference
	backtrace stack
}

func newTranscoder(h handler) *transcoder {

	t := transcoder{
		handler:   h,
		fieldMap:  make(map[string]*tagReference),
		backtrace: stack{len: 0},
	}

	return &t
}

func (t *transcoder) transcode(val reflect.Value) error {

	// Allow the user the pass in a struct or a struct pointer.
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Top level calls should always be of struct type, but
	// we all recursive calls of transcode so must handle
	// raw types.
	if val.Kind() != reflect.Struct {
		return t.handler.field(val, nil)
	}

	t.backtrace.push(val)
	defer t.backtrace.pop()

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		fieldVal := val.Field(i)
		fieldTyp := typ.Field(i)

		tags := parseFieldTags(fieldTyp)

		switch fieldTyp.Type.Kind() {

		case reflect.Struct:
			// Nested structure, do recursive transcoding
			if err := t.transcode(fieldVal); err != nil {
				return err
			}

		case reflect.Array:
			if err := t.handler.array(t, fieldVal, &tags, t.fieldMap[fieldTyp.Name]); err != nil {
				return err
			}

		case reflect.Slice:
			if err := t.handler.slice(t, fieldVal, &tags, t.fieldMap[fieldTyp.Name]); err != nil {
				return err
			}

		default:
			
			if tags.layout.format != none {

				found := t.fieldByName(tags.layout.name)

				if !found.IsValid() || found.Type() == reflect.PtrTo(reflect.TypeOf(reflect.Invalid)) {
					return fmt.Errorf("Cannot locate field name '%s'", tags.layout.name)
				}

				if found.Kind() != reflect.Slice && found.Kind() != reflect.Array {
					return fmt.Errorf("Referenced layout must be of type slice or array; Is of type %s", found.Kind().String())
				}

				ref := &tagReference{
					value: fieldVal,
					tags:  &tags,
				}

				if err := t.handler.layout(found, ref); err != nil {
					return err
				}

				t.fieldMap[tags.layout.name] = ref

			} else {

				if err := t.handler.field(fieldVal, &tags); err != nil {
					return err
				}
			}

		}
	}

	return nil
}

func (s *stack) push(v reflect.Value) {
	s.vals = append(s.vals, v)
	s.len = len(s.vals)
}

func (s *stack) pop() {
	s.len--
}

func (t *transcoder) fieldByName(name string) reflect.Value {
	for i := t.backtrace.len; i != 0; i-- {
		found := t.backtrace.vals[i-1].FieldByName(name)
		if found.IsValid() {
			return found
		}
	}

	// I don't know why this doesn't return a value.Invalid()
	// Maybe something to look into further.
	return reflect.New(reflect.TypeOf(reflect.Invalid))
}

func getValue(val reflect.Value) uint64 {
	var value uint64 = 0

	switch val.Type().Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value = val.Uint()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value = uint64(val.Int())
	default:
		panic(fmt.Errorf("Field type %s unsupported", val.Type().Kind().String()))
	}

	return value
}
