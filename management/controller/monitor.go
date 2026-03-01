package controller

import (
	"context"
	"wireflow/internal/log"
	"wireflow/management/service"

	"github.com/prometheus/common/model"
)

type MonitorController interface {
	GetPeerStatus(ctx context.Context) (model.Vector, error)
}

type monitorController struct {
	monitorService service.MonitorService
	log            *log.Logger
}

func (m *monitorController) GetPeerStatus(ctx context.Context) (model.Vector, error) {
	return m.monitorService.GetPeerStatus(ctx)
}

func NewMonitorController(address string) MonitorController {
	logger := log.GetLogger("monitor-controller")
	svc, err := service.NewMonitorService(address)
	if err != nil {
		logger.Error("init monitor service failed", err)
	}
	return &monitorController{
		monitorService: svc,
		log:            logger,
	}
}
