package services

import (
	"context"
	"fmt"
	"log"
	"sync"

	"AgenticGo/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// IngestDocumentEmbedding memecah dokumen (chunking), mengambil embedding secara asinkron dari OpenAI,
// dan mengunggahnya ke Qdrant menggunakan eksekusi Paralel (Goroutines).
func IngestDocumentEmbedding(ctx context.Context, documentID, title, content, sourceType, productID string, isPortfolio bool) bool {
	collectionName := "document_chunk"
	if isPortfolio {
		collectionName = "portfolio_document_chunk"
	}

	// 1. Hapus chunks lama untuk dokumen ini (jika ada) menggunakan Qdrant Delete
	// Catatan: Ini harus dikirimkan ke endpoint Qdrant API
	deletePayload := map[string]interface{}{
		"filter": map[string]interface{}{
			"must": []map[string]interface{}{
				{
					"key": "document_id",
					"match": map[string]interface{}{
						"value": documentID,
					},
				},
			},
		},
	}
	_, _ = QdrantRequest("POST", "/collections/"+collectionName+"/points/delete", deletePayload)

	// 2. Text Chunking
	text := content
	if text == "" {
		text = title
	}
	
	chunkSize := 500
	overlap := 100
	var chunks []string

	for i := 0; i < len(text); {
		end := i + chunkSize
		if end > len(text) {
			end = len(text)
		}
		
		chunk := text[i:end]
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		
		if end == len(text) {
			break
		}
		i += (chunkSize - overlap)
	}

	if len(chunks) == 0 {
		chunks = append(chunks, title)
	}

	// 3. Proses Embedding Paralel menggunakan Goroutines
	log.Printf("[RAG] Memproses %d chunk secara paralel untuk %s", len(chunks), title)
	
	type PointStruct struct {
		ID      string                 `json:"id"`
		Vector  []float64              `json:"vector"`
		Payload map[string]interface{} `json:"payload"`
	}

	points := make([]PointStruct, len(chunks))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errorCount := 0

	for idx, chunkText := range chunks {
		wg.Add(1)
		go func(index int, textChunk string) {
			defer wg.Done()
			
			embedText := fmt.Sprintf("Document: %s | Chunk %d/%d:\n%s", title, index+1, len(chunks), textChunk)
			
			// Ambil vektor dari OpenAI
			vector, err := utils.GPTEmbedding(embedText)
			if err != nil || len(vector) == 0 {
				log.Printf("[RAG ERROR] Gagal embedding chunk %d: %v", index, err)
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			// Generate UUID stabil
			chunkUUID := uuid.NewMD5(uuid.NameSpaceDNS, []byte(fmt.Sprintf("doc-%s-chunk-%d", documentID, index))).String()

			point := PointStruct{
				ID:     chunkUUID,
				Vector: vector,
				Payload: map[string]interface{}{
					"document_id": documentID,
					"source_type": sourceType,
					"product_id":  productID,
					"chunk_index": index,
					"text":        textChunk,
					"page_number": 1,
				},
			}

			mu.Lock()
			points[index] = point
			mu.Unlock()

		}(idx, chunkText)
	}

	wg.Wait()

	// Kumpulkan points yang berhasil
	var validPoints []PointStruct
	for _, p := range points {
		if p.ID != "" {
			validPoints = append(validPoints, p)
		}
	}

	if len(validPoints) == 0 {
		log.Printf("[RAG ERROR] Tidak ada chunk yang berhasil diekstrak untuk dokumen %s", documentID)
		return false
	}

	// 4. Batch Upsert ke Qdrant
	upsertPayload := map[string]interface{}{
		"points": validPoints,
	}
	
	_, err := QdrantRequest("PUT", "/collections/"+collectionName+"/points", upsertPayload)
	if err != nil {
		log.Printf("[RAG ERROR] Gagal menyimpan ke Qdrant: %v", err)
		return false
	}

	// 5. Update MongoDB
	col := DocumentsCol
	if isPortfolio {
		col = PADocumentsCol
	}

	// Parsing string ID to ObjectID
	oid, err := bson.ObjectIDFromHex(documentID)
	if err == nil {
		col.UpdateOne(ctx, bson.M{"_id": oid}, bson.M{
			"$set": bson.M{
				"chunk_count":    len(chunks),
				"qdrant_indexed": true,
			},
		})
	}

	log.Printf("[RAG OK] %s (%s) di-index ke Qdrant dengan %d chunks", title, documentID, len(validPoints))
	return true
}
