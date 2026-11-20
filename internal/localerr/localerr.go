// Package localerr makes sentinel errors whose messages follow the app's
// language, while errors.Is can still compare them by identity.
package localerr

// Error is a sentinel error with a message formatted on each Error call.
type Error struct {
	message func() string
}

// New returns an Error that says message.
func New(message func() string) *Error {
	return &Error{message: message}
}

// Error says the message in the app's current language.
func (e *Error) Error() string {
	return e.message()
}
