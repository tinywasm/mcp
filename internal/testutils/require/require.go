package require

import (
	"testing"
	"github.com/tinywasm/mcp/internal/testutils/assert"
)

func Equal(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Equal(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func NotEqual(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotEqual(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func EqualValues(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.EqualValues(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func NoError(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if !assert.NoError(t, err, msgAndArgs...) {
		t.FailNow()
	}
}

func Error(t testing.TB, err error, msgAndArgs ...any) {
	t.Helper()
	if !assert.Error(t, err, msgAndArgs...) {
		t.FailNow()
	}
}

func EqualError(t testing.TB, theError error, errString string, msgAndArgs ...any) {
	t.Helper()
	if !assert.EqualError(t, theError, errString, msgAndArgs...) {
		t.FailNow()
	}
}

func ErrorIs(t testing.TB, err, target error, msgAndArgs ...any) {
	t.Helper()
	if !assert.ErrorIs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

func ErrorAs(t testing.TB, err error, target any, msgAndArgs ...any) {
	t.Helper()
	if !assert.ErrorAs(t, err, target, msgAndArgs...) {
		t.FailNow()
	}
}

func NotNil(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotNil(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

func Nil(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Nil(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

func True(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if !assert.True(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

func False(t testing.TB, value bool, msgAndArgs ...any) {
	t.Helper()
	if !assert.False(t, value, msgAndArgs...) {
		t.FailNow()
	}
}

func Contains(t testing.TB, s, contains any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Contains(t, s, contains, msgAndArgs...) {
		t.FailNow()
	}
}

func NotContains(t testing.TB, s, contains any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotContains(t, s, contains, msgAndArgs...) {
		t.FailNow()
	}
}

func Len(t testing.TB, object any, length int, msgAndArgs ...any) {
	t.Helper()
	if !assert.Len(t, object, length, msgAndArgs...) {
		t.FailNow()
	}
}

func Empty(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Empty(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

func NotEmpty(t testing.TB, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotEmpty(t, object, msgAndArgs...) {
		t.FailNow()
	}
}

func Panics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	if !assert.Panics(t, f, msgAndArgs...) {
		t.FailNow()
	}
}

func NotPanics(t testing.TB, f func(), msgAndArgs ...any) {
	t.Helper()
	if !assert.NotPanics(t, f, msgAndArgs...) {
		t.FailNow()
	}
}

func GreaterOrEqual(t testing.TB, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !assert.GreaterOrEqual(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

func LessOrEqual(t testing.TB, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !assert.LessOrEqual(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

func Greater(t testing.TB, e1, e2 any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Greater(t, e1, e2, msgAndArgs...) {
		t.FailNow()
	}
}

func JSONEq(t testing.TB, expected string, actual string, msgAndArgs ...any) {
	t.Helper()
	if !assert.JSONEq(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func Subset(t testing.TB, list, subset any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Subset(t, list, subset, msgAndArgs...) {
		t.FailNow()
	}
}

func Eventually(t testing.TB, condition func() bool, waitFor any, tick any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Eventually(t, condition, waitFor, tick, msgAndArgs...) {
		t.FailNow()
	}
}

func IsType(t testing.TB, expectedType any, object any, msgAndArgs ...any) {
	t.Helper()
	if !assert.IsType(t, expectedType, object, msgAndArgs...) {
		t.FailNow()
	}
}

func NotSame(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotSame(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func Same(t testing.TB, expected, actual any, msgAndArgs ...any) {
	t.Helper()
	if !assert.Same(t, expected, actual, msgAndArgs...) {
		t.FailNow()
	}
}

func NotZero(t testing.TB, i any, msgAndArgs ...any) {
	t.Helper()
	if !assert.NotZero(t, i, msgAndArgs...) {
		t.FailNow()
	}
}

func InDelta(t testing.TB, expected, actual any, delta float64, msgAndArgs ...any) {
	t.Helper()
	if !assert.InDelta(t, expected, actual, delta, msgAndArgs...) {
		t.FailNow()
	}
}
