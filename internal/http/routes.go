package http

import (
	"dispatch-socket-service/internal/http/handlers"
	"dispatch-socket-service/internal/ws"

	"github.com/gin-gonic/gin"
)

func NewRouter(health *handlers.HealthHandler, dispatch *handlers.InternalDispatchHandler, wsHandler *ws.DriverWSHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/health", health.Get)
	r.POST("/internal/dispatch/offer", dispatch.SendOffer)
	r.POST("/internal/dispatch/cancel", dispatch.CancelOffer)
	r.GET("/ws/drivers/connect", gin.WrapH(wsHandler))
	return r
}
