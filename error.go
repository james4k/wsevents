package wsevents

import (
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrUnknown                = errors.New("wsevents: unknown error")
	ErrUnexpectedEvent        = errors.New("wsevents: unexpected event")
	ErrMissingEventName       = errors.New("wsevents: missing event name in json object")
	ErrMissingEventArgs       = errors.New("wsevents: missing event args in json object")
	ErrEventArgsCountMismatch = errors.New("wsevents: number of event method args didn't match received")
)

type ArgsMismatchError struct {
	Expected []reflect.Kind
	Actual   []reflect.Kind
}

func (e ArgsMismatchError) Error() string {
	return fmt.Sprintf("wsevents: type mismatch (args %v, got %v)", e.Expected, e.Actual)
}

func makeArgsMismatchError(fn reflect.Type, jsargs []interface{}) *ArgsMismatchError {
	count := fn.NumIn()
	expected := make([]reflect.Kind, 0, count-1)
	for i := 1; i < count; i += 1 {
		expected = append(expected, fn.In(i).Kind())
	}

	count = len(jsargs)
	actual := make([]reflect.Kind, 0, count)
	for i := 0; i < count; i += 1 {
		actual = append(actual, reflect.TypeOf(jsargs[i]).Kind())
	}

	return &ArgsMismatchError{expected, actual}
}
