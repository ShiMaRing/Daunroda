package parser

import (
	"bitDownloader/peer"
	"encoding/binary"
	"fmt"
	"github.com/jackpal/bencode-go"
	"net"
	"net/http"
	"time"
)

// Unmarshal parses peer IP addresses and ports from a buffer
func Unmarshal(peersBin []byte) ([]peer.Peer, error) {
	//服务器响应的bin解析
	const peerSize = 6
	//peersBin代指的是传回的响应的第二个字段，第一个字段为请求重复时间
	numPeers := len(peersBin) / peerSize
	if len(peersBin)%peerSize != 0 {
		return nil, fmt.Errorf("Received malformed peers")
	}

	peers := make([]peer.Peer, numPeers)
	for i := 0; i < numPeers; i++ {
		var peer peer.Peer
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

//获取到url之后，发起get请求并解析列表
func (t *TorrentFile) requestPeers(peerID [20]byte, port uint16) ([]peer.Peer, error) {
	trackerURL, err := t.buildTrackerURL(peerID, port)
	if err != nil {
		return nil, err
	}

	fmt.Println(trackerURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(trackerURL)
	if err != nil {
		return nil, err
	} //尝试获取信息
	defer resp.Body.Close()

	result := make(map[string]interface{})
	err = bencode.Unmarshal(resp.Body, result)
	if err != nil {
		return nil, err
	}
	return Unmarshal([]byte(result["peers"].(string)))
}
