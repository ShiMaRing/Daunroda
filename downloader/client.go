package downloader

import (
	"bitDownloader/handshake"
	"bitDownloader/peer"
	"bytes"
	"fmt"
	"net"
	"time"
)

// Client 提供tcp连接能力
type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield BitField
	peer     peer.Peer
	InfoHash [20]byte
	peerId   [20]byte
}

//peer 之间进行握手
func completeHandShake(conn net.Conn, infoHash, peerId [20]byte) (*handshake.Handshake, error) {
	//设置握手超时时间为3s
	conn.SetDeadline(time.Now().Add(time.Second * 3))
	defer conn.SetDeadline(time.Time{}) //接除限制

	h := handshake.New(infoHash, peerId)
	_, err := conn.Write(h.Serialize())
	if err != nil {
		return nil, err
	}
	res, err := handshake.Read(conn)
	if err != nil {
		return nil, err
	}
	//检查握手得到的种子哈希值是否正确
	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("Expected infohash %x but got %x", res.InfoHash, infoHash)
	}
	return res, nil
}

//接收bitfield
func recvBitfield(conn net.Conn) (BitField, error) {
	conn.SetDeadline(time.Now().Add(time.Second * 3))
	defer conn.SetDeadline(time.Time{}) //接除限制

	message, err := Read(conn)
	if err != nil {
		return nil, err
	}
	//检查是否为null,可能会接收到keep-alive
	if message == nil {
		err := fmt.Errorf("Expected bitfield but got %s", message)
		return nil, err
	}

	if message.ID != MsgBitfield {
		err := fmt.Errorf("Expected bitfield but got ID %d", message.ID)
		return nil, err
	}

	return message.Payload, nil

}

// New 构建client
func New(peer peer.Peer, peerID, infoHash [20]byte) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), time.Second*3)
	if err != nil {
		return nil, err
	}
	_, err = completeHandShake(conn, infoHash, peerID)
	if err != nil {
		conn.Close()
		return nil, err
	}
	bitfield, err := recvBitfield(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	client := &Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bitfield,
		peer:     peer,
		InfoHash: infoHash,
		peerId:   peerID,
	}
	return client, nil
}

// Read reads and consumes a message from the connection
func (c *Client) Read() (*Message, error) {
	msg, err := Read(c.Conn)
	return msg, err
}

// SendRequest sends a Request message to the peer
func (c *Client) SendRequest(index, begin, length int) error {
	req := FormatRequest(index, begin, length)
	_, err := c.Conn.Write(req.Serialize())
	return err
}

// SendInterested sends an Interested message to the peer
func (c *Client) SendInterested() error {
	msg := Message{ID: MsgInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendNotInterested sends a NotInterested message to the peer
func (c *Client) SendNotInterested() error {
	msg := Message{ID: MsgNotInterested}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendUnchoke sends an Unchoke message to the peer
func (c *Client) SendUnchoke() error {
	msg := Message{ID: MsgUnchoke}
	_, err := c.Conn.Write(msg.Serialize())
	return err
}

// SendHave sends a Have message to the peer
func (c *Client) SendHave(index int) error {
	msg := FormatHave(index)
	_, err := c.Conn.Write(msg.Serialize())
	return err
}
