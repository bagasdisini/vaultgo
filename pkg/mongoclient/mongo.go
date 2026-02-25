package mongoclient

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Connect(uri string, registry *bson.Registry) *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(uri)
	if registry != nil {
		opts.SetRegistry(registry)
	}

	client, err := mongo.Connect(opts)
	if err != nil {
		log.Fatalf("failed to create mongo client: %v", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("failed to ping mongo: %v", err)
	}

	log.Println("connected to MongoDB")
	return client
}
