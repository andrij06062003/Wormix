// Package csx models the exceptions of the original C# server. Code that would throw in C# panics with an
// *Exception; the places that caught exceptions in C# recover them (see Catch), so error handling follows the
// same paths: e.g. any exception while processing a main-server command closes that connection.
package csx

import "fmt"

// Kind of exception, after the .NET exception types the C# code distinguished.
type Kind int

const (
	// Generic is any exception the C# code didn't catch by type (HTTP errors, JSON errors, ...).
	Generic Kind = iota
	// EndOfStream is a truncated message (EndOfStreamException).
	EndOfStream
	// Argument is an invalid header or request (ArgumentException).
	Argument
	// NullReference is a null value where the C# code dereferenced it (NullReferenceException).
	NullReference
)

// Exception is a C#-style exception carried by a panic.
type Exception struct {
	Kind Kind
	Msg  string
}

func (e *Exception) Error() string { return e.Msg }

// Throw panics with an exception of the given kind.
func Throw(kind Kind, format string, args ...any) {
	panic(&Exception{Kind: kind, Msg: fmt.Sprintf(format, args...)})
}

// ThrowErr panics with a generic exception wrapping err.
func ThrowErr(err error) {
	panic(&Exception{Kind: Generic, Msg: err.Error()})
}

// NullRef panics with a NullReferenceException.
func NullRef(what string) {
	panic(&Exception{Kind: NullReference, Msg: "Object reference not set to an instance of an object (" + what + ")"})
}

// Catch runs f and returns the exception it raised, if any. Go runtime errors (nil dereference, index out of
// range) are reported as exceptions too, like their .NET counterparts.
func Catch(f func()) (exc *Exception) {
	defer func() {
		if r := recover(); r != nil {
			exc = FromPanic(r)
		}
	}()
	f()
	return nil
}

// FromPanic converts a recovered panic value to an exception.
func FromPanic(r any) *Exception {
	switch v := r.(type) {
	case *Exception:
		return v
	case error:
		return &Exception{Kind: Generic, Msg: v.Error()}
	default:
		return &Exception{Kind: Generic, Msg: fmt.Sprint(v)}
	}
}
