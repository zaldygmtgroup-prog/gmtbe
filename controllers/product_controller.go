package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"begmt2/config"
	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProductController struct {
	cfg config.Config
	db  *gorm.DB
}

func NewProductController(cfg config.Config, db *gorm.DB) ProductController {
	return ProductController{cfg: cfg, db: db}
}

type productRequest struct {
	NameProduct string `json:"namaproduct" form:"namaproduct" binding:"required,max=150"`
	Photo       string `json:"foto" form:"foto" binding:"omitempty,max=255"`
	Description string `json:"deskripsi" form:"deskripsi"`
	Unit        string `json:"unit" form:"unit" binding:"required,max=50"`
	Price       int64  `json:"price" form:"price" binding:"required,min=1"`
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
	if err := bindProductRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	if isMultipartRequest(c) {
		if photo, ok, err := saveOptionalImageUpload(c, p.cfg.UploadDir, "foto", "products"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid product photo", "error": err.Error()})
			return
		} else if ok {
			req.Photo = photo
		}
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
	if err := bindProductRequest(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}
	photoUploaded := false
	if isMultipartRequest(c) {
		if photo, ok, err := saveOptionalImageUpload(c, p.cfg.UploadDir, "foto", "products"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid product photo", "error": err.Error()})
			return
		} else if ok {
			req.Photo = photo
			photoUploaded = true
		}
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
	if !isMultipartRequest(c) || photoUploaded || req.Photo != "" {
		product.Photo = req.Photo
	}
	product.Description = req.Description
	product.Unit = req.Unit
	product.Price = req.Price

	if err := p.db.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product updated", "product": product})
}

func bindProductRequest(c *gin.Context, req *productRequest) error {
	if isMultipartRequest(c) {
		req.NameProduct = c.PostForm("namaproduct")
		req.Photo = c.PostForm("foto")
		req.Description = c.PostForm("deskripsi")
		req.Unit = c.PostForm("unit")
		price, err := strconv.ParseInt(c.PostForm("price"), 10, 64)
		if err != nil || price < 1 {
			return errors.New("price is required and must be at least 1")
		}
		req.Price = price
		if req.NameProduct == "" {
			return errors.New("namaproduct is required")
		}
		if len(req.NameProduct) > 150 {
			return errors.New("namaproduct must be at most 150 characters")
		}
		if len(req.Photo) > 255 {
			return errors.New("foto must be at most 255 characters")
		}
		if req.Unit == "" {
			return errors.New("unit is required")
		}
		if len(req.Unit) > 50 {
			return errors.New("unit must be at most 50 characters")
		}
		return nil
	}
	return c.ShouldBindJSON(req)
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
