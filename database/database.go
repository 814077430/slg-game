package database

// 注意：MongoDB 代码已屏蔽，使用内存数据库
// 如需启用 MongoDB，取消下方注释并安装 MongoDB

/*
import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	client *mongo.Client
	db     *mongo.Database
}

func InitMongoDB(uri, dbName string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}

	// 测试连接
	if err = client.Ping(ctx, nil); err != nil {
		client.Disconnect(ctx)
		return nil, err
	}

	log.Println("Connected to MongoDB successfully")
	return &Database{
		client: client,
		db:     client.Database(dbName),
	}, nil
}

func (d *Database) GetCollection(collectionName string) *mongo.Collection {
	return d.db.Collection(collectionName)
}

func (d *Database) Client() *mongo.Client {
	return d.client
}

func (d *Database) Disconnect() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return d.client.Disconnect(ctx)
}
*/

// 使用内存数据库
type Database = MemoryDB

// InitMongoDB 返回内存数据库（屏蔽 MongoDB）
func InitMongoDB(uri, dbName string) (*MemoryDB, error) {
	return NewMemoryDB(), nil
}
