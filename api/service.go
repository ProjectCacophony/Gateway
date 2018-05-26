package api

import (
	"net/http"

	muxtrace "github.com/DataDog/dd-trace-go/contrib/gorilla/mux"
	"github.com/json-iterator/go"
	"gitlab.com/Cacophony/Gateway/metrics"
	"gitlab.com/Cacophony/dhelpers"
	"gitlab.com/Cacophony/dhelpers/apihelper"
)

// New creates a new restful Web Service for reporting information about the worker
func New() *muxtrace.Router {
	mux := muxtrace.NewRouter(muxtrace.WithServiceName("Gateway-API"))

	mux.HandleFunc("/stats", getStats)

	return mux
}

func getStats(w http.ResponseWriter, _ *http.Request) {
	// gather data
	var result apihelper.GatewayStatus
	result.Service = apihelper.GenerateServiceInformation()
	result.Events = apihelper.GatewayEventInformation{
		EventsDiscarded:                metrics.EventsDiscarded.Value(),
		EventsGuildCreate:              metrics.EventsGuildCreate.Value(),
		EventsGuildUpdate:              metrics.EventsGuildUpdate.Value(),
		EventsGuildDelete:              metrics.EventsGuildDelete.Value(),
		EventsGuildMemberAdd:           metrics.EventsGuildMemberAdd.Value(),
		EventsGuildMemberUpdate:        metrics.EventsGuildMemberUpdate.Value(),
		EventsGuildMemberRemove:        metrics.EventsGuildMemberRemove.Value(),
		EventsGuildMembersChunk:        metrics.EventsGuildMembersChunk.Value(),
		EventsGuildRoleCreate:          metrics.EventsGuildRoleCreate.Value(),
		EventsGuildRoleUpdate:          metrics.EventsGuildRoleUpdate.Value(),
		EventsGuildRoleDelete:          metrics.EventsGuildRoleDelete.Value(),
		EventsGuildEmojisUpdate:        metrics.EventsGuildEmojisUpdate.Value(),
		EventsChannelCreate:            metrics.EventsChannelCreate.Value(),
		EventsChannelUpdate:            metrics.EventsChannelUpdate.Value(),
		EventsChannelDelete:            metrics.EventsChannelDelete.Value(),
		EventsMessageCreate:            metrics.EventsMessageCreate.Value(),
		EventsMessageUpdate:            metrics.EventsMessageUpdate.Value(),
		EventsMessageDelete:            metrics.EventsMessageDelete.Value(),
		EventsPresenceUpdate:           metrics.EventsPresenceUpdate.Value(),
		EventsChannelPinsUpdate:        metrics.EventsChannelPinsUpdate.Value(),
		EventsGuildBanAdd:              metrics.EventsGuildBanAdd.Value(),
		EventsGuildBanRemove:           metrics.EventsGuildBanRemove.Value(),
		EventsMessageReactionAdd:       metrics.EventsMessageReactionAdd.Value(),
		EventsMessageReactionRemove:    metrics.EventsMessageReactionRemove.Value(),
		EventsMessageReactionRemoveAll: metrics.EventsMessageReactionRemoveAll.Value(),
	}
	result.Available = true

	// return result
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := jsoniter.NewEncoder(w).Encode(result)
	dhelpers.LogError(err)
}
