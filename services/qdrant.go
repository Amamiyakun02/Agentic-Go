package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"AgenticGo/utils"
)

// QdrantRequest adalah helper untuk memanggil Qdrant HTTP API
func QdrantRequest(method, endpoint string, payload map[string]interface{}) ([]byte, error) {
	qdrantURL := os.Getenv("QDRANT_URL")
	qdrantKey := os.Getenv("QDRANT_API_KEY")

	if qdrantURL == "" {
		return nil, fmt.Errorf("qdrant url is empty")
	}

	url := fmt.Sprintf("%s%s", qdrantURL, endpoint)
	
	var reqBody *bytes.Buffer
	if payload != nil {
		jsonPayload, _ := json.Marshal(payload)
		reqBody = bytes.NewBuffer(jsonPayload)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if qdrantKey != "" {
		req.Header.Set("api-key", qdrantKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("qdrant error: %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	return buf.Bytes(), nil
}

// SearchDocuments melakukan semantic search ke Qdrant Vector DB
func SearchDocuments(queryText string, limit int, collectionName string) ([]string, error) {
	if queryText == "" {
		return nil, nil
	}

	qdrantURL := os.Getenv("QDRANT_URL")
	qdrantKey := os.Getenv("QDRANT_API_KEY")

	if qdrantURL == "" || qdrantKey == "" {
		return nil, fmt.Errorf("qdrant credentials not configured")
	}

	// 1. Generate embedding vector
	vector, err := utils.GPTEmbedding(queryText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// 2. Search points in Qdrant
	url := fmt.Sprintf("%s/collections/%s/points/search", qdrantURL, collectionName)
	
	payload := map[string]interface{}{
		"vector": vector,
		"limit":  limit,
		"with_payload": true,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", qdrantKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant api error: status %d", resp.StatusCode)
	}

	var result struct {
		Result []struct {
			Score   float64                `json:"score"`
			Payload map[string]interface{} `json:"payload"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var contexts []string
	for _, hit := range result.Result {
		text, _ := hit.Payload["text"].(string)
		sourceType, _ := hit.Payload["source_type"].(string)
		
		if text != "" {
			contextStr := fmt.Sprintf("Source [%s] (Score: %.4f):\n%s", sourceType, hit.Score, text)
			contexts = append(contexts, contextStr)
		}
	}

	return contexts, nil
}
