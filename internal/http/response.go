package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

func OK(c *gin.Context, message string, data any, meta any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Message: message, Data: data, Meta: meta})
}

func Created(c *gin.Context, message string, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Message: message, Data: data})
}

func Error(c *gin.Context, status int, message string, errors any) {
	c.JSON(status, Envelope{Success: false, Message: message, Errors: errors})
}
