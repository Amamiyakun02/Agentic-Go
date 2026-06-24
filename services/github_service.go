package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type GitHubRepo struct {
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	Description string `json:"description"`
	Language    string `json:"language"`
	Fork        bool   `json:"fork"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type SyncStats struct {
	TotalFound   int
	SuccessCount int
	FailCount    int
	Details      []map[string]string
}

// SyncGitHubRepositories mengambil daftar repositori dari GitHub, mendownload README secara paralel,
// dan mendaftarkan dokumennya ke MongoDB & Qdrant.
func SyncGitHubRepositories(ctx context.Context) (SyncStats, error) {
	var stats SyncStats

	username := os.Getenv("GITHUB_USERNAME")
	if username == "" {
		username = "Amamiyakun02"
	}
	token := os.Getenv("GITHUB_TOKEN")

	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.github.com/users/%s/repos?per_page=100", username)
	if token != "" {
		url = "https://api.github.com/user/repos?per_page=100&type=owner"
	}
	
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "AgenticGo")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	resp, err := client.Do(req)
	if err != nil {
		return stats, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return stats, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}

	var repos []GitHubRepo
	if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
		return stats, err
	}

	stats.TotalFound = len(repos)
	log.Printf("[GITHUB] Menemukan %d repositori, memproses dengan Goroutines...", len(repos))

	// Channel untuk mengumpulkan hasil secara asinkron (Thread-safe)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, repo := range repos {
		if repo.Fork {
			log.Printf("[GITHUB] Melewati fork: %s", repo.Name)
			continue
		}

		wg.Add(1)
		go func(r GitHubRepo) {
			defer wg.Done()

			// 1. Fetch README
			readmeURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/readme", r.Owner.Login, r.Name)
			reqRead, _ := http.NewRequest("GET", readmeURL, nil)
			reqRead.Header.Set("Accept", "application/vnd.github+json")
			reqRead.Header.Set("User-Agent", "AgenticGo")
			if token != "" {
				reqRead.Header.Set("Authorization", "token "+token)
			}

			var readmeContent string
			respRead, err := client.Do(reqRead)
			if err == nil && respRead.StatusCode == 200 {
				var data map[string]interface{}
				if json.NewDecoder(respRead.Body).Decode(&data) == nil {
					if contentB64, ok := data["content"].(string); ok {
						cleanB64 := strings.ReplaceAll(contentB64, "\n", "")
						cleanB64 = strings.ReplaceAll(cleanB64, "\r", "")
						if decoded, err := base64.StdEncoding.DecodeString(cleanB64); err == nil {
							readmeContent = string(decoded)
						}
					}
				}
			}
			if respRead != nil {
				respRead.Body.Close()
			}

			// 2. Format Konten
			fullContent := fmt.Sprintf("Repository Name: %s\nOwner: %s\nURL: %s\nMain Language: %s\nDescription: %s\n",
				r.Name, r.Owner.Login, r.HTMLURL, r.Language, r.Description)
			
			if readmeContent != "" {
				fullContent += fmt.Sprintf("\n--- README.md ---\n%s", readmeContent)
			} else {
				fullContent += "\n(No README.md content available for this repository)"
			}

			title := "GitHub: " + r.Name

			// 3. Simpan ke MongoDB
			filter := bson.M{"source_type": "github", "file_url": r.HTMLURL}
			var existingDoc struct {
				ID bson.ObjectID `bson:"_id"`
			}
			
			var docID string
			err = DocumentsCol.FindOne(ctx, filter).Decode(&existingDoc)
			if err == nil {
				// Update
				docID = existingDoc.ID.Hex()
				DocumentsCol.UpdateOne(ctx, bson.M{"_id": existingDoc.ID}, bson.M{
					"$set": bson.M{
						"title":      title,
						"content":    fullContent,
						"updated_at": time.Now().UTC(),
					},
				})
			} else {
				// Insert
				newID := bson.NewObjectID()
				docID = newID.Hex()
				DocumentsCol.InsertOne(ctx, bson.M{
					"_id":         newID,
					"title":       title,
					"source_type": "github",
					"file_url":    r.HTMLURL,
					"status":      "active",
					"content":     fullContent,
					"chunk_count": 0,
					"product_id":  nil,
					"created_at":  time.Now().UTC(),
					"updated_at":  time.Now().UTC(),
				})
			}

			// 4. Trigger RAG Ingestion (juga berjalan paralel di dalamnya)
			success := IngestDocumentEmbedding(ctx, docID, title, fullContent, "github", "", true)

			mu.Lock()
			if success {
				stats.SuccessCount++
				stats.Details = append(stats.Details, map[string]string{"name": r.Name, "status": "success"})
			} else {
				stats.FailCount++
				stats.Details = append(stats.Details, map[string]string{"name": r.Name, "status": "failed"})
			}
			mu.Unlock()

		}(repo)
	}

	// Tunggu semua repository selesai di-fetch dan di-ingest!
	wg.Wait()

	return stats, nil
}
