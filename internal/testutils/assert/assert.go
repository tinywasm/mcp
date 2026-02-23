package assert

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

var AnError = errors.New("assert.AnError general error for testing")

func Equal(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		msg := fmt.Sprintf("Not equal: \nexpected: %#v\nactual  : %#v", expected, actual)
		logError(t, msg, msgAndArgs...)
		return false
	}
	return true
}

func NotEqual(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		logError(t, fmt.Sprintf("Should not be equal: %#v", actual), msgAndArgs...)
		return false
	}
	return true
}

func EqualValues(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
	t.Helper()
	if reflect.DeepEqual(expected, actual) {
		return true
	}

	expectedValue := reflect.ValueOf(expected)
	actualValue := reflect.ValueOf(actual)
	if expectedValue.IsValid() && actualValue.IsValid() && actualValue.Type().ConvertibleTo(expectedValue.Type()) {
		if reflect.DeepEqual(expected, actualValue.Convert(expectedValue.Type()).Interface()) {
			return true
		}
	}

	return Equal(t, expected, actual, msgAndArgs...)
}

func NoError(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err != nil {
		msg := fmt.Sprintf("Received unexpected error:\n%+v", err)
		logError(t, msg, msgAndArgs...)
		return false
	}
	return true
}

func Error(t testing.TB, err error, msgAndArgs ...any) bool {
	t.Helper()
	if err == nil {
		msg := "An error is expected but got nil."
		logError(t, msg, msgAndArgs...)
		return false
	}
	return true
}

func EqualError(t testing.TB, theError error, errString string, msgAndArgs ...any) bool {
	t.Helper()
    if theError == nil {
        if errString == "" {
            return true
        }
        logError(t, "Expected error but got nil", msgAndArgs...)
        return false
    }
	if theError.Error() != errString {
		logError(t, fmt.Sprintf("Error message not equal:\nexpected: %q\nactual  : %q", errString, theError.Error()), msgAndArgs...)
		return false
	}
	return true
}

func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) bool {
	t.Helper()
	if !errors.Is(err, target) {
		msg := fmt.Sprintf("Error expected to be: %v\nbut was: %v", target, err)
		logError(t, msg, msgAndArgs...)
		return false
	}
	return true
}

func ErrorAs(t testing.TB, err error, target any, msgAndArgs ...any) bool {
	t.Helper()
	if errors.As(err, target) {
		return true
	}
	logError(t, fmt.Sprintf("Should be able to cast error to target type"), msgAndArgs...)
	return false
}

func NotNil(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()
	if !isNil(object) {
		return true
	}
	logError(t, "Expected not nil", msgAndArgs...)
	return false
}

func Nil(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()
	if isNil(object) {
		return true
	}
	msg := fmt.Sprintf("Expected nil, but got: %#v", object)
	logError(t, msg, msgAndArgs...)
	return false
}

func True(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if !value {
		logError(t, "Should be true", msgAndArgs...)
		return false
	}
	return true
}

func False(t testing.TB, value bool, msgAndArgs ...any) bool {
	t.Helper()
	if value {
		logError(t, "Should be false", msgAndArgs...)
		return false
	}
	return true
}

func Contains(t testing.TB, s, contains any, msgAndArgs ...any) bool {
	t.Helper()
	ok, found := includeElement(s, contains)
	if !ok {
		logError(t, fmt.Sprintf("%#v could not be applied to Contains", s), msgAndArgs...)
		return false
	}
	if !found {
		logError(t, fmt.Sprintf("%#v does not contain %#v", s, contains), msgAndArgs...)
		return false
	}
	return true
}

func NotContains(t testing.TB, s, contains any, msgAndArgs ...any) bool {
	t.Helper()
	ok, found := includeElement(s, contains)
	if !ok {
		logError(t, fmt.Sprintf("%#v could not be applied to NotContains", s), msgAndArgs...)
		return false
	}
	if found {
		logError(t, fmt.Sprintf("%#v should not contain %#v", s, contains), msgAndArgs...)
		return false
	}
	return true
}

func Len(t testing.TB, object any, length int, msgAndArgs ...any) bool {
	t.Helper()
	l := 0
	ok := false
	v := reflect.ValueOf(object)
	switch v.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		l = v.Len()
		ok = true
	}
	if !ok {
		logError(t, fmt.Sprintf("%#v does not have a length", object), msgAndArgs...)
		return false
	}
	if l != length {
		logError(t, fmt.Sprintf("Expected length %d, got %d", length, l), msgAndArgs...)
		return false
	}
	return true
}

func Empty(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()
	if isEmpty(object) {
		return true
	}
	logError(t, fmt.Sprintf("Should be empty, but was %v", object), msgAndArgs...)
	return false
}

func NotEmpty(t testing.TB, object any, msgAndArgs ...any) bool {
	t.Helper()
	if !isEmpty(object) {
		return true
	}
	logError(t, fmt.Sprintf("Should not be empty, but was %v", object), msgAndArgs...)
	return false
}

func Panics(t testing.TB, f func(), msgAndArgs ...any) bool {
	t.Helper()
	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		f()
	}()
	if !didPanic {
		logError(t, "func did not panic", msgAndArgs...)
		return false
	}
	return true
}

func NotPanics(t testing.TB, f func(), msgAndArgs ...any) bool {
	t.Helper()
	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		f()
	}()
	if didPanic {
		logError(t, "func panicked", msgAndArgs...)
		return false
	}
	return true
}

func GreaterOrEqual(t testing.TB, e1, e2 any, msgAndArgs ...any) bool {
    t.Helper()
    f1, ok1 := toFloat(e1)
    f2, ok2 := toFloat(e2)
    if !ok1 || !ok2 {
        logError(t, fmt.Sprintf("GreaterOrEqual: cannot compare %T and %T", e1, e2), msgAndArgs...)
        return false
    }
    if !(f1 >= f2) {
        logError(t, fmt.Sprintf("%v is not greater or equal to %v", e1, e2), msgAndArgs...)
        return false
    }
    return true
}

func LessOrEqual(t testing.TB, e1, e2 any, msgAndArgs ...any) bool {
    t.Helper()
    f1, ok1 := toFloat(e1)
    f2, ok2 := toFloat(e2)
    if !ok1 || !ok2 {
        logError(t, fmt.Sprintf("LessOrEqual: cannot compare %T and %T", e1, e2), msgAndArgs...)
        return false
    }
    if !(f1 <= f2) {
        logError(t, fmt.Sprintf("%v is not less or equal to %v", e1, e2), msgAndArgs...)
        return false
    }
    return true
}

func Greater(t testing.TB, e1, e2 any, msgAndArgs ...any) bool {
    t.Helper()
    f1, ok1 := toFloat(e1)
    f2, ok2 := toFloat(e2)
    if !ok1 || !ok2 {
        logError(t, fmt.Sprintf("Greater: cannot compare %T and %T", e1, e2), msgAndArgs...)
        return false
    }
    if !(f1 > f2) {
        logError(t, fmt.Sprintf("%v is not greater than %v", e1, e2), msgAndArgs...)
        return false
    }
    return true
}

func toFloat(i any) (float64, bool) {
    switch v := i.(type) {
    case int: return float64(v), true
    case int8: return float64(v), true
    case int16: return float64(v), true
    case int32: return float64(v), true
    case int64: return float64(v), true
    case uint: return float64(v), true
    case uint8: return float64(v), true
    case uint16: return float64(v), true
    case uint32: return float64(v), true
    case uint64: return float64(v), true
    case float32: return float64(v), true
    case float64: return v, true
    }
    return 0, false
}

func JSONEq(t testing.TB, expected string, actual string, msgAndArgs ...any) bool {
    t.Helper()
    var expectedJSON, actualJSON any
    if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
        logError(t, fmt.Sprintf("Expected value is not valid JSON: %v", err), msgAndArgs...)
        return false
    }
    if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
        logError(t, fmt.Sprintf("Actual value is not valid JSON: %v", err), msgAndArgs...)
        return false
    }
    return Equal(t, expectedJSON, actualJSON, msgAndArgs...)
}

func Subset(t testing.TB, list, subset any, msgAndArgs ...any) bool {
    t.Helper()
    listVal := reflect.ValueOf(list)
    subsetVal := reflect.ValueOf(subset)
    if listVal.Kind() != reflect.Slice || subsetVal.Kind() != reflect.Slice {
        logError(t, "Subset: arguments must be slices", msgAndArgs...)
        return false
    }

    for i := 0; i < subsetVal.Len(); i++ {
        element := subsetVal.Index(i).Interface()
        ok, found := includeElement(list, element)
        if !ok || !found {
            logError(t, fmt.Sprintf("%#v is not a subset of %#v (missing %v)", subset, list, element), msgAndArgs...)
            return false
        }
    }
    return true
}

func Eventually(t testing.TB, condition func() bool, waitFor any, tick any, msgAndArgs ...any) bool {
    t.Helper()
	var duration time.Duration
	var tickDuration time.Duration

	// simple handling
	if d, ok := waitFor.(time.Duration); ok {
		duration = d
	} else {
		duration = 1 * time.Second
	}
	if d, ok := tick.(time.Duration); ok {
		tickDuration = d
	} else {
		tickDuration = 10 * time.Millisecond
	}

    deadline := time.Now().Add(duration)
    for time.Now().Before(deadline) {
        if condition() {
            return true
        }
        time.Sleep(tickDuration)
    }
    logError(t, "Condition never satisfied", msgAndArgs...)
    return false
}

func IsType(t testing.TB, expectedType any, object any, msgAndArgs ...any) bool {
    t.Helper()
    t1 := reflect.TypeOf(expectedType)
    t2 := reflect.TypeOf(object)
    if t1 != t2 {
        logError(t, fmt.Sprintf("Object expected to be of type %v, but was %v", t1, t2), msgAndArgs...)
        return false
    }
    return true
}

func NotSame(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
    t.Helper()
    if expected == actual {
         logError(t, fmt.Sprintf("Expected and actual point to the same object: %p", expected), msgAndArgs...)
         return false
    }
    return true
}

func Same(t testing.TB, expected, actual any, msgAndArgs ...any) bool {
    t.Helper()
    if expected != actual {
         logError(t, fmt.Sprintf("Expected and actual do not point to the same object: %p != %p", expected, actual), msgAndArgs...)
         return false
    }
    return true
}

func NotZero(t testing.TB, i any, msgAndArgs ...any) bool {
    t.Helper()
    if i == nil {
        logError(t, "Should not be zero, but was nil", msgAndArgs...)
        return false
    }
    v := reflect.ValueOf(i)
    if v.IsZero() {
        logError(t, fmt.Sprintf("Should not be zero, but was %v", i), msgAndArgs...)
        return false
    }
    return true
}

func InDelta(t testing.TB, expected, actual any, delta float64, msgAndArgs ...any) bool {
    t.Helper()
    f1, ok1 := toFloat(expected)
    f2, ok2 := toFloat(actual)
    if !ok1 || !ok2 {
        logError(t, fmt.Sprintf("InDelta: cannot compare %T and %T", expected, actual), msgAndArgs...)
        return false
    }
    diff := math.Abs(f1 - f2)
    if diff > delta {
        logError(t, fmt.Sprintf("Max difference between %v and %v allowed is %v, but difference was %v", expected, actual, delta, diff), msgAndArgs...)
        return false
    }
    return true
}

// Helpers

func logError(t testing.TB, msg string, msgAndArgs ...any) {
	t.Helper()
	if len(msgAndArgs) == 0 {
		t.Error(msg)
		return
	}

	var userMsg string
	if len(msgAndArgs) == 1 {
		userMsg = fmt.Sprint(msgAndArgs[0])
	} else {
		if format, ok := msgAndArgs[0].(string); ok {
			userMsg = fmt.Sprintf(format, msgAndArgs[1:]...)
		} else {
			userMsg = fmt.Sprint(msgAndArgs...)
		}
	}

	t.Errorf("%s\n%s", msg, userMsg)
}

func isNil(object any) bool {
	if object == nil {
		return true
	}
	value := reflect.ValueOf(object)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return value.IsNil()
	}
	return false
}

func isEmpty(object any) bool {
	if object == nil {
		return true
	}
	objValue := reflect.ValueOf(object)
	switch objValue.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return objValue.Len() == 0
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		deref := objValue.Elem().Interface()
		return isEmpty(deref)
	default:
		return false
	}
}

func includeElement(list any, element any) (ok, found bool) {
	listValue := reflect.ValueOf(list)
	elementValue := reflect.ValueOf(element)
	defer func() {
		if e := recover(); e != nil {
			ok = false
			found = false
		}
	}()

	if listValue.Kind() == reflect.String {
		return true, strings.Contains(listValue.String(), elementValue.String())
	}

	if listValue.Kind() == reflect.Map {
		mapKeys := listValue.MapKeys()
		for i := 0; i < len(mapKeys); i++ {
			if reflect.DeepEqual(mapKeys[i].Interface(), element) {
				return true, true
			}
		}
		return true, false
	}

	for i := 0; i < listValue.Len(); i++ {
		if reflect.DeepEqual(listValue.Index(i).Interface(), element) {
			return true, true
		}
	}
	return true, false
}
