package ms_framework

import (
    "time"
    "errors"
    "sync"
    "net"
    "fmt"
    "strings"
)

func ReadInt8(buf []byte) int8 {
    return int8(ReadUint8(buf))
}

func ReadUint8(buf []byte) uint8 {
    return uint8(buf[0])
}

func ReadInt16(buf []byte) int16 {
    return int16(ReadUint16(buf))
}

func ReadUint16(buf []byte) uint16 {
    return (uint16(buf[0]) << 8) | uint16(buf[1])
}

func ReadInt(buf []byte) int {
    return int(ReadUint32(buf))
}

func ReadInt32(buf []byte) int32 {
    return int32(ReadUint32(buf))
}

func ReadUint32(buf []byte) uint32 {
    return (uint32(buf[0]) << 24) |
        (uint32(buf[1]) << 16) |
        (uint32(buf[2]) << 8) |
        uint32(buf[3])
}

func ReadInt64(buf []byte) int64 {
    return int64(ReadUint64(buf))
}

func ReadUint64(buf []byte) uint64 {
    return (uint64(buf[0]) << 56) |
        (uint64(buf[1]) << 48) |
        (uint64(buf[2]) << 40) |
        (uint64(buf[3]) << 32) |
        (uint64(buf[4]) << 24) |
        (uint64(buf[5]) << 16) |
        (uint64(buf[6]) << 8) |
        uint64(buf[7])
}

// func ReadFloat32(buf []byte) float32 {
//     return math.Float32frombits(buf[:4])
// }

// func ReadFloat64(buf []byte) float64 {
//     return math.Float32frombits(buf[:8])
// }

func ReadString(buf []byte) (string, error) {
    idx := -1
    for i, buf := range buf {
        if 0 == int8(buf) {
            idx = i
            break
        }
    }

    if -1 == idx {
        return "", errors.New("error string: without end byte")
    }

    return string(buf[:idx]), nil
}

func WriteInt8(buf []byte, v int8) {
    WriteUint8(buf, uint8(v))
}

func WriteUint8(buf []byte, v uint8) {
    buf[0] = byte(v & 0xFF)
}

func WriteInt16(buf []byte, v int16) {
    WriteUint16(buf, uint16(v))
}

func WriteUint16(buf []byte, v uint16) {
    buf[0] = byte(v >> 8 & 0xFF)
    buf[1] = byte(v & 0xFF)
}

func WriteInt(buf []byte, v int) {
    WriteUint32(buf, uint32(v))
}

func WriteInt32(buf []byte, v int32) {
    WriteUint32(buf, uint32(v))
}

func WriteUint32(buf []byte, v uint32) {
    buf[0] = byte(v >> 24 & 0xFF)
    buf[1] = byte(v >> 16 & 0xFF)
    buf[2] = byte(v >> 8 & 0xFF)
    buf[3] = byte(v & 0xFF)
}

func WriteInt64(buf []byte, v int64) {
    WriteUint64(buf, uint64(v))
}

func WriteUint64(buf []byte, v uint64) {
    buf[0] = byte(v >> 56 & 0xFF)
    buf[1] = byte(v >> 48 & 0xFF)
    buf[2] = byte(v >> 40 & 0xFF)
    buf[3] = byte(v >> 32 & 0xFF)
    buf[4] = byte(v >> 24 & 0xFF)
    buf[5] = byte(v >> 16 & 0xFF)
    buf[6] = byte(v >> 8 & 0xFF)
    buf[7] = byte(v & 0xFF)
}

func WriteString(buf []byte, v string) {
    copy(buf, v)
    buf[len(v)] = 0
}

func GetNowTimestamp() int64 {
    return time.Now().Unix()
}

func GetNowTimestampMs() int64 {
    return time.Now().UnixNano() / 1e6
}

// 单个实例唯一的gid
var globalGID uint32 = 0
var gidMutex sync.Mutex

func GenGid() uint32 {
    var ret uint32
    gidMutex.Lock()
    globalGID += 1
    ret = globalGID
    gidMutex.Unlock()

    return ret
}


var localIPLock sync.Mutex
var outIPLock sync.Mutex
var localIP string
var outBoundIP string

// func GetLocalIP() string {

//     if len(localIP) != 0 {
//         return localIP
//     }

//     localIPLock.Lock()
//     defer localIPLock.Unlock()

//     addrs, err := net.InterfaceAddrs()
//     if err != nil {
//         panic(fmt.Sprintf("net.InterfaceAddrs err %v", err))
//     }

//     for _, address := range addrs {
//         if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
//             if ipnet.IP.To4() != nil && ipnet.IP.String() != GetOutBoundIP() {
//                 localIP = ipnet.IP.String()
//             }
//         }
//     }

//     INFO_LOG("GetLocalIP %v", localIP)
//     return localIP
// }

// func GetOutBoundIP() string  {

//     if len(outBoundIP) != 0 {
//         return outBoundIP
//     }

//     outIPLock.Lock()
//     defer outIPLock.Unlock()

//     conn, err := net.DialTimeout("tcp", "golang.org:http", time.Second)
//     if err != nil {
//         if e, ok := err.(*net.OpError); ok && e.Timeout() == true {
//             return ""
//         } else {
//             panic(fmt.Sprintf("GetOutBoundIP err %v", err))
//         }
//     }

//     outBoundIP = strings.Split(conn.LocalAddr().String(), ":")[0]
//     return outBoundIP
// }

func GetLocalIP() string  {

    if len(localIP) != 0 {
        return localIP
    }

    localIPLock.Lock()
    defer localIPLock.Unlock()

    conn, err := net.Dial("tcp", GlobalCfg.Etcd[0])
    if err != nil {
        if e, ok := err.(*net.OpError); ok && e.Timeout() == true {
            return ""
        } else {
            panic(fmt.Sprintf("GetLocalIP err %v", err))
        }
    }

    localIP = strings.Split(conn.LocalAddr().String(), ":")[0]
    return localIP
}

func DeclareHook(args ...interface{}) {
    // for compile error: declared and not use 
}