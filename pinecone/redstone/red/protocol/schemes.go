package protocol

const (
	SchemeRed  = "red"
	SchemeUOFS = "uofs"
	SchemeS3   = "s3"

	HTTPScheme        = "http"
	HTTPSSecureScheme = "https"
)

func IsSupportedScheme(scheme string) bool {
	return scheme == SchemeRed || scheme == SchemeUOFS || scheme == SchemeS3
}
