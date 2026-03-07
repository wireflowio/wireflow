package server

import (
	"wireflow/pkg/utils/resp"

	"github.com/gin-gonic/gin"
)

func (s *Server) monitorRouter() {

	monitorRouter := s.Group("/api/v1/monitor")
	//monitorRouter.Use(dex.AuthMiddleware())
	{
		monitorRouter.GET("/topology", s.topology())
		monitorRouter.GET("/snapshot", s.nodeSnpashot())
	}
}

func (s *Server) topology() gin.HandlerFunc {
	return func(c *gin.Context) {
		ve, err := s.monitorController.GetTopologySnapshot(c.Request.Context())
		if err != nil {
			resp.Error(c, "get topoloty falied")
			return
		}

		resp.OK(c, ve)
	}
}

func (s *Server) nodeSnpashot() gin.HandlerFunc {
	return func(c *gin.Context) {
		ve, err := s.monitorController.GetNodeSnapshot(c.Request.Context())
		if err != nil {
			resp.Error(c, "get topoloty falied")
			return
		}

		resp.OK(c, ve)
	}
}
