package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gateway/database"
	"gateway/models"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "gateway/proto"
)

func main() {
	if err := database.InitDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	workerAddress, err := requiredEnv("GRPC_WORKER_ADDRESS")
	if err != nil {
		log.Fatal(err)
	}
	httpAddress, err := requiredEnv("HTTP_ADDRESS")
	if err != nil {
		log.Fatal(err)
	}

	// Reuse one connection for the process lifetime to avoid dialing on every request.
	conn, err := grpc.NewClient(workerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Python worker: %v", err)
	}
	defer conn.Close()

	grpcClient := pb.NewTextAnalysisServiceClient(conn)

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) {
		sqlDB, err := database.DB.DB()
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.POST("/api/v1/analyze", func(c *gin.Context) {
		var reqBody struct {
			Texts []string `json:"texts"`
		}

		if err := c.ShouldBindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// Bound worker calls so a stalled inference cannot hold the HTTP request indefinitely.
		ctx, cancel := context.WithTimeout(
			c.Request.Context(),
			5*time.Second,
		)
		defer cancel()

		grpcReq := &pb.AnalyzeRequest{Texts: reqBody.Texts}
		grpcResp, err := grpcClient.AnalyzeText(ctx, grpcReq)

		if err != nil {
			log.Printf("gRPC call failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal inference service error"})
			return
		}

		results := grpcResp.GetResults()

		// Results correspond to inputs by position; guard against an incomplete worker response.
		for i, text := range reqBody.Texts {
			if i < len(results) {
				newRecord := models.AnalysisRecord{
					UserID:         1, // TODO: Replace with the authenticated user ID when JWT support is added.
					RawText:        text,
					SentimentLabel: results[i].GetSentiment(),
					SentimentScore: float64(results[i].GetConfidenceScore()),
				}

				if dbErr := database.DB.Create(&newRecord).Error; dbErr != nil {
					// Persistence is best-effort; inference results remain available if this write fails.
					log.Printf("Failed to persist analysis (text: %s): %v", text, dbErr)
				} else {
					log.Printf("Analysis persisted with ID %d", newRecord.ID)
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":                   "success",
			"results":                  grpcResp.GetResults(),
			"total_processing_time_ms": grpcResp.GetProcessingTimeMs(),
		})
	})

	log.Printf("Go API gateway listening on %s; Python worker address: %s", httpAddress, workerAddress)
	if err := r.Run(httpAddress); err != nil {
		log.Fatalf("Failed to start gateway: %v", err)
	}
}

func requiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}
