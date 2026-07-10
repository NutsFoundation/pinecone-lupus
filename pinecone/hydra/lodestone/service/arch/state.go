package arch

type ServiceLegionaryState string

const (
	ServiceLegionaryStateNew          ServiceLegionaryState = "New"
	ServiceLegionaryStateStarting     ServiceLegionaryState = "Starting"
	ServiceLegionaryStateControlReady ServiceLegionaryState = "ControlReady"
	ServiceLegionaryStateJoining      ServiceLegionaryState = "Joining"
	ServiceLegionaryStateOnline       ServiceLegionaryState = "Online"
	ServiceLegionaryStateRejoining    ServiceLegionaryState = "Rejoining"
	ServiceLegionaryStateOffline      ServiceLegionaryState = "Offline"
	ServiceLegionaryStateTerminated   ServiceLegionaryState = "Terminated"
)
