package client

import (
	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
	lifecyclepb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/lifecycle"
	metapb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/meta"
)

func toJoinResponse(accepted *lifecyclepb.RegisterServiceAccepted) *servicearch.ServiceJoinResponse {
	if accepted == nil {
		return nil
	}
	return &servicearch.ServiceJoinResponse{
		ServiceGUID:        accepted.ServiceGuid,
		InstanceGUID:       accepted.InstanceGuid,
		SessionGUID:        accepted.SessionGuid,
		Status:             accepted.Status,
		RegisterTimeMillis: accepted.RegisterTimeMillis,
		ExpireTimeMillis:   accepted.ExpireTimeMillis,
	}
}

func toDeregisterResult(instruction servicearch.ServiceDeregisterInstruction, deregistered *lifecyclepb.Deregistered) *servicearch.ServiceDeregisterResult {
	if deregistered == nil {
		return &servicearch.ServiceDeregisterResult{
			InstanceGUID: instruction.InstanceGUID,
			Status:       "DISCONNECTED",
			Reason:       instruction.Reason,
		}
	}
	return &servicearch.ServiceDeregisterResult{
		InstanceGUID: deregistered.InstanceGuid,
		Status:       deregistered.Status,
		Reason:       deregistered.Reason,
	}
}

func toShutdownInstruction(command *lifecyclepb.ShutdownServiceCommand) servicearch.ServiceShutdownInstruction {
	if command == nil {
		return servicearch.ServiceShutdownInstruction{}
	}
	return servicearch.ServiceShutdownInstruction{
		InstanceGUID: command.InstanceGuid,
		Reason:       command.Reason,
	}
}

func toServiceMetaDTO(meta *metapb.ServiceMetaDTO) *servicearch.ServiceMeta {
	if meta == nil {
		return nil
	}
	return &servicearch.ServiceMeta{
		GUID:             meta.Guid,
		Name:             meta.Name,
		Type:             meta.Type,
		Alias:            meta.Alias,
		ResourceType:     meta.ResourceType,
		ServiceType:      meta.ServiceType,
		DisplayName:      meta.DisplayName,
		Description:      meta.Description,
		FullName:         meta.FullName,
		GroupNamespace:   meta.GroupNamespace,
		GroupName:        meta.GroupName,
		Scenario:         meta.Scenario,
		PrimaryImplLang:  meta.PrimaryImplLang,
		ExtraInformation: meta.ExtraInformation,
		Level:            meta.Level,
	}
}

func toProtocolMeta(meta servicearch.ServiceMeta) *metapb.ServiceMetaDTO {
	return &metapb.ServiceMetaDTO{
		Guid:             meta.GUID,
		Name:             meta.Name,
		Type:             meta.Type,
		Alias:            meta.Alias,
		ResourceType:     meta.ResourceType,
		ServiceType:      meta.ServiceType,
		DisplayName:      meta.DisplayName,
		Description:      meta.Description,
		FullName:         meta.FullName,
		GroupNamespace:   meta.GroupNamespace,
		GroupName:        meta.GroupName,
		Scenario:         meta.Scenario,
		PrimaryImplLang:  meta.PrimaryImplLang,
		ExtraInformation: meta.ExtraInformation,
		Level:            meta.Level,
	}
}
