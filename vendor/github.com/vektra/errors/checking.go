package errors

// Remove any Here, Cause, Trace, Context or Subject wrappers from err
func Unwrap(err error) error {
	for {
		switch specific := err.(type) {
		case *HereError:
			err = specific.error
		case *CauseError:
			err = specific.error
		case *TraceError:
			err = specific.error
		case *ContextError:
			err = specific.error
		case *SubjectError:
			err = specific.error
		default:
			return err
		}
	}
}

// Check 2 errors are equal by removing any context wrappers
func Equal(err1, err2 error) bool {
	return Unwrap(err1) == Unwrap(err2)
}
