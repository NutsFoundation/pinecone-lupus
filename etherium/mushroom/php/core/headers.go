package core

import (
	"net/http"
	"strings"
)

func copySafeHeaders(source http.Header) http.Header {
	result := http.Header{}
	for key, values := range source {
		normalized := http.CanonicalHeaderKey(key)
		switch strings.ToLower(normalized) {
		case "content-type", "cache-control", "expires", "pragma", "location", "set-cookie":
			for _, value := range values {
				result.Add(normalized, value)
			}
		}
	}
	return result
}
