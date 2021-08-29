# 集群进程类型简介
微服务集群由若干个进程组成，进程按类型划分如下：
* service - 业务微服务（比如聊天，好友，排行榜服务）
* gate - 网关微服务，提供消息路由，负载均衡功能，分为内网网关和外网网关
* etcd - 负责服务发现
* db - 数据库层，用于状态存储

网络连接关系：
* service -> etcd&&db
* gate -> etcd&&db
* gate -> service
* client -> gate

# 集群特点
集群所有进程作为一个整体向外界提供服务，具有以下特点：
* 高并发 - 服务实例理论上可无限横向拓展
* 高可靠 - 服务实例无状态 + 高可靠db集群
* 高可用 - 单个实例crash其他实例可以继续提供服务
* 高可维护 - 基于etcd的动态服务发现使得可以对服务实例自由扩缩容

# 框架介绍 - msf_framework
提供公共功能给微服务使用，几个关键的模块：
* msf_tcp_server.go
   利用golang的协程实现的io多路复用，一个客户端连接对应一个golang协程处理网络包收发，负责应用层message的拆包粘包以及tcp连接的心跳管理
   message是CS单次通信的基础单位，格式是：4Byte长度 + rpcName + rpcArgs（用msgpack序列化）

* msf_remote_mgr.go
   gate作为tcp client和每个service建立连接，通过remote_mgr来管理到tcp server的连接，负责应用层的拆包粘包

* msf_simple_rpc.go
   简单rpc框架，服务间通信基石，message刨去长度信息就是rpc的内容，单条rpc请求的处理过程是：
   1、反序列化得到rpcName
   2、根据rpcName找到rpcHandler
   3、利用golang的reflect+struct来反序列化rpcArgs
   4、根据rpcName和rpcArgs进行process
   每个rpc请求新起一个协程来处理，协程数量就是正在处理的rpc数量
   **参见rpc释义**

* msf_callback_mgr.go
   异步应答管理，用于client->gate以及gate->service的异步请求应答：
   * 在client侧建立Rid<=>callack的映射关系，缓存在callback mgr中
   * 在gate侧建立GRid<=>(client, Rid)的映射关系，缓存在callback mgr中

   利用golang协程对每个callback做超时监控，若请求长时间未应答则触发超时应答，避免请求方阻塞

* msf_load_balancer.go
   负载均衡管理器，提供多种负载均衡策略（暂时只有随机和轮询），目前用于c2s和s2s（见流程拆解）的负载均衡

* msf_flow_velocity_counter.go
   流速统计器，目前用于统计client的请求速度以及gate和service的rpc处理速度

* msf_xxx_driver.go
   第三方进程的clientsdk，目前支持etcd driver/redis driver/mongo driver


# 部分rpc释义
* MSG_C2G_RPC_ROUTE
   客户端到微服务的rpc请求路由，参数InnerRpc是业务层rpc序列化后的内容
   gate收到这条rpc后根据参数Namespace和ServiceName选择目标service，然后将InnerRpc和Rid打包成MSG_G2S_RPC_CALL发往目标service

* MSG_G2S_RPC_CALL
   业务rpc请求，和MSG_C2G_RPC_ROUTE一样都是双层rpc结构，handler内部对参数InnerRpc反序列化，然后调用真正的业务层rpc

* MSG_S2G_RPC_RSP
   service处理完rpc请求后给客户端的应答，通过gate将处理结果透传给client

* MSG_G2C_RPC_RSP
   gate转发给client的rpc应答，client根据Rid调用对应的callback

**以上4条rpc是c2s的全流程，详见流程拆解**

* MSG_HEART_BEAT_REQ/MSG_HEART_BEAT_RSP
   tcp连接的心跳检测：
   若tcp client在10s内没有向tcp server发送数据包，则tcp server主动往tcp client发送MSG_HEART_BEAT_REQ，接着tcp client回应MSG_HEART_BEAT_RSP，连接保持
   若tcp client在20s内没有向tcp server发送数据包，则tcp server主动断开连接

* MSG_C2G_AUTH
   client和gate的建立连接后的校验功能，对于部分不可信的client，gate要开启连接鉴权，并对rpc调用做权限管理

* MSG_S2G_RPC_ACCESS_REPORT
   service向gate汇报接口访问权限，部分接口不允许外网网关调用


**以下5条rpc和s2c流程有关，详见流程拆解**
* MSG_GATE_LOGIN/MSG_GATE_LOGOFF
   这两条rpc是指游戏业务中玩家在微服务的登陆登出操作，主要用于微服务向玩家定点推送消息

* MSG_P2G_REQ_LISTENADDR/MSG_G2P_RSP_LISTENADDR
   推送微服务请求gate的监听地址（监听地址作为gate的唯一标识）

* MSG_S2P_PUSH/MSG_P2G_PUSH/MSG_G2C_PUSH
   s2c主流程相关的三条rpc

* MSG_PUSH_REPLY/MSG_PUSH_RESTORE
   safe push相关的两条rpc，client收到推送消息后给予应答，以及推送目标上线后的推送消息恢复


# rpc调用流程拆解
调用场景主要分为3类：
* c2s - client to service
* s2s - service to service
* s2c - service to client

### c2s
client向微服务集群发起rpc请求，整个过程是异步的，分为以下几步：
* client to gate
   客户端序列化rpc消息发送至gate，rpc中包含Rid，用来唯一标识一次请求，client本地缓存Rid<=>callback的映射关系
* gate to service
   gate将消息路由至service，gate根据rpc中的service标识进行路由，对rpc消息进行重组打包发往service
   rpc中包含GRid，用来唯一标识一次路由，gate本地缓存GRid<=>(Rid+client)的映射关系
* service to gate
   service执行完逻辑后将结果和GRid返回给gate
* gate -> client
   gate收到结果后根据GRid拿到Rid和client，将结果和Rid透传给client，client根据Rid调用callback

### s2s
service之间的服务调用流程和c2s类似（其中一个service作为client），不过s2s支持同步异步两种调用方式，同步调用是基于异步实现的，步骤为：
* serviceA to gate
   serviceA将rpc序列化后发送至gate，然后创建channel，本地缓存Rid<=>channel映射关系，然后协程等待channel消息
* gate -> serviceB
   同c2s
* serviceB -> gate
   同c2s
* gate -> serviceA
   gate将结果透传给serviceA，结果中带有Rid，serviceA根据Rid找到channel，将结果发送给channel，协程得以继续运行


### s2c
service支持主动推送消息给client，由于gate有多份实例，所以需要找到client的tcp连接在哪个gate上，大致步骤为：
* client在gate上登陆时将clientID发送至gate，gate建立clientID<=>(clientConnID+gateID)的映射关系，缓存至redis中
* service推送消息时，根据clientID从redis中拿到clientConnID和gateID，根据gateID找到gate，将消息和clientConnID打包发送至gate
* gate根据clientConnID找到对应的client，将消息发送至client

注：
这里的client特指游戏中的对象，所以存在多个client共用一条tcp连接的情况
比如两个玩家在同一个游戏服务器进程上，游戏服务器进程作为client和gate就只有一条tcp连接
目前推送微服务（push service）负责上诉流程，其他service通过s2s调用push service接口来实现推送功能

# 服务发现
### etcd
集群采用etcd做服务发现
* service启动时向etcd注册带租约的key，所有service的key前缀统一，后缀为serviceName和ip:port信息
* gate启动时向etcd获取前缀下的所有key（service列表），解析出ip:port逐一建立连接，并watch该前缀做到实时感知service上下线

### gate侧的处理
在gate侧，service上线的场景比较单一，只有etcd通知，但是service下线的场景比较复杂，分以下三种情况：
* service主动断开连接
   此时gate能主动感知，也能watch到etcd的delete事件，服务可正常下线
* service <-> etcd之间的租约到期
   比如service满负荷情况下未及时向etcd续约，或者service <-> etcd之间的网络路由故障
   此时gate能收到etcd的delete事件，服务可正常下线
* service <-> gate之间的连接失去响应
   比如service <-> gate之间的网络路由故障，此时gate无法主动感知，也不会收到etcd的delete事件
   所以在gate侧对service做活性检测（利用msf_tcp_server的heartbeat），若service长时间未响应则将其下线

### 保障机制
压测时发现当service负荷较大时会导致其在etcd的key租约过期，从而导致gate接收到service下线通知，此时service虽然运行正常却无法提供服务
为避免这种情况，在service启动后增加租约监控，定时检测，若发现租约过期则进行重新注册
在gate侧也定时拉取所有service列表和本地service连接做对比，判断是否有漏连的service

# 推送服务
基于s2c流程提供unsafe和safe两种推送服务

### unsafe push
  不关心消息是否已经正确到达目标，一般用于非关键消息的推送

### safe push
  通过在s2c流程基础上增加消息应答来确保消息被目标接收，具体步骤为：
* 消息推送初始将消息内容，目标ID存入mongo，状态标为UNARRIVE
* 接收方收到推送消息后，给推送方回应
* 推送方收到回应后，将mongo的记录状态修改为ARRIVED
* 目标上线时根据自己ID去mongo查询是否有状态为UNARRIVE的记录，若有则继续走safe push流程

关键消息采用safe push保证消息必达，但safe push存在重复到达的可能，接收方需自行保证消息幂等性
