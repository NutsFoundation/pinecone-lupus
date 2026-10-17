package arch

import "context"

type LifecyclePolicy string

const (
	LifecyclePolicyManual     LifecyclePolicy = "Manual"
	LifecyclePolicyAutoCreate LifecyclePolicy = "AutoCreate"
	LifecyclePolicyAutoStart  LifecyclePolicy = "AutoStart"
)

type ServiceLodestone interface {
	Name() string
	LifecyclePolicy() LifecyclePolicy
	Legionary() CollectiveServiceLegionary
	SetLegionary(legionary CollectiveServiceLegionary)
	ServiceClient() ServiceClient
	ServiceIdentifier() string
	Declaration() *ServiceDeclaration
	Start(ctx context.Context) error
	Stop(ctx context.Context, reason string) error
}

type ServiceLodestoneProvider interface {
	ProviderName() string
	Create(ctx context.Context, config ServiceLodestoneConfig) (ServiceLodestone, error)
}

type ServiceLodestoneConfig struct {
	Enabled             bool
	Provider            string
	Lifecycle           LifecyclePolicy
	Name                string
	MessageNode         string
	Logger              Logger
	RPC                 RPCConfig
	ServiceIdentifier   string
	RuntimeNodeID       string
	RuntimeNodeAlias    string
	RuntimeNodeMetadata any
	RuntimeNodeNetwork  *NetworkMetadata
	Endpoint            ServiceEndpoint
	AdvertisedEndpoint  ServiceEndpoint
	Version             string
	MetadataJSON        string
	Declaration         ServiceDeclaration
}

type RPCConfig struct {
	Type                        string
	Host                        string
	Port                        int
	Insecure                    bool
	TLS                         RPCTLSConfig
	ClientID                    *int64
	ClientGuid                  string
	AutoReconnect               *bool
	KeepAliveTimeSec            int64
	KeepAliveTimeoutSec         int64
	EnableHeartbeat             bool
	HeartbeatIntervalMillis     int64
	CommandTimeoutMillis        int64
	ControlSyncTimeoutMillis    int64
	InitialConnectTimeoutMillis int64
}

type RPCTLSConfig struct {
	Enable          bool
	ServerName      string
	CAPath          string
	CertificatePath string
	PrivateKeyPath  string
}
