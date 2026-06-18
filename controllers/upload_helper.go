package controllers

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const maxImageUploadSize = 5 << 20
const maxDocumentUploadSize = 10 << 20

var allowedImageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".webp": true,
	".gif":  true,
}

var allowedTransferProofExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".pdf":  true,
}

func saveOptionalImageUpload(c *gin.Context, uploadRoot, fieldName, subdir string) (string, bool, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) || strings.Contains(err.Error(), "no such file") {
			return "", false, nil
		}
		return "", false, err
	}

	savedPath, err := saveImageUpload(c, uploadRoot, fieldName, subdir, file)
	if err != nil {
		return "", false, err
	}
	return savedPath, true, nil
}

func saveRequiredImageUpload(c *gin.Context, uploadRoot, fieldName, subdir string) (string, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		return "", err
	}
	return saveImageUpload(c, uploadRoot, fieldName, subdir, file)
}

func saveImageUpload(c *gin.Context, uploadRoot, fieldName, subdir string, file *multipart.FileHeader) (string, error) {
	return saveFileUpload(c, uploadRoot, fieldName, subdir, file, allowedImageExts, maxImageUploadSize, "jpg, jpeg, png, webp, or gif")
}

func saveRequiredTransferProofUpload(c *gin.Context, uploadRoot, fieldName, subdir string) (string, error) {
	file, err := c.FormFile(fieldName)
	if err != nil {
		return "", err
	}
	return saveFileUpload(c, uploadRoot, fieldName, subdir, file, allowedTransferProofExts, maxDocumentUploadSize, "jpg, jpeg, png, or pdf")
}

func saveFileUpload(c *gin.Context, uploadRoot, fieldName, subdir string, file *multipart.FileHeader, allowedExts map[string]bool, maxSize int64, allowedMessage string) (string, error) {
	if file.Size > maxSize {
		return "", fmt.Errorf("%s exceeds %dMB limit", fieldName, maxSize>>20)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedExts[ext] {
		return "", fmt.Errorf("%s must be %s", fieldName, allowedMessage)
	}

	targetDir := filepath.Join(uploadRoot, subdir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	targetPath := filepath.Join(targetDir, filename)
	if err := c.SaveUploadedFile(file, targetPath); err != nil {
		return "", err
	}

	return "/uploads/" + strings.Trim(strings.ReplaceAll(filepath.Join(subdir, filename), "\\", "/"), "/"), nil
}

func isMultipartRequest(c *gin.Context) bool {
	return strings.HasPrefix(strings.ToLower(c.ContentType()), "multipart/form-data")
}
