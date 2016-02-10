package testingutils

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

const EPSILON = .0000000000001

type callHistoryElem struct {
	a, b reflect.Value
}

/*	IsEqual recursively unpacks objects to examine whether they are deep equal with
	EPSILON margin of error allowed for differences between float64 components */
func IsEqual(a, b reflect.Value) bool {
	return IsEqual_LoopBreaker(a, b, make(map[callHistoryElem]struct{}))
}

/*	IsEqual_LoopBreaker does the work of IsEqual, keeping track of all the recursive
	calls it's made so far, thus avoiding loops */
func IsEqual_LoopBreaker(a, b reflect.Value, callHistory map[callHistoryElem]struct{}) bool {
	//if there's loop then we haven't found a problem so far
	if _, exists := callHistory[callHistoryElem{a, b}]; exists {
		return true
	}
	callHistory[callHistoryElem{a, b}] = struct{}{}

	//if a and b aren't the same thing => not equal
	if a.Kind() != b.Kind() {
		return false
	}

	//if a and b are pointers indirect to their values
	if a.Kind() == reflect.Ptr && !a.IsNil() && !b.IsNil() {
		a = a.Elem()
		b = b.Elem()
	}

	//if a and b are slices or structs, check their elements
	if a.Kind() == reflect.Slice {
		//iterate over members, returning false right away if any member is false
		if a.Len() != b.Len() {
			return false
		}
		for i := 0; i < a.Len(); i++ {
			if !IsEqual_LoopBreaker(a.Index(i), b.Index(i), callHistory) {
				return false
			}
		}
		return true
	}

	//if a & b are structs, iterate over fields, returning false right away if any member is false
	if a.Kind() == reflect.Struct {
		if a.NumField() != b.NumField() {
			return false
		}
		//iterate over struct's fields
		for i := 0; i < a.NumField(); i++ {
			if !IsEqual_LoopBreaker(a.Field(i), b.Field(i), callHistory) {
				return false
			}
		}
		return true
	}

	//TODO: if a & b are maps, iterate over keys & values in case we need to compare floats
	//Note, maps are already working well for some types because of fallthrough to reflect.DeepEqual

	//if a & b are float64 see whether they are within EPSILON of each other
	if a.Kind() == reflect.Float64 {
		//calculate difference between a and b
		diff := a.Float() - b.Float()
		//return whether diff is smaller than EPSILON (but not sure if diff is negative of positive)
		return (diff < EPSILON) && (-diff < EPSILON)
	}

	//cover any unhandled case, like intentional ones (err, int, etc) and anything
	//we missed by accident (is ok becaue DeepEqual is conservative)
	return reflect.DeepEqual(a.Interface(), b.Interface())
}

func PrintTruncated(val interface{}) string {
	result := fmt.Sprint(val)
	if len(result) < 500 {
		return result
	}
	return result[:500] + "\n[...output truncated...]"
}

/*	run test on func fnptr unsing invals as parameters and checking for expectvals as results
	returns true if test ok, returns false if test fails */
func RunTest(fnptr, invals, expectvals interface{}, t *testing.T) bool {
	rvals := func(vals interface{}) (result []reflect.Value, names []string) {
		//figure out whether vals is a struct (if it is, we need to
		//read each field of struct into slice elements of result
		valOfVals := reflect.ValueOf(vals)
		if valOfVals.Kind() == reflect.Struct {
			result = make([]reflect.Value, valOfVals.NumField())
			names = make([]string, valOfVals.NumField())
			//load them in
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

	//convert invals to slice of reflect.Values
	in, _ /*inNames*/ := rvals(invals)
	//convert expectvals to slice of reflect.Values
	expect, exNames := rvals(expectvals)

	if len(in) != fn.Type().NumIn() {
		log.Fatal("The number of in params doesn't match function parameters.")
	}
	got := fn.Call(in)
	if len(got) != fn.Type().NumOut() {
		log.Fatal("The number of expect params doesn't match function results.")
	}

	pass := true
	for i := 0; i < len(got); i++ {
		if !IsEqual(got[i], expect[i]) {
			//if this function returns more than one result, figure out the name of problem result
			var name string
			if len(got) > 1 {
				name = " (" + exNames[i] + ")"
			}
			t.Logf("Expected%s: %s\n\n", name, PrintTruncated(expect[i]))
			t.Logf("Got     %s: %s\n\n", name, PrintTruncated(got[i]))
			pass = false
		}
	}
	return pass
}

/* run battery of all tests provided */
func RunAllTests(fnptr, allInVals, allExpectVals interface{}, t *testing.T) {
	fracture := func(blob interface{}) (result []interface{}) {
		result = make([]interface{}, reflect.ValueOf(blob).Len())
		for i := 0; i < reflect.ValueOf(blob).Len(); i++ {
			result[i] = reflect.ValueOf(blob).Index(i).Interface()
		}
		return
	}

	//confirm that allInVals and allExpectVals are slices
	if reflect.ValueOf(allInVals).Kind() != reflect.Slice {
		log.Fatal("allInVals is not a slice")
	}
	if reflect.ValueOf(allExpectVals).Kind() != reflect.Slice {
		log.Fatal("allExpectVals is not a slice")
	}
	//convert allInVals & allExpectVals interface{} "blobs" into slice of interfaces
	allIn := fracture(allInVals)
	allExpect := fracture(allExpectVals)

	if len(allIn) != len(allExpect) {
		log.Fatal("Number of input tests doesn't match number of expected results")
	}
	for i := 0; i < len(allIn); i++ {
		t.Logf("Testing case %v\n", i)
		pass := RunTest(fnptr, allIn[i], allExpect[i], t)
		if !pass {
			t.Errorf("FAIL case %v\n", i)
		}
	}
}
