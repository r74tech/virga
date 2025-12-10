package memdb

import (
	"fmt"
	"reflect"
	"time"
)

// TimeFieldIndex is used to index a time.Time field
type TimeFieldIndex struct {
	Field string
}

// FromObject extracts the time value from the object
func (t *TimeFieldIndex) FromObject(obj interface{}) (bool, []byte, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v) // Dereference pointer if needed

	// Check if it's a struct
	if v.Kind() != reflect.Struct {
		return false, nil, fmt.Errorf("TimeFieldIndex can only be used on structs")
	}

	// Get the field
	fieldVal := v.FieldByName(t.Field)
	if !fieldVal.IsValid() {
		return false, nil, fmt.Errorf("field %s not found", t.Field)
	}

	// Check if it's a time.Time or *time.Time
	var timeVal time.Time
	switch fieldVal.Interface().(type) {
	case time.Time:
		timeVal = fieldVal.Interface().(time.Time)
	case *time.Time:
		if fieldVal.IsNil() {
			return false, nil, nil
		}
		timeVal = *fieldVal.Interface().(*time.Time)
	default:
		return false, nil, fmt.Errorf("field %s is not a time.Time", t.Field)
	}

	// Convert to Unix nano for ordering
	val := timeVal.UnixNano()

	// Convert to bytes for indexing
	buf := make([]byte, 8)
	for i := 0; i < 8; i++ {
		buf[i] = byte(val >> (56 - i*8))
	}

	return true, buf, nil
}

// FromArgs is not implemented for TimeFieldIndex
func (t *TimeFieldIndex) FromArgs(args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("TimeFieldIndex expects 1 argument")
	}

	switch arg := args[0].(type) {
	case time.Time:
		val := arg.UnixNano()
		buf := make([]byte, 8)
		for i := 0; i < 8; i++ {
			buf[i] = byte(val >> (56 - i*8))
		}
		return buf, nil
	case *time.Time:
		if arg == nil {
			return nil, fmt.Errorf("nil time pointer")
		}
		val := arg.UnixNano()
		buf := make([]byte, 8)
		for i := 0; i < 8; i++ {
			buf[i] = byte(val >> (56 - i*8))
		}
		return buf, nil
	default:
		return nil, fmt.Errorf("argument must be time.Time, got %T", args[0])
	}
}
