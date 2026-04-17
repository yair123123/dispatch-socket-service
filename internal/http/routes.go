package http

import (
	"dispatch-socket-service/internal/http/handlers"
	"dispatch-socket-service/internal/ws"

	"github.com/gin-gonic/gin"
)

func NewRouter(health *handlers.HealthHandler, dispatch *handlers.InternalDispatchHandler, wsHandler *ws.DriverWSHandler, internalSecret string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", health.Get)
	internal := r.Group("/internal")
	internal.Use(InternalAuthMiddleware(internalSecret))
	internal.POST("/dispatch/offer", dispatch.SendOffer)
	internal.POST("/dispatch/cancel", dispatch.CancelOffer)
	internal.POST("/dispatch/start-round", dispatch.StartRound)
	if wsHandler != nil {
		r.GET("/ws/drivers/connect", gin.WrapH(wsHandler))
	}
	return r
}
