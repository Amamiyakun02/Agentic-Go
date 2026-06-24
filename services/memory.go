package services

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetAppInventory mengambil inventory app dari MongoDB
func GetAppInventory(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	var doc struct {
		Apps []map[string]interface{} `bson:"apps"`
	}
	err := DeviceAppInventoryCol.FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return doc.Apps, nil
}

// GetUserProfile mengambil profil pengguna terdaftar
func GetUserProfile(ctx context.Context, userID string) (name, phone, email string) {
	if userID == "" || userID == "guest_live" {
		return "", "", ""
	}

	var user struct {
		Name  string  `bson:"name"`
		Email string  `bson:"email"`
		Phone *string `bson:"phone"`
	}

	// Coba cari sebagai string atau ObjectID
	oid, err := bson.ObjectIDFromHex(userID)
	filter := bson.M{"_id": userID}
	if err == nil {
		filter = bson.M{"_id": oid}
	}

	if err := UsersCol.FindOne(ctx, filter).Decode(&user); err == nil {
		p := ""
		if user.Phone != nil {
			p = *user.Phone
		}
		return user.Name, p, user.Email
	}

	return "", "", ""
}

// GetChatHistory merangkum riwayat pesan
func GetChatHistory(ctx context.Context, sessionID string, limit int64) string {
	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: -1}}).SetLimit(limit)
	cursor, err := MessagesCol.Find(ctx, bson.M{"session_id": sessionID}, opts)
	if err != nil {
		return "(Riwayat kosong)"
	}
	defer cursor.Close(ctx)

	var msgs []struct {
		Sender    string    `bson:"sender"`
		Content   string    `bson:"content"`
		Timestamp time.Time `bson:"timestamp"`
	}
	if err := cursor.All(ctx, &msgs); err != nil {
		return "(Gagal memuat riwayat)"
	}

	if len(msgs) == 0 {
		return "No conversation yet."
	}

	// Reverse array karena diambil dari terbaru (desc)
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	var history string
	for _, m := range msgs {
		sender := "User"
		if m.Sender != "user" {
			sender = "Agent"
		}
		timeStr := m.Timestamp.Format("15:04:05")
		history += fmt.Sprintf("[%s] %-10s: %s\n", timeStr, sender, m.Content)
	}

	return history
}

// GetOrCreateSession mengembalikan ID session atau membuat baru jika belum ada.
// Menggunakan Redis sebagai cache cepat (Fire-and-forget logic untuk write)
func GetOrCreateSession(ctx context.Context, userID, sessionID string) string {
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}

	// Coba cek di Redis (sangat cepat)
	if RedisClient != nil {
		if val, err := RedisClient.Get(ctx, "session:"+sessionID).Result(); err == nil && val != "" {
			return sessionID
		}
	}

	// Jika tidak ada di Redis, cek MongoDB
	var session struct {
		ID string `bson:"session_id"`
	}
	err := SessionsCol.FindOne(ctx, bson.M{"session_id": sessionID}).Decode(&session)
	
	if err != nil {
		// Buat sesi baru
		newSession := bson.M{
			"session_id": sessionID,
			"user_id":    userID,
			"created_at": time.Now().UTC(),
			"updated_at": time.Now().UTC(),
		}
		SessionsCol.InsertOne(ctx, newSession)
	}

	// Fire-and-forget: Simpan ke Redis di background tanpa memblokir HTTP/WebSocket
	if RedisClient != nil {
		go func(sid string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			RedisClient.Set(bgCtx, "session:"+sid, "active", 24*time.Hour)
		}(sessionID)
	}

	return sessionID
}

// SaveMessage menyimpan pesan ke MongoDB dan memicu invalidasi cache jika perlu.
// Fungsi ini berjalan secara asinkron (Fire-and-forget) agar Gemini Streaming tidak terhenti/lag.
func SaveMessage(sessionID, sender, content string) {
	// Lakukan di goroutine
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		msg := bson.M{
			"session_id": sessionID,
			"sender":     sender,
			"content":    content,
			"timestamp":  time.Now().UTC(),
		}
		MessagesCol.InsertOne(bgCtx, msg)
		
		// Opsional: Jika kita cache riwayat obrolan di Redis, kita bisa invalidate di sini:
		if RedisClient != nil {
			RedisClient.Del(bgCtx, "history:"+sessionID)
		}
	}()
}

