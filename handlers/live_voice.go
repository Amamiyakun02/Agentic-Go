package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"AgenticGo/gemini"
	"AgenticGo/services"
	"AgenticGo/utils"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Upgrader digunakan untuk mengubah request HTTP menjadi koneksi WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// ============================================================
// Tool Declarations — didaftarkan di sesi awal Gemini Live API
// ============================================================

func BuildToolDeclarations() []map[string]interface{} {
	return []map[string]interface{}{
		// RAG: Pencarian dokumen internal
		{
			"name":        "search_documents",
			"description": "Pencarian ke database dokumen internal (RAG). Gunakan jika pengguna menanyakan informasi tentang panduan, referensi internal, atau portofolio.",
			"parameters": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "STRING",
						"description": "Kata kunci pencarian.",
					},
				},
				"required": []string{"query"},
			},
		},
		// Device: Buka aplikasi
		{
			"name":        "open_app",
			"description": "Buka aplikasi tertentu di HP Android user menggunakan nama package.",
			"parameters": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"package_name": map[string]interface{}{
						"type":        "STRING",
						"description": "Nama package aplikasi Android (contoh: 'com.whatsapp').",
					},
				},
				"required": []string{"package_name"},
			},
		},
		// Device: Klik teks UI
		{
			"name":        "click_text",
			"description": "Simulasikan klik pada teks antarmuka (UI) tertentu di layar HP Android.",
			"parameters": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "STRING",
						"description": "Teks tombol atau elemen UI yang ingin diklik.",
					},
				},
				"required": []string{"text"},
			},
		},
		// Device: Input teks
		{
			"name":        "input_text",
			"description": "Memasukkan teks ke kolom input yang sedang aktif di HP Android.",
			"parameters": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"value": map[string]interface{}{
						"type":        "STRING",
						"description": "Teks yang ingin dimasukkan.",
					},
				},
				"required": []string{"value"},
			},
		},
		// Device: Baca layar
		{
			"name":        "read_screen",
			"description": "Membaca semua teks, tombol, dan elemen UI yang tampil di layar HP Android saat ini.",
			"parameters": map[string]interface{}{
				"type":       "OBJECT",
				"properties": map[string]interface{}{},
			},
		},
		// Device: List apps
		{
			"name":        "list_installed_apps",
			"description": "Melihat daftar aplikasi yang terinstal di HP Android beserta package name-nya.",
			"parameters": map[string]interface{}{
				"type":       "OBJECT",
				"properties": map[string]interface{}{},
			},
		},
		// Spotify: play
		{
			"name":        "spotify_play_music",
			"description": "Memutar musik di Spotify. Bisa dilampirkan query berupa judul lagu.",
			"parameters": map[string]interface{}{
				"type": "OBJECT",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "STRING",
						"description": "Judul lagu, nama artis, atau URI Spotify.",
					},
				},
			},
		},
		// Spotify: pause
		{
			"name":        "spotify_pause_music",
			"description": "Menjeda (Pause) musik yang sedang berputar di Spotify.",
			"parameters": map[string]interface{}{
				"type":       "OBJECT",
				"properties": map[string]interface{}{},
			},
		},
		// Spotify: next
		{
			"name":        "spotify_next_track",
			"description": "Melompati (Next) ke lagu berikutnya di Spotify.",
			"parameters": map[string]interface{}{
				"type":       "OBJECT",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// ============================================================
// Tool Execution — menjalankan tool berdasarkan nama
// ============================================================

func executeTool(toolName string, args map[string]interface{}) string {
	switch toolName {
	case "search_documents":
		query, _ := args["query"].(string)
		if query == "" {
			return "Error: query kosong"
		}
		results, err := services.SearchDocuments(query, 3, "document_chunk")
		if err != nil {
			return fmt.Sprintf("Error mencari dokumen: %v", err)
		}
		if len(results) == 0 {
			return "Data tidak ditemukan."
		}
		return strings.Join(results, "\n")

	default:
		// Semua tool lainnya diteruskan ke MCP server
		mcpURL := os.Getenv("MCP_URL")
		if mcpURL == "" {
			mcpURL = "https://agentmcp-service-579f62a8.fastapicloud.dev/mcp"
		}
		return callMCPTool(mcpURL, toolName, args)
	}
}

func callMCPTool(mcpURL, toolName string, args map[string]interface{}) string {
	payload := map[string]interface{}{
		"tool_name": toolName,
		"arguments": args,
	}
	jsonPayload, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(mcpURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "Error: Failed to connect to MCP Server: " + err.Error()
	}
	defer resp.Body.Close()

	var mcpResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&mcpResp); err != nil {
		return "Error: Failed to parse MCP response"
	}

	if data, ok := mcpResp["data"]; ok {
		dataBytes, _ := json.Marshal(data)
		return string(dataBytes)
	} else if msg, ok := mcpResp["message"].(string); ok {
		return msg
	}
	return "Tool executed, no explicit data returned."
}

// ============================================================
// LiveVoiceHandler — WebSocket endpoint /live-voice
// ============================================================

func LiveVoiceHandler(c *gin.Context) {
	clientWs, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("[Gemini Live] Upgrade failed:", err)
		return
	}
	defer clientWs.Close()

	log.Println("[Gemini Live] Android client terhubung. Menunggu init payload...")

	// 1. Dapatkan API Key
	geminiAPIKey := os.Getenv("GEMINI_API_KEY")
	if geminiAPIKey == "" {
		log.Println("[Gemini Live] GEMINI_API_KEY is not set.")
		clientWs.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "GEMINI_API_KEY is missing"))
		return
	}

	// 2. Tunggu pesan init dari Client Android
	_, initMsg, err := clientWs.ReadMessage()
	if err != nil {
		log.Println("[Gemini Live] Gagal menerima pesan inisialisasi:", err)
		return
	}

	var initData map[string]interface{}
	if err := json.Unmarshal(initMsg, &initData); err != nil {
		log.Println("[Gemini Live] Invalid JSON init:", err)
		return
	}

	userID, _ := initData["user_id"].(string)
	sessionID, _ := initData["session_id"].(string)
	role, _ := initData["role"].(string)

	// Handle session_id null/empty untuk sinkronisasi dengan sesi chat teks
	if sessionID == "" || sessionID == "null" {
		sessionID = ""
	}

	if initData["type"] == "init" {
		log.Printf("[Gemini Live] Init diterima. User: %s, Session: %s\n", userID, sessionID)
	} else {
		log.Println("[Gemini Live] Tidak menerima pesan init yang valid, fallback ke guest.")
		userID = "guest_live"
		sessionID = "live_session_guest"
		role = "customer"
	}

	// 3. Ambil Konteks Sesi, History, User Info dari DB
	ctxDB := context.Background()
	sessionID = services.GetOrCreateSession(ctxDB, userID, sessionID)
	userName, userPhone, userEmail := services.GetUserProfile(ctxDB, userID)
	appInventory, _ := services.GetAppInventory(ctxDB, userID)
	historyMessages := services.GetChatHistory(ctxDB, sessionID, 10)

	// RAG Search awal
	retrieval, _ := services.SearchDocuments("(Sesi Live Voice Dimulai)", 3, "document_chunk")

	// Build system instruction
	instruction := "Anda adalah Vienna. AI Assistant berbasis Gemini. RESPOND IN id-ID. YOU MUST RESPOND UNMISTAKABLY IN id-ID."
	contextPrompt := utils.BuildPrompt(
		sessionID, userID, "VERIFIED", "browsing", "-", "-", "-", "-", historyMessages,
		false, retrieval, "(Sesi Live Voice Dimulai)", role,
		userName, userPhone, userEmail, appInventory,
	)
	systemInstructionText := instruction + "\n\n[CONTEXT DARI SISTEM]\n" + contextPrompt

	// 4. Hubungkan ke Gemini API
	geminiClient, err := gemini.Connect(geminiAPIKey)
	if err != nil {
		log.Println("[Gemini Live] Gagal terhubung ke Gemini:", err)
		notifyClientError(clientWs, "Gagal terhubung ke AI")
		return
	}
	defer geminiClient.Conn.Close()

	// 5. Kirim konfigurasi awal ke Gemini dengan tool declarations
	toolDeclarations := BuildToolDeclarations()
	if err := geminiClient.SendSetup(systemInstructionText, toolDeclarations); err != nil {
		log.Println("[Gemini Live] Gagal mengirim Setup ke Gemini:", err)
		notifyClientError(clientWs, "Gagal konfigurasi AI")
		return
	}

	// 6. Inisialisasi channels dan context untuk koordinasi goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	clientWriteCh := make(chan []byte, 200)     // Channel untuk menulis ke Android
	var clientWriteMu sync.Mutex                // Mutex untuk proteksi write ke clientWs
	var agentTextBuf strings.Builder            // Buffer untuk mengumpulkan teks agent

	// ===== GOROUTINE 1: Client Writer =====
	// Menulis pesan ke Android secara serial dan thread-safe
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-clientWriteCh:
				if !ok {
					return
				}
				clientWriteMu.Lock()
				err := clientWs.WriteMessage(websocket.TextMessage, msg)
				clientWriteMu.Unlock()
				if err != nil {
					log.Println("[Gemini Live] Gagal menulis ke Android:", err)
					cancel()
					return
				}
			}
		}
	}()

	// ===== GOROUTINE 2: Heartbeat =====
	// Menjaga koneksi WebSocket tetap hidup
	go func() {
		ticker := time.NewTicker(25 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pingMsg, _ := json.Marshal(map[string]string{"type": "ping"})
				select {
				case clientWriteCh <- pingMsg:
				default:
					// Channel penuh, koneksi mungkin lambat
					log.Println("[Gemini Live] Heartbeat channel penuh, kemungkinan koneksi lambat")
				}
			}
		}
	}()

	// ===== GOROUTINE 3: Gemini Receiver =====
	// Membaca respons dari Gemini dan meneruskan audio/text/tool calls
	go func() {
		defer cancel() // Batalkan semua goroutine jika Gemini disconnect
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			_, msg, err := geminiClient.Conn.ReadMessage()
			if err != nil {
				log.Println("[Gemini Live] Koneksi dari Gemini API ditutup:", err)
				return
			}

			var responseData map[string]interface{}
			if err := json.Unmarshal(msg, &responseData); err != nil {
				continue
			}

			// ---- Parse serverContent ----
			serverContent := GetMapKey(responseData, "serverContent", "server_content")
			if serverContent != nil {
				modelTurn := GetMapKey(serverContent, "modelTurn", "model_turn")
				if modelTurn != nil {
					if parts, ok := modelTurn["parts"].([]interface{}); ok {
						for _, p := range parts {
							part, ok := p.(map[string]interface{})
							if !ok {
								continue
							}

							// Tangkap Text Response
							if text, ok := part["text"].(string); ok && text != "" {
								agentTextBuf.WriteString(text)
							}

							// Tangkap Audio Response
							if inlineData, ok := part["inlineData"].(map[string]interface{}); ok {
								if audioBase64, ok := inlineData["data"].(string); ok {
									payload, _ := json.Marshal(map[string]string{"audio": audioBase64})
									select {
									case clientWriteCh <- payload:
									case <-ctx.Done():
										return
									}
								}
							}
						}
					}
				}

				// Cek apakah turn selesai
				turnComplete, _ := serverContent["turnComplete"].(bool)
				if !turnComplete {
					turnComplete, _ = serverContent["turn_complete"].(bool)
				}

				if turnComplete && agentTextBuf.Len() > 0 {
					// Fire-and-forget: simpan ke DB di goroutine terpisah
					textToSave := strings.TrimSpace(agentTextBuf.String())
					agentTextBuf.Reset()
					go func(sid, text string) {
						services.SaveMessage(sid, "user", "[Voice Input]")
						services.SaveMessage(sid, "assistant", text)
						log.Printf("[Gemini Live] Voice context disimpan ke DB: session=%s\n", sid)
					}(sessionID, textToSave)
				}
			}

			// ---- Parse toolCall / tool_call ----
			toolCallData := GetMapKey(responseData, "toolCall", "tool_call")
			if toolCallData != nil {
				functionCalls, ok := toolCallData["functionCalls"].([]interface{})
				if !ok {
					functionCalls, _ = toolCallData["function_calls"].([]interface{})
				}

				for _, fcRaw := range functionCalls {
					fc, ok := fcRaw.(map[string]interface{})
					if !ok {
						continue
					}

					toolName, _ := fc["name"].(string)
					callID, _ := fc["id"].(string)
					if callID == "" {
						callID = fmt.Sprintf("live_tool_%d", time.Now().UnixNano())
					}
					args, _ := fc["args"].(map[string]interface{})

					log.Printf("[Gemini Live] Tool call: %s(%v)\n", toolName, args)

					// Eksekusi tool di goroutine terpisah untuk non-blocking
					go func(name, id string, toolArgs map[string]interface{}) {
						resultText := executeTool(name, toolArgs)
						truncated := resultText
						if len(truncated) > 120 {
							truncated = truncated[:120]
						}
						log.Printf("[Gemini Live] Tool result (%s): %s\n", name, truncated)

						if err := geminiClient.SendToolResponse(id, name, resultText); err != nil {
							log.Printf("[Gemini Live] Gagal kirim tool response: %v\n", err)
							cancel()
						}
					}(toolName, callID, args)
				}
			}
		}
	}()

	// ===== MAIN GOROUTINE: Android Receiver =====
	// Membaca audio dari Client Android dan meneruskan ke Gemini
	for {
		select {
		case <-ctx.Done():
			log.Println("[Gemini Live] Context cancelled, menutup koneksi Android.")
			goto cleanup
		default:
		}

		_, msg, err := clientWs.ReadMessage()
		if err != nil {
			log.Println("[Gemini Live] Client Android disconnect:", err)
			break
		}

		var clientInput map[string]interface{}
		if err := json.Unmarshal(msg, &clientInput); err != nil {
			continue
		}

		// Handle pong response dari heartbeat
		if msgType, _ := clientInput["type"].(string); msgType == "pong" {
			continue
		}

		if audioChunk, ok := clientInput["audio"].(string); ok {
			if err := geminiClient.SendAudio(audioChunk); err != nil {
				log.Println("[Gemini Live] Gagal meneruskan audio ke Gemini:", err)
				notifyClientError(clientWs, "Koneksi ke AI terputus")
				break
			}
		}
	}

cleanup:
	cancel() // Membersihkan semua goroutine
	log.Println("[Gemini Live] Sesi selesai dan dibersihkan.")
}

// ============================================================
// Helper Functions
// ============================================================

// GetMapKey mencari key dengan fallback camelCase/snake_case
func GetMapKey(data map[string]interface{}, keys ...string) map[string]interface{} {
	for _, key := range keys {
		if val, ok := data[key].(map[string]interface{}); ok {
			return val
		}
	}
	return nil
}

// notifyClientError mengirim pesan error ke Android sebelum menutup koneksi
func notifyClientError(ws *websocket.Conn, message string) {
	errPayload, _ := json.Marshal(map[string]string{"error": message})
	ws.WriteMessage(websocket.TextMessage, errPayload)
}
