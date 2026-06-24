package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	MongoClient          *mongo.Client
	DB                   *mongo.Database
	UsersCol             *mongo.Collection
	SessionsCol          *mongo.Collection
	MessagesCol          *mongo.Collection
	DevicesCol           *mongo.Collection
	DeviceAppInventoryCol *mongo.Collection
	DocumentsCol         *mongo.Collection
	PADocumentsCol       *mongo.Collection
)

// InitMongoDB inisialisasi koneksi ke MongoDB Atlas
func InitMongoDB() error {
	mongoUser := os.Getenv("MONGO_USER")
	mongoPass := os.Getenv("MONGO_PASS")
	dbName := os.Getenv("MONGO_DB_NAME")

	if mongoUser == "" || mongoPass == "" {
		log.Println("[Warning] MONGO_USER atau MONGO_PASS tidak disetel.")
		return nil
	}

	uri := fmt.Sprintf("mongodb+srv://%s:%s@aimer.wngbxyv.mongodb.net/?retryWrites=true&w=majority&appName=aimer", mongoUser, mongoPass)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("failed to connect to mongodb: %w", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to ping mongodb: %w", err)
	}

	MongoClient = client
	if dbName == "" {
		dbName = "PersonalAssistant" // default fallback
	}
	DB = client.Database(dbName)

	UsersCol = DB.Collection("users")
	SessionsCol = DB.Collection("chatsessions")
	MessagesCol = DB.Collection("messages")
	DevicesCol = DB.Collection("devices")
	DeviceAppInventoryCol = DB.Collection("device_app_inventory")
	DocumentsCol = DB.Collection("documents")
	PADocumentsCol = DB.Collection("pa_documents")

	log.Println("[MongoDB] Berhasil terhubung ke cluster.")
	return nil
}
