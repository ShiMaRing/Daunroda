package downloader

import (
	"encoding/binary"
	"fmt"
	"io"
)

//A message has a length, an ID and a payload.
type messageID uint8

const (
	MsgChoke         messageID = 0
	MsgUnchoke       messageID = 1
	MsgInterested    messageID = 2
	MsgNotInterested messageID = 3
	MsgHave          messageID = 4
	MsgBitfield      messageID = 5
	MsgRequest       messageID = 6
	MsgPiece         messageID = 7
	MsgCancel        messageID = 8
)

// Message 中间字段给出message信息
type Message struct {
	ID      messageID
	Payload []byte
}

// <length prefix><message ID><payload>

// Serialize 转化为字节流
func (m *Message) Serialize() []byte {
	if m == nil {
		return make([]byte, 4) //keepalive-message
	}
	length := uint32(len(m.Payload) + 1)

	buf := make([]byte, length+4) //加上前面的四位长度

	//前四位payload长度
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)

	return buf
}

// Read 从网络流中读取数据
func Read(r io.Reader) (*Message, error) {
	var lengthBuf = make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, nil
	}
	length := binary.BigEndian.Uint32(lengthBuf)
	if length == 0 {
		// keep-alive message
		return nil, nil
	}
	message := make([]byte, length)

	_, err = io.ReadFull(r, message)
	if err != nil {
		return nil, err
	}
	m := &Message{
		messageID(message[0]),
		message[1:],
	}
	return m, nil
}

func (m *Message) name() string {
	if m == nil {
		return "KeepAlive"
	}
	switch m.ID {
	case MsgChoke:
		return "Choke"
	case MsgUnchoke:
		return "Unchoke"
	case MsgInterested:
		return "Interested"
	case MsgNotInterested:
		return "NotInterested"
	case MsgHave:
		return "Have"
	case MsgBitfield:
		return "Bitfield"
	case MsgRequest:
		return "Request"
	case MsgPiece:
		return "Piece"
	case MsgCancel:
		return "Cancel"
	default:
		return fmt.Sprintf("Unknown#%d", m.ID)
	}
}

func (m *Message) String() string {
	if m == nil {
		return m.name()
	}
	return fmt.Sprintf("%s [%d]", m.name(), len(m.Payload))
}

func FormatHave(index int) *Message {
	msg := &Message{}
	msg.ID = MsgHave
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	msg.Payload = payload
	return msg
}

//发送请求，需要标明请求的第几个 piece  piece下开始地址  请求的块长度

func FormatRequest(index, begin, length int) *Message {
	msg := new(Message)
	msg.ID = MsgRequest
	msg.Payload = make([]byte, 12)
	binary.BigEndian.PutUint32(msg.Payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(msg.Payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(msg.Payload[8:12], uint32(length))
	return msg
}

// ParseHave parses a HAVE message
func ParseHave(msg *Message) (int, error) {
	//解析haveMsg ,获取peer拥有的index序号
	if msg.ID != MsgHave {
		return 0, fmt.Errorf("Expected HAVE (ID %d), got ID %d", MsgHave, msg.ID)
	}
	//检查payload
	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("Expected payload length 4, got length %d", len(msg.Payload))
	}
	idx := binary.BigEndian.Uint32(msg.Payload)
	return int(idx), nil
}

// ParsePiece 解析一个piece消息
func ParsePiece(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("Expected PIECE (ID %d), got ID %d", MsgPiece, msg.ID)
	}
	//接下来检查payload
	if len(msg.Payload) < 8 {
		//至少为8
		return 0, fmt.Errorf("Payload too short. %d < 8", len(msg.Payload))
	}
	idx := int(binary.BigEndian.Uint32(msg.Payload[0:4]))
	if idx != index {
		return 0, fmt.Errorf("Expected index %d, got %d", index, idx)
	}
	//接下来检查block
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("Begin offset too high. %d >= %d", begin, len(buf))
	}
	data := msg.Payload[8:]
	if begin+len(data) > len(buf) {
		return 0, fmt.Errorf("Data too long [%d] for offset %d with length %d", len(data), begin, len(buf))
	}
	copy(buf[begin:], data)
	return len(data), nil
}
