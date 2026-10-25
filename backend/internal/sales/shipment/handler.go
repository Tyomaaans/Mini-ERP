package shipment

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"mini-erp/pkg"
)

type ShipmentHandler struct{ shipmentSvc ShipmentService }

func NewShipmentHandler(shipmentSvc ShipmentService) *ShipmentHandler {
	return &ShipmentHandler{shipmentSvc: shipmentSvc}
}

func (h *ShipmentHandler) CreateShipment(c *gin.Context) {
	var req CreateShipmentRequest
	if err := pkg.BindJSON(c, &req); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.shipmentSvc.CreateShipment(c.Request.Context(), req)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusCreated, response)
}

func (h *ShipmentHandler) GetShipment(c *gin.Context) {
	response, err := h.shipmentSvc.GetShipment(c.Request.Context(), c.Param("id"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}

func (h *ShipmentHandler) GetShipments(c *gin.Context) {
	query, err := pkg.ParsePagination(c.Request.URL.Query(), []string{"created_at", "shipment_number"}, []string{"status", "warehouse_id", "sales_order_id"})
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, total, err := h.shipmentSvc.GetShipments(c.Request.Context(), query)
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.ListResponse(c, response, query, total)
}

func (h *ShipmentHandler) Post(c *gin.Context) {
	if err := pkg.BindAction(c); err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	response, err := h.shipmentSvc.Post(c.Request.Context(), c.Param("id"), c.GetHeader("Idempotency-Key"))
	if err != nil {
		pkg.ErrorResponse(c, err)
		return
	}

	pkg.SuccessResponse(c, http.StatusOK, response)
}
