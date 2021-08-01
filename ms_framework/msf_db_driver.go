package ms_framework


import (
    "github.com/go-redis/redis"
    "fmt"
    "time"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
    // "go.mongodb.org/mongo-driver/mongo/readpref"
    "context"
)


func CeateRedisSingle(addr []string) *redis.Client {
    client := redis.NewClient(&redis.Options{
        Addr:                addr[0],
        DB:                  15,
        DialTimeout:         10 * time.Second,
        ReadTimeout:         30 * time.Second,
        WriteTimeout:        30 * time.Second,
        PoolSize:            10,
        PoolTimeout:         30 * time.Second,
        IdleTimeout:         500 * time.Millisecond,
        IdleCheckFrequency:  500 * time.Millisecond,
    })
    
    _, err := client.Ping().Result()
    if err != nil {
        panic(fmt.Sprintf("redis client init error %v", err))
    }

    INFO_LOG("create redis client ok %v", addr)

    return client
}

func CreateRedisCluster(addr []string) *redis.ClusterClient {
    client := redis.NewClusterClient(&redis.ClusterOptions{
        Addrs:              addr,
        DialTimeout:        10 * time.Second,
        ReadTimeout:        30 * time.Second,
        WriteTimeout:       30 * time.Second,
        PoolSize:           10,
        PoolTimeout:        30 * time.Second,
        IdleTimeout:        500 * time.Millisecond,
        IdleCheckFrequency: 500 * time.Millisecond,
    })

    err := client.Ping().Err()
    if err != nil {
        panic(fmt.Sprintf("redis cluster init error %v", err))
    }

    INFO_LOG("create redis cluster client ok %v", addr)

    return client
}

func CreateMongo (addr []string) *mongo.Client {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(addr[0]))
    if err != nil {
        panic(fmt.Sprintf("mongo client init error %v", err))
    }

    INFO_LOG("create mongo client ok %v", addr[0])

    return client
}

var gDBResource map[string]interface{} = make(map[string]interface{})
func CreateDBResource() {
    if len(GlobalCfg.DbCfg) == 0 {
        return
    }

    for _, dbCfg := range(GlobalCfg.DbCfg) {
        if dbCfg.Type == "redisSingle" {
            gDBResource[dbCfg.Name] = CeateRedisSingle(dbCfg.Addr)
        } else if dbCfg.Type == "redisCluster" {
            gDBResource[dbCfg.Name] = CreateRedisCluster(dbCfg.Addr)
        } else if dbCfg.Type == "mongo" {
            gDBResource[dbCfg.Name] = CreateMongo(dbCfg.Addr)
        }
    }
}

func ReleaseDBResource() {
    for _, dbCfg := range(GlobalCfg.DbCfg) {
        if dbCfg.Type == "redisSingle"  {
            v, ok := gDBResource[dbCfg.Name]
            if ok {
                client := v.(*redis.Client)
                client.Close()
            }

        } else if dbCfg.Type == "redisCluster" {
            v, ok := gDBResource[dbCfg.Name]
            if ok {
                client := v.(*redis.ClusterClient)
                client.Close()
            }

        } else if dbCfg.Type == "mongo" {
            v, ok := gDBResource[dbCfg.Name]
            if ok {
                client := v.(*mongo.Client)
                client.Disconnect(context.Background())
            }
        }
    }

    gDBResource = nil
}

func GetRedisSingle(name string) *redis.Client {
    v, ok := gDBResource[name]
    if !ok {
        return nil
    }

    return v.(*redis.Client)
}

func GetRedisCluster(name string) *redis.ClusterClient {
    v, ok := gDBResource[name]
    if !ok {
        return nil
    }

    return v.(*redis.ClusterClient)
}

func GetMongo(name string) *mongo.Client {
    v, ok := gDBResource[name]
    if !ok {
        return nil
    }

    return v.(*mongo.Client)
}
