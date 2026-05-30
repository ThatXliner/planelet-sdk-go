package planelet

import (
	"encoding/base64"
	"strings"
)

// DecodeRawBody decodes a base64-encoded webhook body into raw bytes.
func DecodeRawBody(rawBodyBase64 string) []byte {
	data, err := base64.StdEncoding.DecodeString(rawBodyBase64)
	if err != nil {
		return nil
	}
	return data
}

// DecodeRawBodyText decodes a base64-encoded webhook body into a UTF-8 string.
func DecodeRawBodyText(rawBodyBase64 string) string {
	return string(DecodeRawBody(rawBodyBase64))
}

// FirstHeader returns the first value for the given header name (case-insensitive),
// or an empty string if the header is not present.
func FirstHeader(headers map[string][]string, name string) string {
	lower := strings.ToLower(name)
	for k, vals := range headers {
		if strings.ToLower(k) == lower && len(vals) > 0 {
			return vals[0]
		}
	}
	return ""
}
