package handlers

import (
	"net/http"

	"dispatch-socket-service/internal/models"
	"dispatch-socket-service/internal/services"

	"github.com/gin-gonic/gin"
)

type InternalDispatchHandler struct {
	offers *services.OfferDeliveryService
}

func NewInternalDispatchHandler(offers *services.OfferDeliveryService) *InternalDispatchHandler {
	return &InternalDispatchHandler{offers: offers}
}

func (h *InternalDispatchHandler) SendOffer(c *gin.Context) {
	var req models.SendOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.offers.DeliverOfferBatch(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *InternalDispatchHandler) CancelOffer(c *gin.Context) {
	var req models.CancelOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.offers.CancelOffer(c.Request.Context(), req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
