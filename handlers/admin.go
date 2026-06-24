package handlers

import (
	"context"
	"net/http"
	"time"

	"AgenticGo/models"
	"AgenticGo/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// SyncDeviceApps endpoint untuk sinkronisasi daftar aplikasi dari perangkat Android
// @Summary Sync device apps inventory
// @Description Receives app inventory list from Android device and stores it in MongoDB
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.SyncDeviceAppsRequest true "App Inventory Info"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/admin/sync-device-apps [post]
func SyncDeviceApps(c *gin.Context) {
	var req models.SyncDeviceAppsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "Invalid request: " + err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"user_id": req.UserID, "device_id": req.DeviceID}
	update := bson.M{
		"$set": bson.M{
			"apps":       req.Apps,
			"updated_at": time.Now().UTC(),
		},
	}

	// Update if exists, otherwise insert (Upsert)
	opts := options.UpdateOne().SetUpsert(true)
	_, err := services.DeviceAppInventoryCol.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": "Gagal menyimpan inventory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "App inventory synchronized successfully.",
		"count":   len(req.Apps),
	})
}

// SyncGitHub endpoint untuk memicu sinkronisasi repositori
// @Summary Sync GitHub repositories
// @Description Fetches repos from GitHub and ingests into Qdrant concurrently
// @Tags admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /v1/admin/sync-github [post]
func SyncGitHub(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	stats, err := services.SyncGitHubRepositories(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"stats":  stats,
	})
}
