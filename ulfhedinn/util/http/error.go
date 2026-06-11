package http

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"os"

	"github.com/NutsFoundation/pinecone-lupus/ulfhedinn/util/cli"
)

func classifyError(err error) int {
	if err == nil {
		return cli.ExitSuccess
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) {
		return cli.ExitTimeout
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return cli.ExitTimeout
		}
		err = urlErr.Err
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return cli.ExitResolveFailure
	}

	var tlsErr tls.RecordHeaderError
	if errors.As(err, &tlsErr) {
		return cli.ExitTLSFailure
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return cli.ExitConnectFailure
		}
		return cli.ExitNetworkFailure
	}

	return cli.ExitGenericError
}
