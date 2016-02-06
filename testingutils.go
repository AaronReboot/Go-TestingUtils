package testingutils

import (
	"fmt"
	"log"
	"reflect"
	"testing"
)

/*	recursively unpacks objects to examine whether they are deep equal with
	EPSILON margin of error allowed for differences between float64 components */
func IsEqual(a, b interface{}) bool {
	const EPSILON = .0000000000001

	//if a and b aren't the same thing => not equal
	if reflect.ValueOf(a).Kind() != reflect.ValueOf(b).Kind() {
		return false
	}

	//if a and b are slices or structs, check their elements
	if reflect.ValueOf(a).Kind() == reflect.Slice {
		//iterate over members, returning false right away if any member is false
		if reflect.ValueOf(a).Len() != reflect.ValueOf(b).Len() {
			return false
		}
		for i := 0; i < reflect.ValueOf(a).Len(); i++ {
			if !IsEqual(reflect.ValueOf(a).Index(i).Interface(), reflect.ValueOf(b).Index(i).Interface()) {
				return false
			}
		}
		return true
	}

	//if a & b are structs, iterate over fields, returning false right away if any member is false
	if reflect.ValueOf(a).Kind() == reflect.Struct {
		if reflect.ValueOf(a).NumField() != reflect.ValueOf(b).NumField() {
			return false
		}
		//iterate over struct's fields
		for i := 0; i < reflect.ValueOf(a).NumField(); i++ {
			if !IsEqual(reflect.ValueOf(a).Field(i).Interface(), reflect.ValueOf(b).Field(i).Interface()) {
				return false
			}
		}
		return true
	}

	//TODO: if a & b are maps, iterate over keys & values in case we need to compare floats
	//Note, maps are already working well for some types because of fallthrough to reflect.DeepEqual

	//if a & b are float64 see whether they are within EPSILON of each other
	if reflect.ValueOf(a).Kind() == reflect.Float64 && reflect.ValueOf(b).Kind() == reflect.Float64 {
		//find diff, by subtracting smaller from greater
		var (
			aflt, bflt float64
			ok         bool
		)
		if aflt, ok = a.(float64); !ok {
			log.Fatal("Fail sanity check: a is not a float64")
		}
		if bflt, ok = b.(float64); !ok {
			log.Fatal("Fail sanity check: b is not a float64")
		}
		//return whether diff is smaller than EPSILON (dunno if aflt or bflt is bigger)
		return ((aflt - bflt) < EPSILON) && ((bflt - aflt) < EPSILON)
	}

	//cover any unhandled case, like intentional ones (err, int, etc) and anything
	//we missed by accident (is ok becaue DeepEqual is conservative)
	return reflect.DeepEqual(a, b)
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
		if reflect.ValueOf(vals).Kind() == reflect.Struct {
			result = make([]reflect.Value, reflect.ValueOf(vals).NumField())
			names = make([]string, reflect.ValueOf(vals).NumField())
			//load them in
			for i := 0; i < reflect.ValueOf(vals).NumField(); i++ {
				result[i] = reflect.ValueOf(vals).Field(i)
				names[i] = reflect.ValueOf(vals).Type().Field(i).Name
			}
		} else {
			result = make([]reflect.Value, 1)
			result[0] = reflect.ValueOf(vals)
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
		if !IsEqual(got[i].Interface(), expect[i].Interface()) {
			//if this function returns more than one result, figure out the name of problem result
			var name string
			if len(got) > 1 {
				name = " (" + exNames[i] + ")"
			}
			t.Logf("Expected%s: %s\n\n", name, PrintTruncated(expect[i].Interface()))
			t.Logf("Got     %s: %s\n\n", name, PrintTruncated(got[i].Interface()))
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
