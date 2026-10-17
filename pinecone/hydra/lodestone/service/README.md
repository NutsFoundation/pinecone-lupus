# RedQueen Service Registration

The service provider accepts `ServiceLodestoneConfig` and uses the existing
RedQueen metadata and service-control gRPC protocol. It does not create another
service process or require CGO.

## Registration Data

- `Endpoint` remains supported. `AdvertisedEndpoint` is a compatible alias for
  the application's published endpoint, separate from the registry RPC target.
- Either endpoint field may be used alone. When both are present they must agree
  after comparison normalization; conflicts fail instead of silently choosing.
  The selected endpoint's protocol casing and address are preserved.
- `Version` and valid `MetadataJSON` are forwarded to the instance registration.
  They do not replace `RuntimeNodeMetadata` or `RuntimeNodeNetwork`.
- Endpoint-free SDK configurations remain supported. Applications may impose
  stronger requirements, such as Ethercyber's reachable advertised endpoint.

## Transport Policy

Configure exactly one of `RPC.TLS.Enable` and `RPC.Insecure`:

| TLS.Enable | Insecure | Result |
| --- | --- | --- |
| true | false | TLS with normal certificate verification |
| false | true | Explicit plaintext gRPC |
| false | false | Configuration error |
| true | true | Configuration error |

TLS supports `ServerName`, `CAPath`, `CertificatePath`, and `PrivateKeyPath`, with
TLS 1.2 as the minimum. Client certificate and private key must be supplied
together. TLS options while TLS is disabled are errors. Filesystem paths are
interpreted by the process opening them; absolute paths avoid working-directory
ambiguity. Certificate failures never downgrade to plaintext. No certificate
verification bypass is provided.

Migration: earlier releases always dialed plaintext and ignored `RPC.Insecure`.
Callers that relied on that behavior must explicitly set `Insecure: true` or
configure TLS. The low-level `redqueen/client.Config` follows the same policy.
Public constructor signatures and the existing protocol remain unchanged.

## Connection Ownership

EOF and transport failures clear the affected stream and finish its pending
requests before notifying `OFFLINE`. Cleanup is bound to the stream, so a delayed
failure from an old stream cannot terminate a replacement connection. Business
error frames fail their correlated request without disconnecting the transport.

`AutoReconnect: false` suppresses background SDK reconnects, not disconnect
notifications or cleanup. Ethercyber uses this mode and owns its retry loop.
Explicit startup or on-demand recovery remains available. Termination cancels
pending connection work and requests; an in-flight dial cannot restore a stopped
client. A caller's short startup timeout does not become the lifetime of a
successfully established control stream.

Tests use fake streams and ephemeral local gRPC listeners with generated test
certificates. They do not require a running RedQueen registry.
