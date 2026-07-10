package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
	conductcore "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/conduct/core"
	servicecore "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/core"
	redclient "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/client"
	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
	guidcore "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/core"
)

const ProviderName = "RedQueen"

type RedQueenServiceProvider struct {
	allocator   guidarch.GuidAllocator
	allocator64 guidarch.GuidAllocator64
}

var _ servicearch.ServiceLodestoneProvider = (*RedQueenServiceProvider)(nil)

func NewRedQueenServiceProvider(allocator guidarch.GuidAllocator, allocator64 ...guidarch.GuidAllocator64) *RedQueenServiceProvider {
	if allocator == nil {
		allocator = guidcore.NewUUIDV7GuidAllocator()
	}
	var allocator64Value guidarch.GuidAllocator64
	if len(allocator64) > 0 {
		allocator64Value = allocator64[0]
	}
	if allocator64Value == nil {
		allocator64Value = guidcore.NewDefaultGuid64Allocator(guidcore.NewRandomWorkerIdAssigner())
	}
	return &RedQueenServiceProvider{
		allocator:   allocator,
		allocator64: allocator64Value,
	}
}

func (p *RedQueenServiceProvider) ProviderName() string {
	return ProviderName
}

func (p *RedQueenServiceProvider) Create(ctx context.Context, config servicearch.ServiceLodestoneConfig) (servicearch.ServiceLodestone, error) {
	if strings.TrimSpace(config.ServiceIdentifier) == "" {
		return nil, errors.New("Lodestone.Service.serviceIdentifier is required")
	}
	clientID, err := servicecore.ResolveClientID(config.RPC.ClientID, p.allocator64)
	if err != nil {
		return nil, err
	}
	clientName := resolveClientName(config)
	client := redclient.NewServiceClient(redclient.Config{
		Address:                rpcAddress(config.RPC),
		ClientID:               clientID,
		ClientName:             clientName,
		ClientGUID:             config.RPC.ClientGuid,
		CommandTimeout:         millis(config.RPC.CommandTimeoutMillis),
		ControlSyncTimeout:     millis(config.RPC.ControlSyncTimeoutMillis),
		HeartbeatInterval:      millis(config.RPC.HeartbeatIntervalMillis),
		EnableHeartbeat:        config.RPC.EnableHeartbeat,
		AutoSynchronizeOnError: true,
	}, p.allocator)

	serviceMeta, err := p.resolveServiceMeta(ctx, client, config)
	if err != nil {
		return nil, err
	}

	joinRequest := servicearch.ServiceJoinRequest{
		ServiceGUID: serviceMeta.GUID,
	}
	runtimeNodeID := strings.TrimSpace(config.RuntimeNodeID)
	if runtimeNodeID != "" {
		joinRequest.RuntimeNodeID = runtimeNodeID
		joinRequest.RuntimeNodeAlias = config.RuntimeNodeAlias
		joinRequest.RuntimeNodeMetadataJSON = runtimeNodeMetadataJSON(config.RuntimeNodeMetadata)
	}
	legionary := conductcore.NewCollectiveServiceLegionary(config.Name, client, joinRequest)
	declaration := config.Declaration
	lodestone := servicecore.NewGenericServiceLodestone(
		config.Name,
		config.Lifecycle,
		client,
		config.ServiceIdentifier,
		&declaration,
	)
	lodestone.SetLegionary(legionary)
	return lodestone, nil
}

func (p *RedQueenServiceProvider) resolveServiceMeta(ctx context.Context, client servicearch.ServiceClient, config servicearch.ServiceLodestoneConfig) (*servicearch.ServiceMeta, error) {
	metaPort := client.Meta()
	identifier := normalizeServicePath(config.ServiceIdentifier)
	var meta *servicearch.ServiceMeta
	var err error
	if strings.Contains(identifier, "/") {
		meta, err = metaPort.QueryServiceMetaByPath(ctx, identifier)
	} else {
		meta, err = metaPort.QueryServiceMetaByGUID(ctx, identifier)
	}
	if err != nil {
		return nil, err
	}
	if meta != nil && meta.GUID != "" {
		return meta, nil
	}
	if !config.Declaration.Enabled {
		return nil, errors.New("Lodestone.Service.serviceIdentifier not found and declaration is disabled")
	}
	declarationPath := normalizeServicePath(config.Declaration.Path)
	if identifier != declarationPath {
		return nil, errors.New("Lodestone.Service.declaration.path must equal serviceIdentifier")
	}
	parentPath := parentPathOf(declarationPath)
	if parentPath == "" {
		return nil, errors.New("Lodestone.Service.declaration.path requires a parent application path")
	}
	createdGUID, err := metaPort.CreateNewService(ctx, parentPath, declarationToMeta(config.Declaration))
	if err != nil {
		return nil, err
	}
	if createdGUID != "" {
		meta, err = metaPort.QueryServiceMetaByGUID(ctx, createdGUID)
		if err != nil {
			return nil, err
		}
		if meta != nil && meta.GUID != "" {
			return meta, nil
		}
	}
	return metaPort.QueryServiceMetaByPath(ctx, declarationPath)
}

func declarationToMeta(declaration servicearch.ServiceDeclaration) servicearch.ServiceMeta {
	return servicearch.ServiceMeta{
		Name:             leafNameOf(declaration.Path),
		Type:             declaration.Type,
		Alias:            declaration.Alias,
		ResourceType:     declaration.ResourceType,
		ServiceType:      declaration.ServiceType,
		DisplayName:      declaration.DisplayName,
		Description:      declaration.Description,
		FullName:         declaration.FullName,
		GroupNamespace:   declaration.GroupNamespace,
		GroupName:        declaration.GroupName,
		Scenario:         declaration.Scenario,
		PrimaryImplLang:  declaration.PrimaryImplLang,
		ExtraInformation: declaration.ExtraInformation,
		Level:            declaration.Level,
	}
}

func normalizeServicePath(raw string) string {
	return strings.TrimSpace(raw)
}

func parentPathOf(raw string) string {
	clean := path.Clean(raw)
	if clean == "." || clean == "/" || !strings.HasPrefix(clean, "/") {
		return ""
	}
	parent := path.Dir(clean)
	if parent == "." || parent == "/" {
		return ""
	}
	return parent
}

func leafNameOf(raw string) string {
	clean := path.Clean(raw)
	if clean == "." || clean == "/" {
		return ""
	}
	return path.Base(clean)
}

func millis(value int64) time.Duration {
	if value <= 0 {
		return 0
	}
	return time.Duration(value) * time.Millisecond
}

func rpcAddress(config servicearch.RPCConfig) string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}

func resolveClientName(config servicearch.ServiceLodestoneConfig) string {
	if config.Name != "" {
		return config.Name
	}
	return config.ServiceIdentifier
}

func runtimeNodeMetadataJSON(metadata any) string {
	if metadata == nil {
		return ""
	}
	if text, ok := metadata.(string); ok {
		return strings.TrimSpace(text)
	}
	bytes, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Sprint(metadata)
	}
	return string(bytes)
}
