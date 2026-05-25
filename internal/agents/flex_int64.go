package agents

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// flexInt64 unmarshals protobuf JSON int64 fields, which grpc-gateway emits as
// decimal strings (not JSON numbers) to avoid precision loss.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*f = 0
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf("flexInt64: parse %q: %w", s, err)
		}
		*f = flexInt64(n)
		return nil
	}
	var n int64
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	*f = flexInt64(n)
	return nil
}

func (f flexInt64) Int64() int64 {
	return int64(f)
}
