package controller

import (
	"context"
	"wireflow/internal/log"
	"wireflow/management/service"
	"wireflow/monitor"
)

type MonitorController interface {
	GetTopologySnapshot(ctx context.Context) ([]monitor.PeerSnapshot, error)
}

type monitorController struct {
	monitorService service.MonitorService
	log            *log.Logger
}

func (m *monitorController) GetTopologySnapshot(ctx context.Context) ([]monitor.PeerSnapshot, error) {
	return m.monitorService.GetTopologySnapshot(ctx)
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
