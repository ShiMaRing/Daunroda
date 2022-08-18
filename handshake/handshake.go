package handshake

import (
	"fmt"
	"io"
)

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
	//由传递过来的流信息获取handshake
	lengthBuf := make([]byte, 1)
	full, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}
	if full == 0 {
		err := fmt.Errorf("pstrlen cannot be 0")
		return nil, err
	}
	pstrLen := int(lengthBuf[0])
	handshakeBuf := make([]byte, 48+pstrLen)

	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash [20]byte
	var peerId [20]byte
	copy(infoHash[:], handshakeBuf[pstrLen+8:pstrLen+28])
	copy(peerId[:], handshakeBuf[pstrLen+28:pstrLen+48])

	handshake := &Handshake{
		Pstr:     string(handshakeBuf[0:pstrLen]),
		InfoHash: infoHash,
		PeerID:   peerId,
	}
	return handshake, nil
}

// New 创建handshake
func New(infoHash, peerId [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerId,
	}
}
