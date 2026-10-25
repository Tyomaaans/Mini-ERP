package report

import (
	"context"

	"mini-erp/internal/reporting/domain"
	"mini-erp/pkg"
)

// Interface

type ReportService interface {
	Summary(ctx context.Context, kind string, query pkg.ListQuery) ([]SummaryResponse, int64, error)
	LowStock(ctx context.Context, query pkg.ListQuery) ([]LowStockResponse, int64, error)
	Outstanding(ctx context.Context, kind string, query pkg.ListQuery) ([]OutstandingResponse, int64, error)
	AuditLogs(ctx context.Context, query pkg.ListQuery) ([]AuditResponse, int64, error)
}

type reportService struct{ reportRepo domain.ReportRepository }

func NewReportService(reportRepo domain.ReportRepository) ReportService {
	return &reportService{reportRepo: reportRepo}
}

func (s *reportService) Summary(ctx context.Context, kind string, query pkg.ListQuery) ([]SummaryResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if kind != "sales" && kind != "purchasing" {
		return nil, 0, pkg.ErrInvalidInput
	}

	if err := actor.Require("reports." + kind + ".read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.reportRepo.Summary(ctx, kind, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]SummaryResponse, len(entities))
	for i := range entities {
		response[i] = ToSummaryResponse(entities[i])
	}

	return response, total, nil
}

func (s *reportService) LowStock(ctx context.Context, query pkg.ListQuery) ([]LowStockResponse, int64, error) {
	actor := pkg.ActorFrom(ctx)
	if err := actor.Require("reports.inventory.read"); err != nil {
		return nil, 0, err
	}

	if id := query.Filters["warehouse_id"]; id != "" && len(actor.WarehouseScope()) > 0 {
		if err := actor.RequireWarehouse(id); err != nil {
			return nil, 0, err
		}
	}

	entities, total, err := s.reportRepo.LowStock(ctx, query, actor.WarehouseScope())
	if err != nil {
		return nil, 0, err
	}

	response := make([]LowStockResponse, len(entities))
	for i := range entities {
		response[i] = ToLowStockResponse(entities[i])
	}

	return response, total, nil
}

func (s *reportService) Outstanding(ctx context.Context, kind string, query pkg.ListQuery) ([]OutstandingResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("reports.finance.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.reportRepo.Outstanding(ctx, kind, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]OutstandingResponse, len(entities))
	for i := range entities {
		response[i] = ToOutstandingResponse(entities[i])
	}

	return response, total, nil
}

func (s *reportService) AuditLogs(ctx context.Context, query pkg.ListQuery) ([]AuditResponse, int64, error) {
	if err := pkg.ActorFrom(ctx).Require("audit_logs.read"); err != nil {
		return nil, 0, err
	}

	entities, total, err := s.reportRepo.AuditLogs(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	response := make([]AuditResponse, len(entities))
	for i := range entities {
		response[i] = ToAuditResponse(entities[i])
	}

	return response, total, nil
}
