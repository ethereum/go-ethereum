package monitor

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"math/big"
	"sync"
	"time"
)

var DatabaseName = "SystemUsage"
var TransactionCollectionName = "transactions"

//var OperationCollectionName = "operations"

type TransactionSystemUsageRow struct {
	BlockId          int64           `bson:"block_id"`
	TransactionIndex int             `bson:"transaction_index"`
	Operations       []OperationData `bson:"operations"`
	UpdateTime       time.Time       `bson:"update_time"`
}

type Idb interface {
	SaveBlockData(data BlockData) error
	SaveTxData(data TransactionData) error
	GetBlockData(blockId *big.Int) (BlockData, error)
	GetTransactionData(blockId string, transactionIndex int) (TransactionData, error)
	GetOperationData(op string) ([]OperationData, error)
}

type MongoDb struct {
	uri                       string
	client                    mongo.Client
	dbName                    string
	transactionCollectionName string
}

var mongoDb *MongoDb
var once sync.Once

func NewMongoDb(uri string) (*MongoDb, error) {

	once.Do(func() {

		mongoDb = &MongoDb{}
		mongoDb.dbName = DatabaseName
		mongoDb.transactionCollectionName = TransactionCollectionName
		clientOptions := options.Client().ApplyURI(uri)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		client, err := mongo.Connect(ctx, clientOptions)

		if err != nil {
			log.Error("Failed to connect to mongo", err)
		}

		// Check the connection
		err = client.Ping(context.TODO(), nil)

		if err != nil {
			log.Error("Failed to ping mongo", err)
		}

		fmt.Println("Connected to MongoDB!")
		mongoDb.uri = uri
		mongoDb.client = *client
	})
	return mongoDb, nil
}

func (mongoDb *MongoDb) SaveBlockData(data BlockData) error {
	blockId := data.BlockId

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := mongoDb.client.Database(mongoDb.dbName).Collection(TransactionCollectionName)

	for _, transactionData := range data.TransactionDataList {
		transaction := TransactionSystemUsageRow{
			blockId.Int64(),
			transactionData.TransactionIndex,
			transactionData.OperationDataList,
			time.Now(),
		}
		_, err := coll.InsertOne(ctx, transaction)

		if err != nil {
			log.Error("Failed to insert data\n", err)
		}
	}
	log.Info("Finished save block data\n")

	return nil
}

func (mongoDb *MongoDb) SaveTxData(data TransactionData) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := mongoDb.client.Database(mongoDb.dbName).Collection(TransactionCollectionName)

	transaction := TransactionSystemUsageRow{
		big.NewInt(-1).Int64(),
		data.TransactionIndex,
		data.OperationDataList,
		time.Now(),
	}
	_, err := coll.InsertOne(ctx, transaction)

	if err != nil {
		log.Error("Failed to insert data\n", err)
	}
	log.Info("Finished save tx data\n")

	return nil
}

func (mongoDb *MongoDb) GetBlockData(blockId *big.Int) (BlockData, error) {

	blockData := BlockData{TransactionDataList: []TransactionData{}}
	blockData.BlockId = blockId

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	coll := mongoDb.client.Database(mongoDb.dbName).Collection(TransactionCollectionName)

	cur, err := coll.Find(ctx, bson.M{"block_id": blockId})

	if err != nil {
		log.Error("Failed to find block data", err)
		return BlockData{}, err
	}

	for cur.Next(ctx) {

		var elem TransactionData

		err := cur.Decode(&elem)

		if err != nil {
			log.Error("Failed to convert to TransactionData", err)
			return BlockData{}, err
		}

		blockData.TransactionDataList = append(blockData.TransactionDataList, elem)
	}

	return blockData, nil
}

func (mongoDb *MongoDb) GetTransactionData(blockId string, transactionIndex int) (TransactionData, error) {
	return TransactionData{}, nil
}

func (mongoDb *MongoDb) GetOperationData(op string) ([]OperationData, error) {
	return []OperationData{}, nil
}

//func (mongoDb *MongoDb) SaveTransactionData(data TransactionData) error {
//	for _, operationData := range data.OperationDataList {
//		coll := mongoDb.client.Database(mongoDb.dbName).Collection(OperationCollectionName)
//		_, err := coll.InsertOne(context.TODO(), operationData)
//
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
//	return nil
//}
