package controllers

import (
	"errors"
	"net/http"
	"time"

	"begmt2/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type NotificationController struct {
	db *gorm.DB
}

func NewNotificationController(db *gorm.DB) NotificationController {
	return NotificationController{db: db}
}

type notificationResponse struct {
	models.Notification
	Status string `json:"status"`
}

func (n NotificationController) ListNotifications(c *gin.Context) {
	role := c.GetString("role")
	status := c.Query("status")

	query := n.db.Where("role = ?", role).Order("created_at DESC")
	switch status {
	case "terbaca":
		query = query.Where("read_at IS NOT NULL")
	case "belum_terbaca":
		query = query.Where("read_at IS NULL")
	case "":
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": "status must be terbaca or belum_terbaca"})
		return
	}

	var notifications []models.Notification
	if err := query.Find(&notifications).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get notifications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notifications": mapNotificationResponses(notifications)})
}

func (n NotificationController) GetNotification(c *gin.Context) {
	notificationID, ok := parseUintParam(c, "id", "invalid notification id")
	if !ok {
		return
	}

	var notification models.Notification
	if err := n.db.Where("role = ?", c.GetString("role")).First(&notification, notificationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "notification not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to get notification"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"notification": mapNotificationResponse(notification)})
}

func (n NotificationController) MarkNotificationAsRead(c *gin.Context) {
	notificationID, ok := parseUintParam(c, "id", "invalid notification id")
	if !ok {
		return
	}

	now := time.Now()
	var notification models.Notification
	err := n.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role = ?", c.GetString("role")).First(&notification, notificationID).Error; err != nil {
			return err
		}

		if notification.ReadAt == nil {
			notification.ReadAt = &now
			if err := tx.Save(&notification).Error; err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "notification not found"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to mark notification as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "notification marked as read", "notification": mapNotificationResponse(notification)})
}

func (n NotificationController) MarkAllNotificationsAsRead(c *gin.Context) {
	role := c.GetString("role")
	now := time.Now()

	if err := n.db.Model(&models.Notification{}).
		Where("role = ? AND read_at IS NULL", role).
		Update("read_at", &now).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to mark notifications as read"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all notifications marked as read"})
}

func mapNotificationResponses(notifications []models.Notification) []notificationResponse {
	responses := make([]notificationResponse, 0, len(notifications))
	for _, notification := range notifications {
		responses = append(responses, mapNotificationResponse(notification))
	}
	return responses
}

func mapNotificationResponse(notification models.Notification) notificationResponse {
	status := "belum_terbaca"
	if notification.ReadAt != nil {
		status = "terbaca"
	}

	return notificationResponse{
		Notification: notification,
		Status:       status,
	}
}
