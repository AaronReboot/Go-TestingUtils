// Package testingutils is a simple helper package for running table tests.
// If you're running Go 1.7 or higher, use new parameterized tests instead.
package testingutils

import (
	"fmt"
	"reflect"
	"testing"
)

//const EPSILON = .0000000000001
const EPSILON = .00001

type callHistoryElem struct {
	a, b reflect.Value
}

// IsEqual recursively unpacks objects to examine whether they are deep equal
// with EPSILON margin of error allowed for differences between float64 and
// float32 components
func IsEqual(a, b reflect.Value, t *testing.T) (bool, string) {
	return IsEqualLoopBreaker(a, b, make(map[callHistoryElem]struct{}), t)
}

// IsEqualLoopBreaker does the work of IsEqual, keeping track of all the
// recursive calls it's made so far, thus avoiding loops
func IsEqualLoopBreaker(a, b reflect.Value, callHistory map[callHistoryElem]struct{}, t *testing.T) (bool, string) {
	//if there's loop then we haven't found a problem so far
	if _, exists := callHistory[callHistoryElem{a, b}]; exists {
		return true, ""
	}

	//if a and b aren't the same thing => not equal
	if a.Kind() != b.Kind() {
		return false, "Not same types"
	}

	//if a and b are pointers indirect to their values
	if a.Kind() == reflect.Ptr && !a.IsNil() && !b.IsNil() {
		a = a.Elem()
		b = b.Elem()
	}

	//if a & b are float64 see whether they are within EPSILON of each other
	if a.Kind() == reflect.Float64 || a.Kind() == reflect.Float32 {
		//calculate difference between a and b
		diff := a.Float() - b.Float()
		//return whether diff is smaller than EPSILON (but not sure if diff is negative of positive)
		ok := (diff < EPSILON) && (-diff < EPSILON)
		var msg string
		if !ok {
			msg = fmt.Sprintf("Failing on a floating-point comparison: %f != %f\n", a.Float(), b.Float())
			t.Logf(msg)
		}
		return ok, msg
	}

	// add this call to the call history
	callHistory[callHistoryElem{a, b}] = struct{}{}

	// if a and b are slices or structs, check their elements
	if a.Kind() == reflect.Slice || a.Kind() == reflect.Array {
		// iterate over members, returning false right away if any member is false
		if a.Len() != b.Len() {
			return false, "Slices having different lengths"
		}
		for i := 0; i < a.Len(); i++ {
			pass, msg := IsEqualLoopBreaker(a.Index(i), b.Index(i), callHistory, t)
			if !pass {
				return false, msg
			}
		}
		return true, ""
	}

	// if a & b are structs, iterate over fields, returning false right away if any member is false
	if a.Kind() == reflect.Struct {
		if a.NumField() != b.NumField() {
			return false, "Number of fields in struct do not match"
		}
		// iterate over struct's fields
		for i := 0; i < a.NumField(); i++ {
			ok, msg := IsEqualLoopBreaker(a.Field(i), b.Field(i), callHistory, t)
			if !ok {
				return false, msg
			}
		}
		return true, ""
	}

	//TODO: if a & b are maps, iterate over keys & values in case we need to compare floats
	//Note, maps are already working well for some types because of fallthrough to reflect.DeepEqual

	//cover any unhandled case, like intentional ones (err, int, etc) and anything
	//we missed by accident (is ok because DeepEqual is conservative)
	ok := reflect.DeepEqual(a.Interface(), b.Interface())
	if !ok {
		return false, "reflect.DeepEqual failed"
	}
	return true, ""
}

func PrintTruncated(val interface{}) string {
	result := fmt.Sprint(val)
	if len(result) < 5000 {
		return result
	}
	return result[:5000] + "\n[...output truncated...]"
}

// RunTest runs test on func fnptr unsing invals as parameters and checking
// for expectvals as results returns true if test ok, returns false if test fails
func RunTest(fnptr, invals, expectvals interface{}, t *testing.T) (bool, string) {
	rvals := func(vals interface{}) (result []reflect.Value, names []string) {
		// figure out whether vals is a struct (if it is, we need to
		// read each field of struct into slice elements of result
		valOfVals := reflect.ValueOf(vals)
		if valOfVals.Kind() == reflect.Struct {
			result = make([]reflect.Value, valOfVals.NumField())
			names = make([]string, valOfVals.NumField())
			// load them in
			for i := 0; i < valOfVals.NumField(); i++ {
				result[i] = valOfVals.Field(i)
				names[i] = valOfVals.Type().Field(i).Name
			}
		} else {
			result = make([]reflect.Value, 1)
			result[0] = valOfVals
		}
		return
	}

	//obtain function/value reflect thing from fn
	fn := reflect.ValueOf(fnptr)

	// convert invals to slice of reflect.Values
	in, _ /*inNames*/ := rvals(invals)
	// convert expectvals to slice of reflect.Values
	expect, exNames := rvals(expectvals)

	if len(in) != fn.Type().NumIn() {
		t.Fatal("The number of in params doesn't match function parameters.")
		return false, ""
	}
	got := fn.Call(in)
	if len(got) != fn.Type().NumOut() {
		t.Fatal("The number of expect params doesn't match function results.")
		return false, ""
	}

	var (
		pass = true
		msg  = ""
	)
	for i := 0; i < len(got); i++ {
		pass, msg = IsEqual(got[i], expect[i], t)
		if !pass {
			//if this function returns more than one result, figure out the name of problem result
			var name string
			if len(got) > 1 {
				name = " (" + exNames[i] + ")"
			}
			t.Logf("Expected%s: %s\n\n", name, PrintTruncated(expect[i]))
			t.Logf("Got     %s: %s\n\n", name, PrintTruncated(got[i]))
			return false, msg
		}
	}
	return true, ""
}

// RunAllTests runs battery of all tests provided
func RunAllTests(fnptr, allInVals, allExpectVals interface{}, t *testing.T) {
	fracture := func(blob interface{}) (result []interface{}) {
		result = make([]interface{}, reflect.ValueOf(blob).Len())
		for i := 0; i < reflect.ValueOf(blob).Len(); i++ {
			result[i] = reflect.ValueOf(blob).Index(i).Interface()
		}
		return
	}

	// confirm that allInVals and allExpectVals are slices
	if reflect.ValueOf(allInVals).Kind() != reflect.Slice {
		t.Fatal("allInVals is not a slice")
		return
	}
	if reflect.ValueOf(allExpectVals).Kind() != reflect.Slice {
		t.Fatal("allExpectVals is not a slice")
		return
	}
	// convert allInVals & allExpectVals interface{} "blobs" into slice of interfaces
	allIn := fracture(allInVals)
	allExpect := fracture(allExpectVals)

	if len(allIn) != len(allExpect) {
		t.Fatal("Number of input tests doesn't match number of expected results")
		return
	}
	for i := 0; i < len(allIn); i++ {
		t.Logf("Testing case %v\n", i)
		pass, msg := RunTest(fnptr, allIn[i], allExpect[i], t)
		if !pass {
			t.Errorf("FAIL case %v (%s)\n", i, msg)
		}
	}
}
