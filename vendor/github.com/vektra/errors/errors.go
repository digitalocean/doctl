package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// Imported functionality from the stdlib errors package

// New returns an error that formats as the given text.
func New(text string) *ErrorString {
	return &ErrorString{text}
}

// Created a new error formated according to the fmt rules.
func Format(f string, val ...interface{}) error {
	return fmt.Errorf(f, val...)
}

// ErrorString is a trivial implementation of error.
type ErrorString struct {
	s string
}

func (e *ErrorString) Error() string {
	return e.s
}

// HereError wraps another error with location information
type HereError struct {
	error
	pc  uintptr
	loc string
}

// Wrap an error with location information derived from the caller location
func Here(orig error) error {
	if orig == nil {
		return nil
	}

	// If the error is already a Here, then don't redecorate it because we want
	// to preserve the most accurate info, which is upstream of this call.
	if he, ok := orig.(*HereError); ok {
		return he
	}

	pc, file, line, ok := runtime.Caller(1)

	if ok {
		return &HereError{
			error: orig,
			pc:    pc,
			loc:   fmt.Sprintf("%s:%d", file, line),
		}
	}

	return &HereError{error: orig}
}

// Return a good string representation of the location and error
func (h *HereError) Error() string {
	return h.Location() + ": " + h.error.Error()
}

// Return the full path and line information for the location
func (h *HereError) FullLocation() string {
	return h.loc
}

// Return a short version of the location information
func (h *HereError) Location() string {
	lastSlash := strings.LastIndex(h.loc, "/")
	secondLastSlash := strings.LastIndex(h.loc[:lastSlash], "/")

	return h.loc[secondLastSlash+1:]
}

// Contains 2 errors, an updated error and a causing error
type CauseError struct {
	error
	cause error
}

// Wraps an error containing the information about what caused this error
func Cause(err error, cause error) error {
	if err == nil {
		return nil
	}

	return &CauseError{
		error: err,
		cause: cause,
	}
}

// Return the causing error
func (c *CauseError) Cause() error {
	return c.cause
}

// Contains an error and a stacktrace
type TraceError struct {
	error
	trace string
}

// Wraps an error with a stacktrace derived from the calling location
func Trace(err error) error {
	if err == nil {
		return nil
	}

	buf := make([]byte, 1024)
	sz := runtime.Stack(buf, false)

	return &TraceError{
		error: err,
		trace: string(buf[:sz]),
	}
}

// Return the stacktrace
func (t *TraceError) Trace() string {
	return t.trace
}

// Adds a string describing the context of this error without changing
// the underlying error itself.
type ContextError struct {
	error
	context string
}

func Context(err error, ctx string) error {
	if err == nil {
		return nil
	}

	return &ContextError{
		error:   err,
		context: ctx,
	}
}

func (c *ContextError) Context() string {
	return c.context
}

// Return a string containing the context as well as the underlying error
func (c *ContextError) Error() string {
	return c.context + ": " + c.error.Error()
}

// Attaches a subject to an error. An example would be failing
// to open a file, the path of the file could be added as the subject
// to be shown or retrieved later.
type SubjectError struct {
	error
	subject interface{}
}

func Subject(err error, sub interface{}) error {
	if err == nil {
		return nil
	}

	return &SubjectError{
		error:   err,
		subject: sub,
	}
}

// Return the subject of the error
func (c *SubjectError) Subject() interface{} {
	return c.subject
}

// Return a string containing the context as well as the underlying error
// This uses fmt.Sprintf's %s to convert the subject to a string.
func (c *SubjectError) Error() string {
	return fmt.Sprintf("%s: %s", c.error.Error(), c.subject)
}
