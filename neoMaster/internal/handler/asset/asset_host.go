package asset

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	assetmodel "neomaster/internal/model/asset"
	"neomaster/internal/pkg/logger"
	assetservice "neomaster/internal/service/asset"
)

// AssetHostHandler 资产主机处理器
type AssetHostHandler struct {
	service *assetservice.AssetHostService
}

// NewAssetHostHandler 创建 AssetHostHandler 实例
func NewAssetHostHandler(service *assetservice.AssetHostService) *AssetHostHandler {
	return &AssetHostHandler{
		service: service,
	}
}

// -----------------------------------------------------------------------------
// AssetHost Handlers
// -----------------------------------------------------------------------------

// CreateHost 创建主机
func (h *AssetHostHandler) CreateHost(c *gin.Context) {
	var host assetmodel.AssetHost
	if err := c.ShouldBindJSON(&host); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.CreateHost(c.Request.Context(), &host); err != nil {
		logger.LogError(err, "", 0, "", "create_host", "HANDLER", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 200, "msg": "Host created successfully", "data": host})
}

// GetHost 获取主机详情
func (h *AssetHostHandler) GetHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	host, err := h.service.GetHost(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": host})
}

// UpdateHost 更新主机
func (h *AssetHostHandler) UpdateHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var host assetmodel.AssetHost
	if err := c.ShouldBindJSON(&host); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	host.ID = id // Ensure ID is set from URL

	if err := h.service.UpdateHost(c.Request.Context(), &host); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Host updated successfully"})
}

// DeleteHost 删除主机
func (h *AssetHostHandler) DeleteHost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.service.DeleteHost(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Host deleted successfully"})
}

// ListHosts 获取主机列表
func (h *AssetHostHandler) ListHosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	ip := c.Query("ip")
	hostname := c.Query("hostname")
	os := c.Query("os")

	hosts, total, err := h.service.ListHosts(c.Request.Context(), page, pageSize, ip, hostname, os)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": map[string]interface{}{
			"list":  hosts,
			"total": total,
			"page":  page,
			"size":  pageSize,
		},
	})
}

// -----------------------------------------------------------------------------
// AssetService Handlers
// -----------------------------------------------------------------------------

// CreateService 创建服务
func (h *AssetHostHandler) CreateService(c *gin.Context) {
	var service assetmodel.AssetService
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if err := h.service.CreateService(c.Request.Context(), &service); err != nil {
		logger.LogError(err, "", 0, "", "create_service", "HANDLER", nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 200, "msg": "Service created successfully", "data": service})
}

// GetService 获取服务详情
func (h *AssetHostHandler) GetService(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	service, err := h.service.GetService(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": service})
}

// UpdateService 更新服务
func (h *AssetHostHandler) UpdateService(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var service assetmodel.AssetService
	if err := c.ShouldBindJSON(&service); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	service.ID = id

	if err := h.service.UpdateService(c.Request.Context(), &service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Service updated successfully"})
}

// DeleteService 删除服务
func (h *AssetHostHandler) DeleteService(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.service.DeleteService(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "Service deleted successfully"})
}

// ListServicesByHostID 获取指定主机的服务列表
func (h *AssetHostHandler) ListServicesByHostID(c *gin.Context) {
	hostIDStr := c.Param("host_id")
	hostID, err := strconv.ParseUint(hostIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Host ID"})
		return
	}

	services, err := h.service.ListServicesByHostID(c.Request.Context(), hostID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": services})
}

// ListServices 获取服务列表 (Global)
func (h *AssetHostHandler) ListServices(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	port, _ := strconv.Atoi(c.DefaultQuery("port", "0"))
	name := c.Query("name")
	proto := c.Query("proto")

	services, total, err := h.service.ListServices(c.Request.Context(), page, pageSize, port, name, proto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": map[string]interface{}{
			"list":  services,
			"total": total,
			"page":  page,
			"size":  pageSize,
		},
	})
}
