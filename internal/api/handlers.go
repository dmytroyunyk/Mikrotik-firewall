package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// handleGetStats godoc
// @Summary      Get system statistics
// @Description  Returns total events, blocked IPs and events in last 24h
// @Tags         stats
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]int
// @Failure      500  {object}  map[string]string
// @Router       /stats [get]
func (s *Server) handleGetStats(c *gin.Context) {
	stats, err := s.db.GetStats()
	if err != nil {
		s.logger.Error("failed to get status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get statistics",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"total_events": stats["total_events"],
		"blocked_ips":  stats["blocked_ips"],
		"events_24h":   stats["events_24h"],
	})
}

// handleGetBlocked godoc
// @Summary      Get blocked IPs
// @Description  Returns list of currently blocked IP addresses
// @Tags         blocked
// @Produce      json
// @Security     ApiKeyAuth
// @Success      200  {object}  map[string]interface{}
// @Failure      500  {object}  map[string]string
// @Router       /blocked [get]
func (s *Server) handleGetBlocked(c *gin.Context) {
	blocked, err := s.db.GetBlockedIPs()
	if err != nil {
		s.logger.Error("failed to gest blocked IPs", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get blocked IPs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count": len(blocked),
		"items": blocked,
	})
}

// handleUnblock godoc
// @Summary      Unblock an IP
// @Description  Removes an IP from the blacklist
// @Tags         blocked
// @Produce      json
// @Security     ApiKeyAuth
// @Param        ip   path      string  true  "IP address to unblock"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /blocked/{ip} [delete]
func (s *Server) handleUnblock(c *gin.Context) {
	ip := c.Param("ip")

	if ip == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "IP address is required",
		})
		return
	}

	err := s.client.Unblock(ip)
	if err != nil {
		s.logger.Error("failed to unblock IP on router", "ip", ip, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to unblock IP on router",
		})
		return
	}

	err = s.db.MarkAsUnblocked(ip)
	if err != nil {
		s.logger.Error("failed to mark IP as unblocked in db", "ip", ip, "error", err)
	}

	s.logger.Info("IP unblocked via API", "IP", ip)
	c.JSON(http.StatusOK, gin.H{
		"message": "IP unblocked successfully",
		"ip":      ip,
	})
}

// handleGetEvents godoc
// @Summary      Get recent events
// @Description  Returns recent attack events
// @Tags         events
// @Produce      json
// @Security     ApiKeyAuth
// @Param        limit  query     int  false  "Number of events to return"
// @Success      200    {object}  map[string]interface{}
// @Failure      500    {object}  map[string]string
// @Router       /events [get]
func (s *Server) handleGetEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")

	var limit int
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || limit <= 0 {
		limit = 50
	}

	events, err := s.db.GetRecentEvents(limit)
	if err != nil {
		s.logger.Error("failed to get events", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get events",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(events),
		"items": events,
	})
}

// handleGetTopAttackers godoc
// @Summary      Get top attackers
// @Description  Returns top attacking IP addresses
// @Tags         attackers
// @Produce      json
// @Security     ApiKeyAuth
// @Param        limit  query     int  false  "Number of attackers to return"
// @Success      200    {object}  map[string]interface{}
// @Failure      500    {object}  map[string]string
// @Router       /attackers [get]
func (s *Server) handleGetTopAttackers(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")

	var limit int
	if _, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || limit <= 0 {
		limit = 10
	}

	attackers, err := s.db.GetTopAttackers(limit)
	if err != nil {
		s.logger.Error("failed to get top attackers", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get top attackers",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"count": len(attackers),
		"items": attackers,
	})
}
