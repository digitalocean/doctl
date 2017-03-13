package errors

import "fmt"

func addDetails(err error, dets map[string]string) {
	switch specific := err.(type) {
	case *HereError:
		dets["location"] = specific.FullLocation()
		addDetails(specific.error, dets)
	case *CauseError:
		dets["cause"] = specific.cause.Error()

		i := 2
		if cause, ok := specific.cause.(*CauseError); ok {
			for {
				dets[fmt.Sprintf("cause%d", i)] = cause.cause.Error()

				if sub, ok := cause.cause.(*CauseError); ok {
					cause = sub
				} else {
					break
				}
			}
		}

		addDetails(specific.error, dets)
	case *TraceError:
		dets["trace"] = specific.trace
		addDetails(specific.error, dets)
	case *ContextError:
		dets["context"] = specific.Context()
		addDetails(specific.error, dets)
	case *SubjectError:
		dets["subject"] = fmt.Sprintf("%s", specific.Subject())
		addDetails(specific.error, dets)
	default:
		dets["error"] = err.Error()
	}
}

// Derive a map of detailed information about an error.
// For HereErrors, map includes a "location" key
// For CauseErrors, map includes one or more "cause" keys
// For TraceErrors, map includes a "trace" key
// For ContextErrors, map includes a "context" key
// For SubjectErrors, map includes a "subject" key
func Details(err error) map[string]string {
	dets := map[string]string{}

	addDetails(err, dets)

	return dets
}
