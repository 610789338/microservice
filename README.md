# 微服务集群特点
* 高并发 - 服务实例动态扩缩容
* 高可靠 - 服务实例无状态 + 高可靠db集群
* 高可用 - 单个实例crash后其他实例依然可以提供服务
* 服务自动发现 - 基于etcd实现



# 微服务集群进程分类
* business service - 业务微服务（比如聊天，好友，排行榜服务），提供rpc供客户端调用
* gate - 网关微服务，提供消息路由，负载均衡功能
* push - 推送微服务，负责主动推送消息给客户端
* db(etcd/mongo/redis) - etcd负责服务发现，mongo/redis负责状态存储



# rpc调用场景
主要分为3类：
* c2s - client to service
* s2s - service to service
* s2c - service to client


## c2s
整个过程是异步的，大致可以拆分为：
* client to gate -> 客户端序列化rpc消息发送至gate
* gate to service -> gate将消息路由至service
* service to gate -> service执行完逻辑后将结果返回至gate
* gate -> client -> gate将结果透传给client


## s2s
service之间的服务调用流程和c2s类似（其中一个service作为client），不过s2s支持同步异步两种调用方式，同步调用是基于异步实现的，大致步骤为：
* serviceA to gate -> A服务将rpc（带rid）序列化后发送至gate，然后创建channel，并建立rid=>channel缓存，然后协程等待管道消息
* gate -> serviceB -> gate将消息路由至serviceB
* serviceB -> gate -> serviceB执行完逻辑后将结果返回至gate
* gate -> serviceA -> gate将结果透传给serviceA，结果中带有rid，serviceA根据rid去缓存中找到channel，将结果发送给channel，协程得以继续运行


## s2c
service支持主动推送消息给client，由于gate有多份实例，所以需要找到client的tcp连接在哪个gate上，大致步骤为：
* client和gate建立连接后将clientID发送至gate，gate建立clientID=>clientConnID+gateID的映射关系，缓存至redis中
* service推送消息时，根据clientID从redis中拿到clientConnID和gateID，根据gateID找到gate，将消息和clientConnID打包发送至gate
* gate根据clientConnID找到对应client，将消息发送至client

