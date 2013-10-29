package qbs

import (
	"fmt"
	"reflect"
	"runtime"
)

type Assert struct {
	tester
}

type tester interface {
	Fail()
	Failed() bool
	FailNow()
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Skip(args ...interface{})
	Skipf(format string, args ...interface{})
	SkipNow()
	Skipped() bool
}

type benchmarker interface {
	StartTimer()
	StopTimer()
}

func NewAssert(t tester) *Assert {
	return &Assert{t}
}

func (ast *Assert) Nil(value interface{}, logs ...interface{}) {
	ast.nilAssert(false, true, value, logs...)
}

func (ast *Assert) MustNil(value interface{}, logs ...interface{}) {
	ast.nilAssert(true, true, value, logs...)
}

func (ast *Assert) NotNil(value interface{}, logs ...interface{}) {
	ast.nilAssert(false, false, value, logs...)
}

func (ast *Assert) MustNotNil(value interface{}, logs ...interface{}) {
	ast.nilAssert(true, false, value, logs...)
}

func (ast *Assert) nilAssert(fatal bool, isNil bool, value interface{}, logs ...interface{}) {
	if isNil != (value == nil || reflect.ValueOf(value).IsNil()) {
		ast.logCaller()
		if len(logs) > 0 {
			ast.Log(logs...)
		} else {
			if isNil {
				ast.Log("value is not nil:", value)
			} else {
				ast.Log("value is nil")
			}
		}
		ast.failIt(fatal)
	}
}

func (ast *Assert) True(boolValue bool, logs ...interface{}) {
	ast.trueAssert(false, boolValue, logs...)
}

func (ast *Assert) MustTrue(boolValue bool, logs ...interface{}) {
	ast.trueAssert(true, boolValue, logs...)
}

func (ast *Assert) trueAssert(fatal bool, value bool, logs ...interface{}) {
	if !value {
		ast.logCaller()
		if len(logs) > 0 {
			ast.Log(logs...)
		} else {
			ast.Logf("value is not true")
		}
		ast.failIt(fatal)
	}
}

func (ast *Assert) Equal(expected, actual interface{}, logs ...interface{}) {
	ast.equalSprintAssert(false, true, expected, actual, logs...)
}

func (ast *Assert) MustEqual(expected, actual interface{}, logs ...interface{}) {
	ast.equalSprintAssert(true, true, expected, actual, logs...)
}

func (ast *Assert) NotEqual(expected, actual interface{}, logs ...interface{}) {
	ast.equalSprintAssert(false, false, expected, actual, logs...)
}

func (ast *Assert) MustNotEqual(expected, actual interface{}, logs ...interface{}) {
	ast.equalSprintAssert(true, false, expected, actual, logs...)
}

func (ast *Assert) equalSprintAssert(fatal bool, isEqual bool, expected, actual interface{}, logs ...interface{}) {
	expectedStr := fmt.Sprint(expected)
	actualStr := fmt.Sprint(actual)
	if isEqual != (expectedStr == actualStr) {
		ast.logCaller()
		if len(logs) > 0 {
			ast.Log(logs...)
		} else {
			if isEqual {
				ast.Log("Values not equal")
			} else {
				ast.Log("Values equal")
			}
		}
		ast.Log("Expected: ", expected)
		ast.Log("Actual: ", actual)
		ast.failIt(fatal)
	}
}

func (ast *Assert) logCaller() {
	_, file, line, _ := runtime.Caller(3)
	ast.Logf("Caller: %v:%d", file, line)
}

func (ast *Assert) failIt(fatal bool) {
	if fatal {
		ast.FailNow()
	} else {
		ast.Fail()
	}
}
