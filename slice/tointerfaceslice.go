package slice

import (
	"fmt"
	"reflect"
)

// Sometimes we have a somewhat generic function that expects to receive
// a slice of _things_, and it doesn't care what those things are. Until
// golang implements generics, that means its type is []interface{}.
// parmapInterfacesToInterfaces in the parmap package is good example of
// this.
//
// And you might expect that, if you have a golang type like:
//
// mySlice := []string{}
//
// ...that you would be able to cast it into the form we need like
// mySlice.([]interface{}).
//
// But you would be wrong. That will fail. We do, however, have this function.
// It uses the reflect package, which is kind of slow, so try to use it
// sparingly. But it does solve this problem for us.
func ToInterfaceSlice(slice interface{}) ([]interface{}, error) {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		return []interface{}{}, fmt.Errorf("given a non-slice type")
	}

	// Keep the distinction between nil and empty slice input
	if s.IsNil() {
		return []interface{}{}, fmt.Errorf("nil slice given")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret, nil
}
