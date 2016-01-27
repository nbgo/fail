// Package fail provides routines to create/wrap errors which will have additional information (like place of creation).
package fail

import (
	"bytes"
	"fmt"
	"gopkg.in/stack.v1"
	"reflect"
	"strings"
	"errors"
)

// CompositeError is the interface that represents an error that can provide information about its cause/inner error.
//
// InnerError returns the inner error.
type CompositeError interface {
	error
	InnerError() error
}

// ErrorWithLocation is the interface that represents an error that has information about the place in code where it occurred.
//
// Location is supposed to return code line information (source file and line number) and function name where error occurred.
type ErrorWithLocation interface {
	error
	Location() string
}

// ErrorWithStackTrace is the interface that represents an error that has information about stack trace.
//
// StackTrace is supposed to return stack trace as a multiline string where each line has information about code line and function.
type ErrorWithStackTrace interface {
	error
	StackTrace() string
}

// ErrorWrapper is the interface that represents an object that wraps original error.
//
// GetOriginalError returns original error that was wrapped.
type ErrorWrapper interface {
	OriginalError() error
}

// ErrorWithFields is error which provides additional information as map.
//
// GetFields returns error's additional information as map
type ErrorWithFields interface {
	error
	Fields() map[string]interface{}
}

// ErrWithReason is error with message and reason.
// Implements CompositeError
type ErrWithReason struct {
	Message string
	Reason  error
}

func (err ErrWithReason) Error() string {
	return fmt.Sprintf("%v: %v", err.Message, err.Reason)
}
// InnerError implements Composite.InnerError
func (err ErrWithReason) InnerError() error {
	return err.Reason
}

type extendedError struct {
	originalError error
	innerError    error
	location      stack.Call
	stackTrace    stack.CallStack
}

func (extErr extendedError) InnerError() error {
	var result error

	if originalCompositeError, isOriginalCompositeError := extErr.originalError.(CompositeError); isOriginalCompositeError {
		result = originalCompositeError.InnerError()
	}

	if result == nil {
		result = extErr.innerError
	}

	return result
}
func (extErr extendedError) Error() string {
	return extErr.originalError.Error()
}
func (extErr extendedError) Location() string {
	// %n is implemented by stack.Call
	//noinspection GoPlaceholderCount
	return fmt.Sprintf("%+v (%n)", extErr.location, extErr.location)
}
func (extErr extendedError) StackTrace() string {
	return StackTraceToString(extErr.stackTrace)
}
func (extErr extendedError) OriginalError() error {
	originalError := extErr.originalError
	if errorWrapper, isErrorWrapper := originalError.(ErrorWrapper); isErrorWrapper {
		return errorWrapper.OriginalError()
	}
	return originalError
}
func (extErr extendedError) Fields() map[string]interface{} {
	if errWithFields, isErrWithFields := extErr.OriginalError().(ErrorWithFields); isErrWithFields {
		return errWithFields.Fields()
	}
	return nil
}

// New creates a new error that captures stack trace and location where it is created
// and keeps information about the original error which is provided as single argument.
// The main idea is supply original error with additional information (stack trace and location).
// Newly created error implements CompositeError, ErrorWithLocation, ErrorWithStackTrace.
func New(err error, additionalStackSkip ...int) error {
	stackSkip := 1
	if len(additionalStackSkip) > 0 {
		stackSkip += additionalStackSkip[0]
	}

	return NewWithInner(err, nil, stackSkip)
}

// NewWithInner creates a new error that captures stack trace and location where it is created
// and keeps information about the original error and its reason (another error).
// The main idea is supply original error with additional information (stack trace and location)
// and keep its reason (another error).
// Newly created error implements CompositeError, ErrorWithLocation, ErrorWithStackTrace.
func NewWithInner(err, inner error, additionalStackSkip ...int) error {
	stackSkip := 1
	if len(additionalStackSkip) > 0 {
		stackSkip += additionalStackSkip[0]
	}
	call := stack.Caller(stackSkip)
	return &extendedError{err, inner, call, stack.Trace().TrimBelow(call).TrimRuntime()}
}

// NewErrWithReason creates new error with reason.
func NewErrWithReason(message string, reason error) error {
	return New(ErrWithReason{message, reason}, 1)
}

// GetInner returns inner error for the given error.
// If given error implements CompositeError then InnerError is called and its result is returned.
// Otherwise nil is returned.
func GetInner(err error) error {
	if compositeError, isCompositeError := err.(CompositeError); isCompositeError {
		return compositeError.InnerError()
	}

	return nil
}

// GetLocation returns code line and function where error occurred.
// If given error implements ErrorWithLocation then Location is called and its result is returned.
// Otherwise empty string is returned.
func GetLocation(err error) string {
	if errorWithLocation, isErrorWithLocation := err.(ErrorWithLocation); isErrorWithLocation {
		return errorWithLocation.Location()
	}

	return ""
}

// GetStackTrace returns stack trace for the given error.
// If given error implements ErrorWithStackTrace then StackTrace is called and its result is returned.
// Otherwise empty string is returned.
func GetStackTrace(err error) string {
	if errorWithStackTrace, isErrorWithStackTrace := err.(ErrorWithStackTrace); isErrorWithStackTrace {
		return errorWithStackTrace.StackTrace()
	}

	return ""
}

// GetFullDetails returns information about the error itself
// and all its inner errors (and their stack traces) recursively.
func GetFullDetails(err error) string {
	var result bytes.Buffer

	currErr := err
	for currErr != nil {
		if result.Len() > 0 {
			result.WriteString("\n")
		}
		result.WriteString(fmt.Sprintf("%v: %v", GetType(currErr), currErr))

		if errorWithStackTrace, isErrorWithStackTrace := currErr.(ErrorWithStackTrace); isErrorWithStackTrace {
			stackTrace := errorWithStackTrace.StackTrace()
			if stackTrace != "" {
				const ident = "    "
				result.WriteString(fmt.Sprintf("\n%v%v", ident, strings.Replace(stackTrace, "\n", "\n" + ident, -1)))
			}
		}

		currErr = GetInner(currErr)
	}

	return result.String()
}

// GetType returns the type of the original error.
// If provided error implements ErrorWrapper then GetType is run for its original error
// until first non-ErrorWrapper is found.
func GetType(err error) reflect.Type {
	var errToGetTypeOf error
	if errorWrapper, isErrorWrapper := err.(ErrorWrapper); isErrorWrapper {
		errToGetTypeOf = errorWrapper.OriginalError()
	} else {
		errToGetTypeOf = err
	}
	return reflect.TypeOf(errToGetTypeOf)
}

// GetOriginalError returns the original error.
// If provided error implements ErrorWrapper then first non-ErrorWrapper is search for recursively.
func GetOriginalError(err error) error {
	if errorWrapper, isErrorWrapper := err.(ErrorWrapper); isErrorWrapper {
		return errorWrapper.OriginalError()
	}
	return err
}

// News creates new error from text.
func News(text string) error {
	return New(errors.New(text), 1)
}

// Newf creates new error from formatted text.
func Newf(format string, a ...interface{}) error {
	return New(fmt.Errorf(format, a...), 1)
}

// StackTraceToString converts stack trace in string representation.
func StackTraceToString(stackTrace stack.CallStack) string {
	var result bytes.Buffer
	for _, call := range stackTrace {
		if result.Len() > 0 {
			result.WriteString("\n")
		}

		// %n is implemented by stack.Call
		//noinspection GoPlaceholderCount
		result.WriteString(fmt.Sprintf("%+v (%n)", call, call))
	}
	return result.String()
}

// StackTrace returns current stack trace.
// Can be called with optional integer single parameter which defines how many closest callers skip.
func StackTrace(additionalStackSkip ...int) string {
	stackSkip := 1
	if len(additionalStackSkip) > 0 {
		stackSkip += additionalStackSkip[0]
	}

	call := stack.Caller(stackSkip)
	stackTrace := stack.Trace().TrimBelow(call).TrimRuntime()
	return StackTraceToString(stackTrace)
}

// IsError check if the first argument error is the same instance as the second argument error.
// If the first error is CompositeError than IsError is called recursively for CompositeError.InnerError().
func IsError(whereToFind, errToFind error) bool {
	if whereToFind == errToFind {
		return true
	}

	if compositeError, isCompositeError := whereToFind.(CompositeError); isCompositeError {
		return IsError(compositeError.InnerError(), errToFind)
	}

	return false
}

// GetErrorByType returns error if desired type.
func GetErrorByType(whereToFind, errExampleToFind error) error {
	if AreErrorsOfEqualType(whereToFind, errExampleToFind) {
		return whereToFind
	}

	if compositeError, isCompositeError := whereToFind.(CompositeError); isCompositeError {
		return GetErrorByType(compositeError.InnerError(), errExampleToFind)
	}

	return nil
}

// AreErrorsOfEqualType checks if 2 errors are of the same type.
// Always returns false if one of the arguments is nil.
func AreErrorsOfEqualType(err1, err2 error) bool {
	if err1 == nil || err2 == nil {
		return false
	}

	err1Type := reflect.TypeOf(err1)
	if err1Type.Kind() == reflect.Ptr {
		err1Type = err1Type.Elem()
	}
	err2Type := reflect.TypeOf(err2)
	if err2Type.Kind() == reflect.Ptr {
		err2Type = err2Type.Elem()
	}
	if err1Type == err2Type {
		return true
	}
	return false;
}