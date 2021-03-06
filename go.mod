module microservice

go 1.14

replace ms_framework => ./ms_framework

replace clientsdk => ./clientsdk

require (
    clientsdk v0.0.0-00010101000000-000000000000
    github.com/cncf/udpa/go v0.0.0-20201120205902-5459f2c99403 // indirect
    github.com/coreos/etcd v3.3.25+incompatible // indirect
    github.com/coreos/go-semver v0.3.0 // indirect
    github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf // indirect
    github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
    github.com/envoyproxy/go-control-plane v0.9.5 // indirect
    github.com/go-redis/redis v6.15.9+incompatible // indirect
    github.com/gogo/protobuf v1.3.2 // indirect
    github.com/golang/protobuf v1.4.2 // indirect
    github.com/google/uuid v1.2.0 // indirect
    github.com/vmihailenco/msgpack v4.0.4+incompatible
    go.etcd.io/etcd v3.3.25+incompatible // indirect
    go.mongodb.org/mongo-driver v1.5.3
    go.uber.org/zap v1.17.0 // indirect
    google.golang.org/grpc v1.26.0 // indirect
    ms_framework v0.0.0-00010101000000-000000000000
)
