package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	api "github.com/raufendro/novacore/internal/http"
	"github.com/raufendro/novacore/pkg/validation"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	if errors := validation.Struct(req); errors != nil {
		api.Error(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	resp, err := h.service.Register(req)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err.Error(), nil)
		return
	}
	api.Created(c, "Registered successfully", resp)
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	if errors := validation.Struct(req); errors != nil {
		api.Error(c, http.StatusUnprocessableEntity, "Validation failed", errors)
		return
	}
	resp, err := h.service.Login(req)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	api.OK(c, "Logged in successfully", resp, nil)
}

func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, "Invalid request body", err.Error())
		return
	}
	resp, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		api.Error(c, http.StatusUnauthorized, err.Error(), nil)
		return
	}
	api.OK(c, "Token refreshed successfully", resp, nil)
}

func (h *Handler) Me(c *gin.Context) {
	idAny, ok := c.Get("user_id")
	if !ok {
		api.Error(c, http.StatusUnauthorized, "Unauthorized", nil)
		return
	}
	user, err := h.service.Current(idAny.(uint))
	if err != nil || user == nil {
		api.Error(c, http.StatusNotFound, "User not found", nil)
		return
	}
	api.OK(c, "Current user retrieved successfully", user, nil)
}

func (h *Handler) Logout(c *gin.Context) {
	api.OK(c, "Logged out successfully", gin.H{"revoked": false}, nil)
}
