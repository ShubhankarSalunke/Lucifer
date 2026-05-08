package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ShubhankarSalunke/chaos-engineering/datamodel"
	"github.com/ShubhankarSalunke/lucifer/connectors"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/health", func(c *gin.Context) {
		agents, err := datamodel.GetAgents(context.Background(), connectors.AWSConfig{})
		if err != nil {
			c.JSON(500, gin.H{"status": "error", "message": "cannot reach orchestrator files", "detail": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "ok", "agents_found": len(agents)})
	})

	r.GET("/api/agents", func(c *gin.Context) {
		agents, err := datamodel.GetAgents(c.Request.Context(), connectors.AWSConfig{})
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		fmt.Printf("Serving %d discovered agents to UI\n", len(agents))
		c.JSON(200, agents)
	})

	r.GET("/api/stats", func(c *gin.Context) {
		stats, err := datamodel.GetFleetStats()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, stats)
	})

	r.GET("/api/metrics/compute/aggregate/:id", func(c *gin.Context) {
		id := c.Param("id")
		
		metrics, err := datamodel.GetComputeMetrics(id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, metrics)
	})

	r.GET("/api/metrics/compute/live/:id", func(c *gin.Context) {
		id := c.Param("id")
		metrics, err := datamodel.GetComputeMetrics(id)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, metrics)
	})

	
	r.GET("/api/results", func(c *gin.Context) {
		results, err := datamodel.GetResults()
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, results)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8001"
	}

	fmt.Printf("Datamodel Server running on port %s\n", port)
	r.Run(":" + port)
}
