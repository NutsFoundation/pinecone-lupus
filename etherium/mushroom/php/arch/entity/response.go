package entity

import "net/http"

type MushroomResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}
