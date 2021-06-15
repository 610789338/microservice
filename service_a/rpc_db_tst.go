package main


import (
	msf "ms_framework"
	"go.mongodb.org/mongo-driver/bson"
	"context"
)

var bgCtx = context.Background()

type RpcDBTestRsp struct {
	Success 	bool
}

type RpcDBTestHandler struct {
	rsp 	RpcDBTestRsp
}

func (r *RpcDBTestHandler) GetReqPtr() interface{} {return nil}
func (r *RpcDBTestHandler) GetRspPtr() interface{} {return &(r.rsp)}

func (r *RpcDBTestHandler) Process(session *msf.Session) {

	r.RedisTest()
	r.RedisClusterTest()
	r.MongoTest()
	r.rsp.Success = true
}

func (r *RpcDBTestHandler) RedisTest() {

	// redis
	redis := msf.GetRedis()
	if nil == redis {
		msf.ERROR_LOG("redis nil")
		return
	}

	_, err := redis.HMSet("REDIS_KEY_TEST", map[string]interface{}{"field1": 123, "field2": "abc"}).Result()
	if err != nil {
		msf.ERROR_LOG("redis HMSet err %v", err)
		return
	}

	val, err := redis.HMGet("REDIS_KEY_TEST", "field1", "field2").Result()
	if err != nil {
		msf.ERROR_LOG("redis HMGet err %v", err)
		return
	}

	msf.DEBUG_LOG("redis HMGet %+v", val)
}

func (r *RpcDBTestHandler) RedisClusterTest() {

	// redis cluster
	redisCluster := msf.GetRedisCluster()
	if nil == redisCluster {
		msf.ERROR_LOG("redis cluster nil")
		return
	}

	_, err := redisCluster.HMSet("REDIS_KEY_TEST", map[string]interface{}{"field1": 123, "field2": "abc"}).Result()
	if err != nil {
		msf.ERROR_LOG("redis cluster HMSet err %v", err)
		return
	}

	val, err := redisCluster.HMGet("REDIS_KEY_TEST", "field1", "field2").Result()
	if err != nil {
		msf.ERROR_LOG("redis cluster HMGet err %v", err)
		return
	}

	msf.DEBUG_LOG("redis cluster HMGet %+v", val)
}

func (r *RpcDBTestHandler) MongoTest() {
	
	// mongo
	mongo := msf.GetMongo()
	if nil == mongo {
		msf.ERROR_LOG("mongo test nil")
		return
	}

	db := mongo.Database("foo")
	coll := db.Collection("foo")
	doc := bson.D{{"x", 1}}
	_, err := coll.InsertOne(bgCtx, doc)
	if err != nil {
		msf.ERROR_LOG("mongo collection insert test err %v", err)
		return
	}

	// update := bson.D{{"$set", bson.D{{"x", 666}}}}
	// update := bson.D{{"$set", bson.D{{"x", bson.D{{"y", 777}}}}}}
	update := bson.D{{"$set", bson.M{"x": bson.M{"y": 888}}}}
	_, err = coll.UpdateOne(bgCtx, doc, update)
	if err != nil {
		msf.ERROR_LOG("mongo collection update err %v", err)
		return
	}
	
	cursor, err := coll.Find(bgCtx, bson.D{})
	if err != nil {
		msf.ERROR_LOG("mongo collection find err %v", err)
		return
	}

	for cursor.Next(bgCtx) {
		var result bson.D
		err = cursor.Decode(&result)
		if err != nil {
			msf.ERROR_LOG("mongo cursor Decode error %v", err)
		}

		msf.INFO_LOG("mongo find result %+v", result)
	}

	delRet, err := coll.DeleteMany(bgCtx, bson.D{})
	if err != nil {
		msf.ERROR_LOG("mongo collection del err %v", err)
		return
	}

	msf.INFO_LOG("mongo del result %+v", delRet)
}
