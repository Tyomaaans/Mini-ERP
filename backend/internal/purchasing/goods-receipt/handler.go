package goodsreceipt

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type GoodsReceiptHandler struct{ goodsReceiptSvc GoodsReceiptService }

func NewGoodsReceiptHandler(goodsReceiptSvc GoodsReceiptService) *GoodsReceiptHandler {
	return &GoodsReceiptHandler{goodsReceiptSvc: goodsReceiptSvc}
}

func (h *GoodsReceiptHandler) CreateGoodsReceipt(c *gin.Context) {
	var req CreateGoodsReceiptRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.goodsReceiptSvc.CreateGoodsReceipt(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *GoodsReceiptHandler) GetGoodsReceipt(c *gin.Context) {
	response, err := h.goodsReceiptSvc.GetGoodsReceipt(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *GoodsReceiptHandler) GetGoodsReceipts(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "gr_number"}, []string{"status", "warehouse_id", "purchase_order_id"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.goodsReceiptSvc.GetGoodsReceipts(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *GoodsReceiptHandler) Post(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.goodsReceiptSvc.Post(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
