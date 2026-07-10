package client

import (
	"time"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
	lifecyclepb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/lifecycle"
	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
)

type frameFactory struct {
	allocator guidarch.GuidAllocator
}

func newFrameFactory(allocator guidarch.GuidAllocator) *frameFactory {
	return &frameFactory{allocator: allocator}
}

func (f *frameFactory) nextGUID() string {
	if f == nil || f.allocator == nil {
		return ""
	}
	return f.allocator.NextGUID().String()
}

func (f *frameFactory) base(clientID int64, sessionGUID string, frameType lifecyclepb.ServiceControlFrameType) *lifecyclepb.ServiceControlFrame {
	return &lifecyclepb.ServiceControlFrame{
		FrameGuid:        f.nextGUID(),
		ClientId:         clientID,
		SessionGuid:      sessionGUID,
		CreateTimeMillis: time.Now().UnixMilli(),
		FrameType:        frameType,
	}
}

func (f *frameFactory) clientMusterFrame(clientID int64, name string, config Config) *lifecyclepb.ServiceControlFrame {
	frame := f.base(clientID, "", lifecyclepb.ServiceControlFrameType_CLIENT_MUSTER)
	attributes := map[string]string{}
	if config.ClientGUID != "" {
		attributes["clientGuid"] = config.ClientGUID
	}
	frame.Body = &lifecyclepb.ServiceControlFrame_ClientMuster{
		ClientMuster: &lifecyclepb.ClientMuster{
			ClientId:         clientID,
			ClientName:       name,
			TransportVersion: config.TransportVersion,
			Attributes:       attributes,
		},
	}
	return frame
}

func (f *frameFactory) registerFrame(clientID int64, sessionGUID string, request servicearch.ServiceJoinRequest) *lifecyclepb.ServiceControlFrame {
	frame := f.base(clientID, sessionGUID, lifecyclepb.ServiceControlFrameType_REGISTER_SERVICE)
	frame.Body = &lifecyclepb.ServiceControlFrame_RegisterService{
		RegisterService: &lifecyclepb.RegisterServiceCommand{
			ServiceGuid:             request.ServiceGUID,
			DeployGuid:              request.DeployGUID,
			EndpointProtocol:        request.Endpoint.Protocol,
			EndpointHost:            request.Endpoint.Host,
			EndpointPort:            request.Endpoint.Port,
			EndpointPath:            request.Endpoint.Path,
			EndpointAddress:         request.Endpoint.Address,
			Version:                 request.Version,
			Zone:                    request.Zone,
			Weight:                  request.Weight,
			Tags:                    request.Tags,
			MetadataJson:            request.MetadataJSON,
			HeartbeatIntervalMillis: request.HeartbeatIntervalMillis,
			LeaseTimeoutMillis:      request.LeaseTimeoutMillis,
			InstanceGuid:            request.InstanceGUID,
			RuntimeNodeId:           request.RuntimeNodeID,
			RuntimeNodeAlias:        request.RuntimeNodeAlias,
			RuntimeNodeMetadataJson: request.RuntimeNodeMetadataJSON,
		},
	}
	return frame
}

func (f *frameFactory) deregisterFrame(clientID int64, sessionGUID string, instruction servicearch.ServiceDeregisterInstruction) *lifecyclepb.ServiceControlFrame {
	frame := f.base(clientID, sessionGUID, lifecyclepb.ServiceControlFrameType_DEREGISTER)
	frame.Body = &lifecyclepb.ServiceControlFrame_Deregister{
		Deregister: &lifecyclepb.DeregisterCommand{
			InstanceGuid: instruction.InstanceGUID,
			Reason:       instruction.Reason,
		},
	}
	return frame
}

func (f *frameFactory) heartbeatFrame(clientID int64, sessionGUID string, instanceGUID string) *lifecyclepb.ServiceControlFrame {
	frame := f.base(clientID, sessionGUID, lifecyclepb.ServiceControlFrameType_HEARTBEAT)
	frame.Body = &lifecyclepb.ServiceControlFrame_Heartbeat{
		Heartbeat: &lifecyclepb.Heartbeat{
			InstanceGuid:     instanceGUID,
			Status:           "ONLINE",
			ClientTimeMillis: time.Now().UnixMilli(),
		},
	}
	return frame
}

func (f *frameFactory) shutdownAcceptedFrame(clientID int64, sessionGUID string, request *lifecyclepb.ServiceControlFrame) *lifecyclepb.ServiceControlFrame {
	frame := f.base(clientID, sessionGUID, lifecyclepb.ServiceControlFrameType_SHUTDOWN_ACCEPTED)
	if request != nil {
		frame.CorrelationGuid = request.FrameGuid
	}
	var instanceGUID, reason string
	if request != nil && request.GetShutdownService() != nil {
		instanceGUID = request.GetShutdownService().InstanceGuid
		reason = request.GetShutdownService().Reason
	}
	frame.Body = &lifecyclepb.ServiceControlFrame_ShutdownAccepted{
		ShutdownAccepted: &lifecyclepb.ShutdownServiceAccepted{
			InstanceGuid: instanceGUID,
			Reason:       reason,
		},
	}
	return frame
}
