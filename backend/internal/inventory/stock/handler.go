package stock

import (
	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type StockHandler struct{ stockSvc StockService }

func NewStockHandler(stockSvc StockService) *StockHandler {
	return &StockHandler{stockSvc: stockSvc}
}

func (h *StockHandler) GetBalances(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"updated_at", "created_at", "quantity_on_hand"}, []string{"warehouse_id", "product_id"}, false)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.stockSvc.GetBalances(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *StockHandler) GetMovements(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"occurred_at", "quantity_delta"}, []string{"warehouse_id", "product_id", "movement_type", "reference_id"}, false)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.stockSvc.GetMovements(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}
