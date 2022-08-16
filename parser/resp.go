package parser

import (
	"encoding/binary"
	"fmt"
	"net"
)

// Peer 提供tracker服务器的响应解析
//服务器将会返回相关列表，其中包括了ip以及端口
//一个长连接，每六个字节对应一个peer
type Peer struct {
	Ip   net.IP
	Port uint16
}

// Unmarshal parses peer IP addresses and ports from a buffer
func Unmarshal(peersBin []byte) ([]Peer, error) {
	//服务器响应的bin解析
	const peerSize = 6
	//peersBin代指的是传回的响应的第二个字段，第一个字段为请求重复时间
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		return nil, fmt.Errorf("Received malformed peers")
	}

	peers := make([]Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		var peer Peer
		var tmp []byte
		copy(tmp, peersBin[i*peerSize:(i+1)*peerSize])
		ip := net.IP(tmp[0:4])
		port := binary.BigEndian.Uint16(tmp[4:6])
		peer.Ip = ip
		peer.Port = port
		peers[i] = peer
	}
	return peers, nil

}
