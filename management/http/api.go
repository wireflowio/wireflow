package http

import (
	"github.com/gin-gonic/gin"
	"linkany/management/dto"
)

func (s *Server) RegisterApis() {
	s.RegisterGroupApis()
}

func (s *Server) RegisterGroupApis() {
	groupApis := s.RouterGroup.Group(PREFIX + "/group")
	groupApis.POST("/join", s.tokenFilter(), s.joinGroup())
	groupApis.POST("/leave", s.tokenFilter(), s.leaveGroup())
	groupApis.POST("/remove", s.tokenFilter(), s.removeGroup())
	groupApis.POST("/add", s.tokenFilter(), s.removeGroup())
}

func (s *Server) joinGroup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var params dto.ApiGroupParams
		if err := c.ShouldBindJSON(&params); err != nil {
			WriteBadRequest(c.JSON, "invalid request: "+err.Error())
			return
		}

		if err := s.groupController.JoinGroup(c, &params); err != nil {
			WriteError(c.JSON, err.Error())
			return
		}

		WriteOK(c.JSON, "joined group successfully")
	}
}

func (s *Server) leaveGroup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var params dto.ApiGroupParams
		if err := c.ShouldBindJSON(&params); err != nil {
			WriteBadRequest(c.JSON, "invalid request: "+err.Error())
			return
		}

		if err := s.groupController.LeaveGroup(c, &params); err != nil {
			WriteError(c.JSON, err.Error())
			return
		}

		WriteOK(c.JSON, "left group successfully")
	}
}

func (s *Server) removeGroup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var params dto.ApiGroupParams
		if err := c.ShouldBindJSON(&params); err != nil {
			WriteBadRequest(c.JSON, "invalid request: "+err.Error())
			return
		}

		if err := s.groupController.RemoveGroup(c, &params); err != nil {
			WriteError(c.JSON, err.Error())
			return
		}

		WriteOK(c.JSON, "remove group successfully")
	}
}

func (s *Server) addGroup() gin.HandlerFunc {
	return func(c *gin.Context) {
		var params dto.ApiGroupParams
		if err := c.ShouldBindJSON(&params); err != nil {
			WriteBadRequest(c.JSON, "invalid request: "+err.Error())
			return
		}

		if err := s.groupController.AddGroup(c, &params); err != nil {
			WriteError(c.JSON, err.Error())
			return
		}

		WriteOK(c.JSON, "add group successfully")
	}
}
