package arch

type ServiceEndpoint struct {
	Protocol string
	Host     string
	Port     int32
	Path     string
	Address  string
}

type ServiceDeclaration struct {
	Enabled          bool
	Path             string
	Name             string
	Type             string
	Alias            string
	ResourceType     string
	ServiceType      string
	DisplayName      string
	Description      string
	FullName         string
	GroupNamespace   string
	GroupName        string
	Scenario         string
	PrimaryImplLang  string
	ExtraInformation string
	Level            string
}

type ServiceMeta struct {
	GUID             string
	Name             string
	Type             string
	Alias            string
	ResourceType     string
	ServiceType      string
	DisplayName      string
	Description      string
	FullName         string
	GroupNamespace   string
	GroupName        string
	Scenario         string
	PrimaryImplLang  string
	ExtraInformation string
	Level            string
}

type ServiceJoinRequest struct {
	ClientID                int64
	ServiceGUID             string
	DeployGUID              string
	InstanceGUID            string
	Endpoint                ServiceEndpoint
	Version                 string
	Zone                    string
	Weight                  int32
	Tags                    []string
	MetadataJSON            string
	HeartbeatIntervalMillis int64
	LeaseTimeoutMillis      int64
	RuntimeNodeID           string
	RuntimeNodeAlias        string
	RuntimeNodeMetadataJSON string
}

type ServiceJoinResponse struct {
	ServiceGUID        string
	InstanceGUID       string
	SessionGUID        string
	Status             string
	RegisterTimeMillis int64
	ExpireTimeMillis   int64
}

type ServiceDeregisterInstruction struct {
	InstanceGUID string
	Reason       string
}

type ServiceDeregisterResult struct {
	InstanceGUID string
	Status       string
	Reason       string
}

type ServiceShutdownInstruction struct {
	InstanceGUID string
	Reason       string
}
