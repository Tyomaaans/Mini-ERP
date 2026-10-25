package report

import (
	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type ReportHandler struct{ reportSvc ReportService }

func NewReportHandler(reportSvc ReportService) *ReportHandler {
	return &ReportHandler{reportSvc: reportSvc}
}

func (h *ReportHandler) Summary(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"date", "order_count", "total_amount"}, []string{"from", "to", "warehouse_id"}, false)
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}

		response, total, err := h.reportSvc.Summary(c.Request.Context(), kind, query)
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}

		pkg.ListResponse(c, response, query, total)
	}
}

func (h *ReportHandler) LowStock(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"quantity_available", "sku", "reorder_level"}, []string{"warehouse_id", "product_id"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.reportSvc.LowStock(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *ReportHandler) Outstanding(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"due_date", "outstanding_amount", "invoice_number"}, []string{"party_id", "overdue"})
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}

		response, total, err := h.reportSvc.Outstanding(c.Request.Context(), kind, query)
		if err != nil {
			pkg.ErrorResponse(c, err)
			return
		}

		pkg.ListResponse(c, response, query, total)
	}
}

func (h *ReportHandler) AuditLogs(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"occurred_at", "action"}, []string{"actor_id", "entity_type", "entity_id", "request_id", "action", "from", "to"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.reportSvc.AuditLogs(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}
