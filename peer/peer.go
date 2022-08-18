package peer

import (
	"net"
	"strconv"
)

// Peer 提供tracker服务器的响应解析
//服务器将会返回相关列表，其中包括了ip以及端口
//一个长连接，每六个字节对应一个peer
type Peer struct {
	Ip   net.IP
	Port uint16
}

func (p Peer) String() string {
	return net.JoinHostPort(p.Ip.String(), strconv.Itoa(int(p.Port)))
}
