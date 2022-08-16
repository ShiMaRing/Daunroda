package downloader

import "io"

//尝试与peers建立TCP连接
type Handshake struct {
	Pstr     string   //比特协议 always BitTorrent protocol
	InfoHash [20]byte //文件信息标识
	PeerID   [20]byte //peerId 随机生成的id
}

// Serialize 序列化方法
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49) //其余都是固定字节数（1+8+20+20）
	buf[0] = byte(len(h.Pstr))
	curr := 1
	curr += copy(buf[curr:], h.Pstr)
	curr += copy(buf[curr:], make([]byte, 8))
	curr += copy(buf[curr:], h.InfoHash[:])
	curr += copy(buf[curr:], h.PeerID[:])
	return buf
}

// Read parses a handshake from a stream
func Read(r io.Reader) (*Handshake, error) {
	// Do Serialize(), but backwards
	// ...
	return nil, nil
}
