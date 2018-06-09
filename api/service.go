package api

import (
	"net/http"

	"github.com/Seklfreak/Robyul2/helpers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"gitlab.com/Cacophony/Gateway/metrics"
	"gitlab.com/Cacophony/dhelpers/apihelper"
	"gitlab.com/Cacophony/dhelpers/cache"
)

// New creates a new restful Web Service for reporting information about the worker
func New() http.Handler {
	router := chi.NewRouter()

	// setup middleware
	router.Use(middleware.Recoverer)
	middleware.DefaultLogger = middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: cache.GetLogger(), NoColor: false})
	router.Use(middleware.Logger)
	router.Use(middleware.DefaultCompress)
	router.Use(render.SetContentType(render.ContentTypeJSON))

	router.HandleFunc("/stats", getStats)

	return router
}

func getStats(w http.ResponseWriter, r *http.Request) {
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
	err := render.Render(w, r, result)
	helpers.RelaxLog(err)
}
