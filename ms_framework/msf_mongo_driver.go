package ms_framework


import (
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    // "go.mongodb.org/mongo-driver/mongo/readpref"
	"fmt"
	"time"
	"context"
)

var bgCtx = context.Background()

// mongo
type MongoDriver struct {
	mongo *mongo.Client
}

func (m *MongoDriver) Init() {
	if len(GlobalCfg.Mongo) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(bgCtx, 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(GlobalCfg.Mongo))
	if err != nil {
		panic(fmt.Sprintf("mongo client init error %v", err))
	}

	m.mongo = client

	INFO_LOG("mongo client init ok %v", GlobalCfg.Mongo)
}

func (m *MongoDriver) GetMongo() *mongo.Client {
	return m.mongo
}

var mongoDriver *MongoDriver = &MongoDriver{}

func StartMongoDriver() {
	mongoDriver.Init()
}

func StopMongoDriver() {
	if mongoDriver.mongo != nil {
		mongoDriver.mongo.Disconnect(bgCtx)
	}
}

func GetMongo() *mongo.Client {
	return mongoDriver.mongo
}
