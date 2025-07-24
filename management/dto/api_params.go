package dto

type ApiGroupParams struct {
	Name  string `json:"name" binding:"required"`
	AppId string `json:"app_id" binding:"required"`
}
