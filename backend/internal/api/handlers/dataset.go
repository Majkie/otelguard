package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/otelguard/otelguard/internal/domain"
	"github.com/otelguard/otelguard/internal/service"
	"go.uber.org/zap"
)

// DatasetHandler handles dataset-related endpoints
type DatasetHandler struct {
	datasetService *service.DatasetService
	logger         *zap.Logger
}

// NewDatasetHandler creates a new dataset handler
func NewDatasetHandler(datasetService *service.DatasetService, logger *zap.Logger) *DatasetHandler {
	return &DatasetHandler{
		datasetService: datasetService,
		logger:         logger,
	}
}

// List returns all datasets for a project
func (h *DatasetHandler) List(c *gin.Context) {
	projectID := c.Query("projectId")
	if projectID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "projectId is required",
		})
		return
	}

	if _, err := uuid.Parse(projectID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid projectId format",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	datasets, total, err := h.datasetService.List(c.Request.Context(), projectID, &service.ListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list datasets", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve datasets",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   datasets,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// Get retrieves a dataset by ID
func (h *DatasetHandler) Get(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid dataset ID",
		})
		return
	}

	dataset, err := h.datasetService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dataset not found",
			})
			return
		}
		h.logger.Error("failed to get dataset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve dataset",
		})
		return
	}

	c.JSON(http.StatusOK, dataset)
}

// Create creates a new dataset
func (h *DatasetHandler) Create(c *gin.Context) {
	var input domain.DatasetCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	dataset, err := h.datasetService.Create(c.Request.Context(), &input)
	if err != nil {
		h.logger.Error("failed to create dataset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create dataset",
		})
		return
	}

	c.JSON(http.StatusCreated, dataset)
}

// Update updates a dataset
func (h *DatasetHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid dataset ID",
		})
		return
	}

	var input domain.DatasetUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	dataset, err := h.datasetService.Update(c.Request.Context(), id, &input)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dataset not found",
			})
			return
		}
		h.logger.Error("failed to update dataset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update dataset",
		})
		return
	}

	c.JSON(http.StatusOK, dataset)
}

// Delete deletes a dataset
func (h *DatasetHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid dataset ID",
		})
		return
	}

	if err := h.datasetService.Delete(c.Request.Context(), id); err != nil {
		h.logger.Error("failed to delete dataset", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete dataset",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListItems returns all items for a dataset
func (h *DatasetHandler) ListItems(c *gin.Context) {
	datasetID := c.Param("id")
	if _, err := uuid.Parse(datasetID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid dataset ID",
		})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	items, total, err := h.datasetService.ListItems(c.Request.Context(), datasetID, &service.ListOptions{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.logger.Error("failed to list dataset items", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve dataset items",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// CreateItem creates a new dataset item
func (h *DatasetHandler) CreateItem(c *gin.Context) {
	var input domain.DatasetItemCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	item, err := h.datasetService.CreateItem(c.Request.Context(), &input)
	if err != nil {
		h.logger.Error("failed to create dataset item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to create dataset item",
		})
		return
	}

	c.JSON(http.StatusCreated, item)
}

// GetItem retrieves a dataset item by ID
func (h *DatasetHandler) GetItem(c *gin.Context) {
	itemID := c.Param("itemId")
	if _, err := uuid.Parse(itemID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid item ID",
		})
		return
	}

	item, err := h.datasetService.GetItemByID(c.Request.Context(), itemID)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dataset item not found",
			})
			return
		}
		h.logger.Error("failed to get dataset item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to retrieve dataset item",
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// UpdateItem updates a dataset item
func (h *DatasetHandler) UpdateItem(c *gin.Context) {
	itemID := c.Param("itemId")
	if _, err := uuid.Parse(itemID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid item ID",
		})
		return
	}

	var input domain.DatasetItemUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	item, err := h.datasetService.UpdateItem(c.Request.Context(), itemID, &input)
	if err != nil {
		if err == domain.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "not_found",
				"message": "Dataset item not found",
			})
			return
		}
		h.logger.Error("failed to update dataset item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to update dataset item",
		})
		return
	}

	c.JSON(http.StatusOK, item)
}

// DeleteItem deletes a dataset item
func (h *DatasetHandler) DeleteItem(c *gin.Context) {
	itemID := c.Param("itemId")
	if _, err := uuid.Parse(itemID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "invalid item ID",
		})
		return
	}

	if err := h.datasetService.DeleteItem(c.Request.Context(), itemID); err != nil {
		h.logger.Error("failed to delete dataset item", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "internal_error",
			"message": "Failed to delete dataset item",
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Import imports dataset items from JSON or CSV
func (h *DatasetHandler) Import(c *gin.Context) {
	var input domain.DatasetImport
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": err.Error(),
		})
		return
	}

	count, err := h.datasetService.Import(c.Request.Context(), &input)
	if err != nil {
		h.logger.Error("failed to import dataset items", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "import_error",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Successfully imported dataset items",
		"count":   count,
	})
}
