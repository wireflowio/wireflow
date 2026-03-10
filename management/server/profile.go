package server

import (
	"wireflow/internal/infra"
	"wireflow/management/server/middleware"

	"github.com/gin-gonic/gin"
)

func (s *Server) profileRouter() {
	profileApi := s.Group("/api/v1/profile")
	//userApi.Use(dex.AuthMiddleware())
	{
		profileApi.POST("/getProfile", middleware.AuthMiddleware(), s.getProfile())
		profileApi.POST("/updateProfile", middleware.AuthMiddleware(), s.updateProfile())
	}
}

func (s *Server) getProfile() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Request.Context().Value(infra.UserIDKey).(string)
		s.profileController.GetProfile(c.Request.Context(), userId)
	}
}

func (s *Server) updateProfile() gin.HandlerFunc {
	return func(c *gin.Context) {}
}
