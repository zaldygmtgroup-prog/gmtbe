package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductController struct {
	db *gorm.DB
}

func NewProductController(db *gorm.DB) ProductController {
	return ProductController{db: db}
}

type productRequest struct {
	NameProduct string `json:"namaproduct" binding:"required,max=150"`
	Photo       string `json:"foto" binding:"omitempty,max=255"`
	Description string `json:"deskripsi"`
	Unit        string `json:"unit" binding:"required,max=50"`
	Price       int64  `json:"price" binding:"required,min=1"`
}

func (p ProductController) ListProducts(c *gin.Context) {
	search := c.Query("search")

	query := p.db.Order("created_at DESC")
	if search != "" {
		like := "%" + search + "%"
		query = query.Where("namaproduct LIKE ? OR deskripsi LIKE ?", like, like)
	}

	var products []models.Product
	if err := query.Find(&products).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get products"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"products": products})
}

func (p ProductController) GetProduct(c *gin.Context) {
	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	var product models.Product
	if err := p.db.First(&product, "id_product = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "product not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (p ProductController) CreateProduct(c *gin.Context) {
	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	product := models.Product{
		NameProduct: req.NameProduct,
		Photo:       req.Photo,
		Description: req.Description,
		Unit:        req.Unit,
		Price:       req.Price,
	}

	if err := p.db.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "product created", "product": product})
}

func (p ProductController) UpdateProduct(c *gin.Context) {
	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	var req productRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	var product models.Product
	if err := p.db.First(&product, "id_product = ?", productID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "product not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get product"})
		return
	}

	product.NameProduct = req.NameProduct
	product.Photo = req.Photo
	product.Description = req.Description
	product.Unit = req.Unit
	product.Price = req.Price

	if err := p.db.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product updated", "product": product})
}

func (p ProductController) DeleteProduct(c *gin.Context) {
	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	result := p.db.Delete(&models.Product{}, "id_product = ?", productID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to delete product"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"message": "product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product deleted"})
}

func parseProductID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid product id"})
		return 0, false
	}

	return uint(id), true
}
