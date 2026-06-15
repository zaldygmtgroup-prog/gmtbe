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
	NameProduct string  `json:"namaproduct" form:"namaproduct" binding:"required,max=150"`
	Photo       string  `json:"foto"        form:"foto"        binding:"omitempty,max=255"`
	Description string  `json:"deskripsi"   form:"deskripsi"`
	Unit        string  `json:"unit"        form:"unit"        binding:"required,max=50"`
	Price       int64   `json:"price"       form:"price"       binding:"required,min=1"`
	Status      string  `json:"status"      form:"status"`
	Komisi      float64 `json:"komisi"      form:"komisi"      binding:"omitempty,min=0"`
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

	for i := range products {
		products[i].PopulateCommissionTiers()
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

	product.PopulateCommissionTiers()

	c.JSON(http.StatusOK, gin.H{"product": product})
}

func (p ProductController) CreateProduct(c *gin.Context) {
	var req productRequest
	multipart := isMultipartRequest(c) // ✅ Dicek sekali saja

	if err := bindProductRequest(c, &req, multipart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	if multipart {
		photo, uploaded, err := saveOptionalImageUpload(c, p.cfg.UploadDir, "foto", "products")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid product photo", "error": err.Error()})
			return
		}
		if uploaded {
			req.Photo = photo
		}
	}

	product := models.Product{
		NameProduct: req.NameProduct,
		Photo:       req.Photo,
		Description: req.Description,
		Unit:        req.Unit,
		Price:       req.Price,
		Status:      req.Status,
		Komisi:      req.Komisi,
	}

	if err := p.db.Create(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create product"})
		return
	}

	product.PopulateCommissionTiers()

	c.JSON(http.StatusCreated, gin.H{"message": "product created", "product": product})
}

func (p ProductController) UpdateProduct(c *gin.Context) {
	productID, ok := parseProductID(c)
	if !ok {
		return
	}

	multipart := isMultipartRequest(c) // ✅ Dicek sekali saja

	var req productRequest
	if err := bindProductRequest(c, &req, multipart); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request", "error": err.Error()})
		return
	}

	photoUploaded := false
	if multipart {
		photo, uploaded, err := saveOptionalImageUpload(c, p.cfg.UploadDir, "foto", "products")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid product photo", "error": err.Error()})
			return
		}
		if uploaded {
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
	product.Description = req.Description
	product.Unit = req.Unit
	product.Price = req.Price
	product.Status = req.Status
	product.Komisi = req.Komisi

	// ✅ Update foto hanya jika: JSON request, atau multipart dengan foto baru diupload
	if !multipart || photoUploaded {
		product.Photo = req.Photo
	}

	if err := p.db.Save(&product).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to update product"})
		return
	}

	product.PopulateCommissionTiers()

	c.JSON(http.StatusOK, gin.H{"message": "product updated", "product": product})
}

// ✅ multipart diterima sebagai parameter, tidak dipanggil ulang di dalam fungsi
func bindProductRequest(c *gin.Context, req *productRequest, multipart bool) error {
	if !multipart {
		return c.ShouldBindJSON(req)
	}

	req.NameProduct = c.PostForm("namaproduct")
	req.Photo = c.PostForm("foto")
	req.Description = c.PostForm("deskripsi")
	req.Unit = c.PostForm("unit")
	req.Status = c.PostForm("status")

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

	priceStr := c.PostForm("price")
	price, err := strconv.ParseInt(priceStr, 10, 64)
	if err != nil || price < 1 {
		return errors.New("price is required and must be at least 1")
	}
	req.Price = price

	komisiStr := c.PostForm("komisi")
	if komisiStr != "" {
		komisi, err := strconv.ParseFloat(komisiStr, 64)
		if err != nil || komisi < 0 { // ✅ Tambah validasi min=0 sesuai struct tag
			return errors.New("komisi must be a valid number and at least 0")
		}
		req.Komisi = komisi
	}

	return nil
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
