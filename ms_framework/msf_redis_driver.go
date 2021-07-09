package ms_framework


import (
    "github.com/go-redis/redis"
    "fmt"
    "time"
)

// redis
type RedisDriver struct {
    redis *redis.Client
}

func (r *RedisDriver) Init() {
    if len(GlobalCfg.Redis) == 0 {
        return
    }

    r.redis = redis.NewClient(&redis.Options{
        Addr:                GlobalCfg.Redis,
        DB:                  15,
        DialTimeout:         10 * time.Second,
        ReadTimeout:         30 * time.Second,
        WriteTimeout:        30 * time.Second,
        PoolSize:            10,
        PoolTimeout:         30 * time.Second,
        IdleTimeout:         500 * time.Millisecond,
        IdleCheckFrequency:  500 * time.Millisecond,
    })
    
    _, err := r.redis.Ping().Result()
    if err != nil {
        panic(fmt.Sprintf("redis client init error %v", err))
    }

    INFO_LOG("redis client init ok %v", GlobalCfg.Redis)
}

func (rc *RedisDriver) GetRedis() *redis.Client {
    return rc.redis
}

// redis cluster
type RedisClusterDriver struct {
    redisCluster *redis.ClusterClient
}

func (rc *RedisClusterDriver) Init() {
    if len(GlobalCfg.RedisCluster) == 0 {
        return
    }

    rc.redisCluster = redis.NewClusterClient(&redis.ClusterOptions{
        Addrs:              GlobalCfg.RedisCluster,
        DialTimeout:        10 * time.Second,
        ReadTimeout:        30 * time.Second,
        WriteTimeout:       30 * time.Second,
        PoolSize:           10,
        PoolTimeout:        30 * time.Second,
        IdleTimeout:        500 * time.Millisecond,
        IdleCheckFrequency: 500 * time.Millisecond,
    })

    err := rc.redisCluster.Ping().Err()
    if err != nil {
        panic(fmt.Sprintf("redis cluster init error %v", err))
    }

    INFO_LOG("redis cluster init ok %v", GlobalCfg.RedisCluster)
}

func (rc *RedisClusterDriver) RedisCluster() *redis.ClusterClient {
    return rc.redisCluster
}

var redisDriver *RedisDriver = &RedisDriver{}
var redisClusterDriver *RedisClusterDriver = &RedisClusterDriver{}

func StartRedisDriver() {
    redisDriver.Init()
    redisClusterDriver.Init()
}

func StopRedisDriver() {
    if redisDriver.redis != nil {
        redisDriver.redis.Close()
    }
    if redisClusterDriver.redisCluster != nil {
        redisClusterDriver.redisCluster.Close()
    }
}

func GetRedis() *redis.Client {
    return redisDriver.redis
}

func GetRedisCluster() *redis.ClusterClient {
    return redisClusterDriver.redisCluster
}
