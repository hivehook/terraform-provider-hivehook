package provider

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// setOptionalString writes the underlying string into m when v is set.
func setOptionalString(m map[string]any, key string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueString()
	}
}

// setOptionalInt64 writes the underlying int64 into m when v is set.
func setOptionalInt64(m map[string]any, key string, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueInt64()
	}
}

// setOptionalBool writes the underlying bool into m when v is set.
func setOptionalBool(m map[string]any, key string, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		m[key] = v.ValueBool()
	}
}

// setOptionalNormalizedJSON parses a jsontypes.Normalized attribute and writes
// the decoded value into m. Returns an error when the JSON payload is malformed.
func setOptionalNormalizedJSON(m map[string]any, key string, v jsontypes.Normalized) error {
	if !v.IsNull() && !v.IsUnknown() {
		var parsed any
		if err := json.Unmarshal([]byte(v.ValueString()), &parsed); err != nil {
			return fmt.Errorf("invalid JSON for %s: %w", key, err)
		}
		m[key] = parsed
	}
	return nil
}

// stringValueOrNull returns types.StringNull() when s is the empty string,
// otherwise types.StringValue(s). Use for Computed-only attributes so that
// an absent (empty) server response is represented as null state rather than
// the empty string, which avoids a perpetual drift.
func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// normalizedFromRaw converts a raw JSON message into a jsontypes.Normalized
// value, preserving the server's exact byte representation. An empty or nil
// payload becomes a null Normalized. Routing through json.RawMessage avoids
// re-encoding through a map[string]any (which has non-deterministic key order
// in Go) and so prevents spurious plan diffs.
func normalizedFromRaw(raw json.RawMessage) jsontypes.Normalized {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return jsontypes.NewNormalizedNull()
	}
	return jsontypes.NewNormalizedValue(string(trimmed))
}
