package server

import (
	"wireflow/management/dto"
	"wireflow/management/server/middleware"
	"wireflow/pkg/utils/resp"

	"github.com/gin-gonic/gin"
)

func (s *Server) memberRouter() {
	g := s.Group("/api/v1/workspaces/:id/members")
	g.Use(middleware.AuthMiddleware())
	{
		g.GET("", s.handleListMembers())
		g.PUT("/:userID", s.handleUpdateMemberRole())
		g.DELETE("/:userID", s.handleRemoveMember())
	}
}

func (s *Server) handleListMembers() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.Param("id")
		members, err := s.memberController.List(c.Request.Context(), wsID)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, members)
	}
}

func (s *Server) handleUpdateMemberRole() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.Param("id")
		userID := c.Param("userID")

		var req struct {
			Role dto.WorkspaceRole `json:"role" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.BadRequest(c, err.Error())
			return
		}

		if err := s.memberController.UpdateRole(c.Request.Context(), wsID, userID, req.Role); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}

func (s *Server) handleRemoveMember() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.Param("id")
		userID := c.Param("userID")

		if err := s.memberController.Remove(c.Request.Context(), wsID, userID); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}
