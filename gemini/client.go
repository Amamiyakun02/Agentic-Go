package gemini

import (
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client membungkus koneksi websocket ke Google Gemini Live API
type Client struct {
	Conn   *websocket.Conn
	mu     sync.Mutex // Proteksi concurrent write ke WebSocket
}

// Connect melakukan dial WSS ke Google Gemini API
func Connect(apiKey string) (*Client, error) {
	url := fmt.Sprintf("wss://generativelanguage.googleapis.com/ws/google.ai.generativelanguage.v1alpha.GenerativeService.BidiGenerateContent?key=%s", apiKey)
	
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gemini wss: %w", err)
	}

	log.Println("[Gemini Live] Terhubung ke Google Gemini Live API.")
	return &Client{Conn: conn}, nil
}

// SendSetup mengirimkan inisialisasi awal ke model Gemini dengan tools declarations
func (c *Client) SendSetup(systemInstruction string, toolDeclarations []map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	generationConfig := map[string]interface{}{
		"response_modalities": []string{"AUDIO", "TEXT"},
		"speech_config": map[string]interface{}{
			"voice_config": map[string]interface{}{
				"prebuilt_voice_config": map[string]interface{}{
					"voice_name": "Aoede",
				},
			},
		},
	}

	setupPayload := map[string]interface{}{
		"setup": map[string]interface{}{
			"model": "models/gemini-3.1-flash-live-preview",
			"system_instruction": map[string]interface{}{
				"parts": []map[string]interface{}{
					{"text": systemInstruction},
				},
			},
			"generation_config": generationConfig,
		},
	}

	// Tambahkan tools jika ada
	if len(toolDeclarations) > 0 {
		setup := setupPayload["setup"].(map[string]interface{})
		setup["tools"] = []map[string]interface{}{
			{"function_declarations": toolDeclarations},
		}
	}

	return c.Conn.WriteJSON(setupPayload)
}

// SendAudio mengirim chunk audio ke Gemini (thread-safe)
func (c *Client) SendAudio(base64Audio string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	payload := map[string]interface{}{
		"realtime_input": map[string]interface{}{
			"media_chunks": []map[string]interface{}{
				{
					"mime_type": "audio/pcm;rate=16000",
					"data":      base64Audio,
				},
			},
		},
	}
	return c.Conn.WriteJSON(payload)
}

// SendToolResponse merespons hasil dari tool execution ke Gemini (thread-safe)
func (c *Client) SendToolResponse(callID, toolName, result string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	payload := map[string]interface{}{
		"tool_response": map[string]interface{}{
			"function_responses": []map[string]interface{}{
				{
					"id":   callID,
					"name": toolName,
					"response": map[string]interface{}{
						"result": result,
					},
				},
			},
		},
	}
	return c.Conn.WriteJSON(payload)
}
