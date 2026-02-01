package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)


func AuthMiddleware(next http.Handler) gin.HandlerFunc {
	return nil
}
