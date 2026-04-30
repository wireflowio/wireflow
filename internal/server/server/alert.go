package server

import (
	"github.com/alatticeio/lattice/internal/server/server/middleware"
	"github.com/alatticeio/lattice/internal/server/service"
	"github.com/alatticeio/lattice/pkg/utils/resp"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (s *Server) alertRouter() {
	r := s.Group("/api/v1/alerts")
	r.Use(middleware.AuthMiddleware())
	{
		// Alert rules
		r.GET("/rules", s.tenantMiddleware.Handle(), s.listAlertRules())
		r.GET("/rules/:id", s.tenantMiddleware.Handle(), s.getAlertRule())
		r.POST("/rules", s.tenantMiddleware.Handle(), s.createAlertRule())
		r.PUT("/rules/:id", s.tenantMiddleware.Handle(), s.updateAlertRule())
		r.DELETE("/rules/:id", s.tenantMiddleware.Handle(), s.deleteAlertRule())

		// Alert history
		r.GET("/history", s.tenantMiddleware.Handle(), s.listAlertHistory())

		// Alert channels
		r.GET("/channels", s.tenantMiddleware.Handle(), s.listAlertChannels())
		r.POST("/channels", s.tenantMiddleware.Handle(), s.createAlertChannel())
		r.PUT("/channels/:id", s.tenantMiddleware.Handle(), s.updateAlertChannel())
		r.DELETE("/channels/:id", s.tenantMiddleware.Handle(), s.deleteAlertChannel())

		// Alert silences
		r.GET("/silences", s.tenantMiddleware.Handle(), s.listAlertSilences())
		r.POST("/silences", s.tenantMiddleware.Handle(), s.createAlertSilence())
		r.DELETE("/silences/:id", s.tenantMiddleware.Handle(), s.deleteAlertSilence())
	}
}

func (s *Server) listAlertRules() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		data, err := s.alertController.ListRules(c.Request.Context(), wsID)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) getAlertRule() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		data, err := s.alertController.GetRule(c.Request.Context(), id)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) createAlertRule() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		var req service.CreateAlertRuleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.Error(c, err.Error())
			return
		}
		data, err := s.alertController.CreateRule(c.Request.Context(), wsID, req)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) updateAlertRule() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.CreateAlertRuleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.Error(c, err.Error())
			return
		}
		data, err := s.alertController.UpdateRule(c.Request.Context(), id, req)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) deleteAlertRule() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := s.alertController.DeleteRule(c.Request.Context(), id); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}

func (s *Server) listAlertHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		items, total, err := s.alertController.ListHistory(c.Request.Context(), wsID, page, pageSize)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, gin.H{
			"items": items,
			"total": total,
			"page":  page,
			"size":  pageSize,
		})
	}
}

func (s *Server) listAlertChannels() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		data, err := s.alertController.ListChannels(c.Request.Context(), wsID)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) createAlertChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		var req service.CreateChannelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.Error(c, err.Error())
			return
		}
		data, err := s.alertController.CreateChannel(c.Request.Context(), wsID, req)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) updateAlertChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req service.CreateChannelRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.Error(c, err.Error())
			return
		}
		data, err := s.alertController.UpdateChannel(c.Request.Context(), id, req)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) deleteAlertChannel() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := s.alertController.DeleteChannel(c.Request.Context(), id); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}

func (s *Server) listAlertSilences() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		data, err := s.alertController.ListSilences(c.Request.Context(), wsID)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) createAlertSilence() gin.HandlerFunc {
	return func(c *gin.Context) {
		wsID := c.GetString("workspace_id")
		if wsID == "" {
			resp.Error(c, "workspace_id required")
			return
		}
		var req service.CreateSilenceRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			resp.Error(c, err.Error())
			return
		}
		createdBy := c.GetString("user_id")
		data, err := s.alertController.CreateSilence(c.Request.Context(), wsID, createdBy, req)
		if err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, data)
	}
}

func (s *Server) deleteAlertSilence() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := s.alertController.DeleteSilence(c.Request.Context(), id); err != nil {
			resp.Error(c, err.Error())
			return
		}
		resp.OK(c, nil)
	}
}
