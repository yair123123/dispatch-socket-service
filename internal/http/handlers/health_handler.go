package handlers

import "github.com/gin-gonic/gin"

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler { return &HealthHandler{} }

func (h *HealthHandler) Get(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
