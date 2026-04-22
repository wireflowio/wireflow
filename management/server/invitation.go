package server

import (
	"wireflow/management/dto"
	"wireflow/management/server/middleware"
	"wireflow/pkg/utils/resp"

	"github.com/gin-gonic/gin"
)

func (s *Server) invitationRouter() {
	g := s.Group("/api/v1/workspaces/:id/invitations")
	g.Use(middleware.AuthMiddleware())
	{
		g.POST("", s.handleCreateInvitation())
		g.GET("", s.handleListInvitations())
		g.DELETE("/:invID", s.handleRevokeInvitation())
	}

	// Accept is a public-but-authenticated endpoint (no workspace membership required).
	acceptGroup := s.Group("/api/v1/invite")
	acceptGroup.Use(middleware.AuthMiddleware())
	{
		acceptGroup.POST("/:token/accept", s.handleAcceptInvitation())
	}
}

func (s *Server) handleCreateInvitation() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.Param("id")
		inviterID := c.GetString("user_id")

		var req struct {
			Email string            `json:"email" binding:"required,email"`
			Role  dto.WorkspaceRole `json:"role"  binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.BadRequest(c, err.Error())
			return
		}

		inv, err := s.invitationController.Create(c.Request.Context(), wsID, inviterID, req.Email, req.Role)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, inv)
	}
}

func (s *Server) handleListInvitations() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.Param("id")
		invs, err := s.invitationController.List(c.Request.Context(), wsID)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, invs)
	}
}

func (s *Server) handleRevokeInvitation() gin.HandlerFunc {
	return func(c *gin.Context) {
		invID := c.Param("invID")
		if err := s.invitationController.Revoke(c.Request.Context(), invID); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}

func (s *Server) handleAcceptInvitation() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		userID := c.GetString("user_id")
		if err := s.invitationController.Accept(c.Request.Context(), token, userID); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}
