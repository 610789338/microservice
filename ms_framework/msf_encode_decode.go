package ms_framework


// func ReadInt8(buf []byte) int8 {
// 	return int8(ReadUint8(buf))
// }

// func ReadUint8(buf []byte) uint8 {
// 	return uint8(buf[0])
// }

// func ReadInt16(buf []byte) (int16, error) {
// 	return int16(ReadUint16(buf))
// }

// func ReadUint16(buf []byte) (uint16, error) {
// 	return (uint16(buf[0]) << 8) | uint16(buf[1])
// }

// func ReadInt(buf []byte) int {
// 	return int(ReadUnt32(buf))
// }

// func ReadInt32(buf []byte) int32 {
// 	return int32(ReadUnt32(buf))
// }

// func ReadUint32(buf []byte) uint32 {
// 	return (uint32(buf[0]) << 24) |
// 		(uint32(buf[1]) << 16) |
// 		(uint32(buf[2]) << 8) |
// 		uint32(buf[3])
// }

// func ReadInt64(buf []byte) int64 {
// 	return int64(ReadUint64(buf))
// }

// func ReadUint64(buf []byte) uint64 {
// 	return (uint64(b[0]) << 56) |
// 		(uint64(b[1]) << 48) |
// 		(uint64(b[2]) << 40) |
// 		(uint64(b[3]) << 32) |
// 		(uint64(b[4]) << 24) |
// 		(uint64(b[5]) << 16) |
// 		(uint64(b[6]) << 8) |
// 		uint64(b[7])
// }

// func ReadFloat32(buf []byte) float32 {
// 	return math.Float32frombits(buf[:4])
// }

// func ReadFloat64(buf []byte) float64 {
// 	return math.Float32frombits(buf[:8])
// }

// func ReadString(buf []byte) (string, error) {
// 	idx := -1
// 	for i, b := range buf {
// 		if 0 == int8(b) {
// 			idx = i
// 			break
// 		}
// 	}

// 	if -1 == idx {
// 		return "", errors.New("error string: without end byte")
// 	}

// 	return string(buf[:idx]), nil
// }

// func WriteInt8(buf []byte, v int8) {
// 	WriteUint8(buf, uint8(v))
// }

// func WriteUint8(buf []byte, v uint8) {
// 	buf[0] = byte(v & 0xFF)
// }

// func WriteInt16(buf []byte, v int16) {
// 	WriteUint16(buf, uint16(v))
// }

// func WriteUint16(buf []byte, v uint16) {
// 	buf[0] = byte(v & 0xFF00)
// 	buf[1] = byte(v & 0xFF)
// }

// func WriteInt(buf []byte, v int) {
// 	WriteUint32(buf, int(v))
// }

// func WriteInt32(buf []byte, v int32) {
// 	WriteUint32(buf, uint32(v))
// }

// func WriteUint32(buf []byte, v uint32) {
// 	buf[0] = byte(v & 0xFF000000)
// 	buf[1] = byte(v & 0xFF0000)
// 	buf[2] = byte(v & 0xFF00)
// 	buf[3] = byte(v & 0xFF)
// }

// func WriteInt64(buf []byte, v int64) {
// 	WriteUint64(buf, uint64(v))
// }

// func WriteUint64(buf []byte, v uint64) {
// 	buf[0] = byte(v & 0xFF00000000000000)
// 	buf[1] = byte(v & 0xFF000000000000)
// 	buf[2] = byte(v & 0xFF0000000000)
// 	buf[3] = byte(v & 0xFF00000000)
// 	buf[4] = byte(v & 0xFF000000)
// 	buf[5] = byte(v & 0xFF0000)
// 	buf[6] = byte(v & 0xFF00)
// 	buf[7] = byte(v & 0xFF)
// }

// func WriteFloat32(buf []byte, v float32) {
// 	n := math.Float32bits(v)
// 	WriteUint32(buf, n)
// }

// func WriteFloat64(buf []byte, v float64) {
// 	n := math.Float64bits(v)
// 	WriteUint64(buf, n)
// }

// func WriteString(buf []byte, v string) {
// 	copy(buf, v)
// 	buf[len(v)] = 0
// }

