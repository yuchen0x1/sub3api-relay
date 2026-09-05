package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/yuchen0x1/sub3api-relay/internal/pkg/response"
	"github.com/yuchen0x1/sub3api-relay/internal/service"
)

type PublicCardHandler struct {
	publicCardService *service.PublicCardService
}

func NewPublicCardHandler(publicCardService *service.PublicCardService) *PublicCardHandler {
	return &PublicCardHandler{publicCardService: publicCardService}
}

type publicCardRedeemHTTPRequest struct {
	Code string `json:"code" binding:"required"`
}

type publicCardQueryHTTPRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

func (h *PublicCardHandler) Redeem(c *gin.Context) {
	setNoStore(c)

	var req publicCardRedeemHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.publicCardService.Redeem(c.Request.Context(), service.PublicCardRedeemRequest{Code: req.Code})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

func (h *PublicCardHandler) Query(c *gin.Context) {
	setNoStore(c)

	var req publicCardQueryHTTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.publicCardService.QueryKey(c.Request.Context(), service.PublicCardKeyQueryRequest{APIKey: req.APIKey})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

func setNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}
