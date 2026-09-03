// Package api exposes the HTTP surface of the notification service:
// creating notifications, checking their status, and managing templates.
package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/wwoes/notification-service/internal/notification"
	"github.com/wwoes/notification-service/internal/store"
)

// Handler groups the HTTP handlers and their dependencies.
type Handler struct {
	notifications *notification.Service
	templates     *store.TemplateStore
}

func NewHandler(notifications *notification.Service, templates *store.TemplateStore) *Handler {
	return &Handler{notifications: notifications, templates: templates}
}

// Register attaches all routes to rg (e.g. router.Group("/api/v1")).
func (h *Handler) Register(rg *gin.RouterGroup) {
	rg.POST("/notifications", h.createNotification)
	rg.GET("/notifications/:id", h.getNotification)
	rg.POST("/templates", h.createTemplate)
	rg.GET("/templates", h.listTemplates)
}

type createNotificationRequest struct {
	TelegramID int64                  `json:"telegram_id" binding:"required"`
	Template   string                 `json:"template" binding:"required"`
	Params     map[string]interface{} `json:"params"`
}

// POST /api/v1/notifications
// Registers (if needed) the target Telegram user, renders the named
// template with params, and queues it for delivery.
func (h *Handler) createNotification(c *gin.Context) {
	var req createNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	n, err := h.notifications.CreateNotification(c.Request.Context(), notification.CreateInput{
		TelegramID: req.TelegramID,
		Template:   req.Template,
		Params:     req.Params,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create notification"})
		return
	}

	c.JSON(http.StatusAccepted, n)
}

// GET /api/v1/notifications/:id
func (h *Handler) getNotification(c *gin.Context) {
	n, err := h.notifications.GetNotification(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "notification not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load notification"})
		return
	}
	c.JSON(http.StatusOK, n)
}

type createTemplateRequest struct {
	Name string `json:"name" binding:"required"`
	Body string `json:"body" binding:"required"`
}

// POST /api/v1/templates
func (h *Handler) createTemplate(c *gin.Context) {
	var req createTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	t, err := h.templates.Create(c.Request.Context(), req.Name, req.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create template"})
		return
	}
	c.JSON(http.StatusCreated, t)
}

// GET /api/v1/templates
func (h *Handler) listTemplates(c *gin.Context) {
	templates, err := h.templates.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list templates"})
		return
	}
	c.JSON(http.StatusOK, templates)
}
