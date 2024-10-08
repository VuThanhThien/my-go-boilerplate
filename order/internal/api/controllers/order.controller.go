package controllers

import (
	"net/http"
	"strconv"

	"github.com/VuThanhThien/golang-gorm-postgres/order/internal/api/services"
	"github.com/VuThanhThien/golang-gorm-postgres/order/internal/models/dto"
	"github.com/gin-gonic/gin"
)

type OrderController struct {
	service services.IOrderService
}

func NewOrderController(service services.IOrderService) OrderController {
	return OrderController{service: service}
}

// @Summary Get Orders
// @Description Get orders by filter
// @Tags Orders
// @Accept json
// @Produce json
// @Param order query dto.GetOrderRequestDto true "Order details"
// @Success 200 {object} dto.GetOrderResponseDto
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /orders [get]
func (c *OrderController) GetOrders(ctx *gin.Context) {
	var getOrderDto dto.GetOrderRequestDto
	if err := ctx.ShouldBindQuery(&getOrderDto); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	status := ctx.Query("status")
	if status != "" {
		getOrderDto.Status = dto.OrderStatus(status)
	}
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	orders, err := c.service.GetOrders(getOrderDto, page, pageSize)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": orders})
}

// @Summary Get Order
// @Description Get an order by ID
// @Tags Orders
// @Accept json
// @Produce json
// @Param id path int true "Order ID"
// @Success 200 {object} dto.GetOrderResponseDto
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /orders/{id} [get]
func (c *OrderController) GetOrder(ctx *gin.Context) {
	var readIdRequest dto.ReadIdRequest
	if err := ctx.ShouldBindUri(&readIdRequest); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	order, err := c.service.GetOrder(uint(readIdRequest.ID))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": order})
}

// @Summary Create Order
// @Description Create a new order
// @Tags Orders
// @Accept json
// @Produce json
// @Param order body dto.CreateOrderRequestDto true "Order details"
// @Success 200 {object} dto.CreateOrderResponseDto
// @Failure 400 {object} gin.H
// @Failure 500 {object} gin.H
// @Router /orders [post]
func (c *OrderController) CreateOrder(ctx *gin.Context) {
	var createOrderDto dto.CreateOrderRequestDto
	if err := ctx.ShouldBindJSON(&createOrderDto); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	order, err := c.service.CreateOrder(createOrderDto)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": order})
}
