package server

import (
	"wireflow/pkg/utils/resp"

	"github.com/gin-gonic/gin"
)

func (s *Server) monitorRouter() {

	monitorRouter := s.Group("/api/v1/monitor")
	//monitorRouter.Use(dex.AuthMiddleware())
	{
		monitorRouter.GET("/topology", s.topology()) //注册用户
	}
}

func (s *Server) topology() gin.HandlerFunc {
	return func(c *gin.Context) {
		ve, err := s.monitorController.GetPeerStatus(c.Request.Context())
		if err != nil {
			resp.Error(c, "get topoloty falied")
			return
		}

		resp.OK(c, ve)
	}
}
