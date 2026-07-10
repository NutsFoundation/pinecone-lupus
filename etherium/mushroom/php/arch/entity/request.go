package entity

import "net/http"

type MushroomRequest struct {
	Method     string
	URI        string
	Path       string
	RawQuery   string
	Host       string
	Scheme     string
	RemoteAddr string
	Header     http.Header
	Body       []byte
}
