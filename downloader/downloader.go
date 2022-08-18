package downloader

import (
	"bitDownloader/peer"
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"runtime"
	"time"
)

//提供最终的下载方法
const MaxBlockSize = 16384 //16k

// MaxBacklog 一个client所能持有的最大未完成请求数量
// MaxBacklog is the number of unfulfilled requests a client can have in its pipeline
const MaxBacklog = 5

// Torrent holds data required to download a torrent from a list of peers
type Torrent struct {
	Peers       []peer.Peer
	PeerID      [20]byte
	InfoHash    [20]byte
	PieceHashes [][20]byte
	PieceLength int
	Length      int
	Name        string
}

//每一个piece请求
type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

//每一个piece请求的结果
type pieceResult struct {
	index int
	buf   []byte
}

//管理每一个client
type pieceProgress struct {
	index      int
	client     *Client
	buf        []byte
	downloaded int
	requested  int
	backlog    int
}

// Download 下载文件并将所有数据保存在内存中 ，返回的[]byte 切片为文件数据
func (t *Torrent) Download() ([]byte, error) {
	log.Println("Starting download for", t.Name)
	//创建请求队列
	workChan := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)
	//接下来创建每一个请求填充入请求队列
	for index, hash := range t.PieceHashes {
		p := &pieceWork{
			hash:   hash,
			length: t.calculatePieceSize(index), //需要计算开始结束边界
		}
		workChan <- p //完成请求入队，此时需要开启协程同步处理请求
	}
	for _, peer := range t.Peers {
		go t.startDownloadWorker(peer, workChan, results)
	}
	//此时正在进行下载

	//创建缓冲数组，将接收下载到的数据
	buf := make([]byte, t.Length)

	//记录已经完成的次数
	donePieces := 0

	for donePieces < len(t.PieceHashes) {
		//获取res
		res := <-results
		//计算开始于结束下标
		start, end := t.calculateBoundsForPiece(res.index)
		copy(buf[start:end], res.buf)
		donePieces++
		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		//获取当前允许的goroutine
		numWorkers := runtime.NumGoroutine() - 1 // subtract 1 for main thread
		log.Printf("(%0.2f%%) Downloaded piece #%d from %d peers\n", percent, res.index, numWorkers)
	}
	close(workChan)
	return buf, nil
}

func (t *Torrent) calculateBoundsForPiece(index int) (begin int, end int) {
	begin = index * t.PieceLength
	end = (index + 1) * t.PieceLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}
func (t *Torrent) calculatePieceSize(index int) int {
	//需要注意边界问题
	begin, end := t.calculateBoundsForPiece(index)
	return end - begin

}

//开始下载，向各个peer发起请求，对应几个peer就对应几个工作线程
func (t *Torrent) startDownloadWorker(peer peer.Peer, workChan chan *pieceWork, results chan *pieceResult) {
	//首先需要创建客户端
	c, err := New(peer, t.PeerID, t.InfoHash)
	if err != nil {
		log.Printf("Could not handshake with %s. Disconnecting\n", peer.Ip)
		return
	}
	defer c.Conn.Close()

	log.Printf("Completed handshake with %s\n", peer.Ip)
	//此时以及完成了握手以及获取了peer存有的piece
	//发送unbolck ，interested消息

	c.SendUnchoke()
	c.SendInterested()

	//接下来要由工作队列中不断取出请求并进行下载
	for work := range workChan {
		if !c.Bitfield.HasPiece(work.index) {
			//如果没有想要的piece就返回
			workChan <- work
			continue
		}
		//尝试开始下载
		buf, err := attemptDownloadPiece(c, work)

		//下载失败，此时不应该再向该peer请求
		if err != nil {
			workChan <- work
			return
		}

		//检查下载数据的完整性
		err = checkIntegrity(work, buf)

		//校验失败
		if err != nil {
			log.Printf("Piece #%d failed integrity check\n", work.index)
			workChan <- work
			continue
		}

		//此时数据成功下载,发送message说明数据已经完成下载
		c.SendHave(work.index)

		//将结果添加至结果队列
		result := &pieceResult{
			index: work.index,
			buf:   buf,
		}
		results <- result
	}

}

//检查完整性
func checkIntegrity(work *pieceWork, buf []byte) error {
	hash := sha1.Sum(buf)
	if !bytes.Equal(work.hash[:], hash[:]) {
		return fmt.Errorf("Index %d failed integrity check", work.index)
	}
	return nil
}

//尝试下载piece，需要多块下载
func attemptDownloadPiece(c *Client, work *pieceWork) ([]byte, error) {
	state := pieceProgress{
		index:  work.index,
		client: c,
		buf:    make([]byte, work.length),
	}
	c.Conn.SetDeadline(time.Now().Add(time.Second * 30)) //30秒下载piece
	defer c.Conn.SetDeadline(time.Time{})
	for state.downloaded < work.length {
		//没有下载好并且没有阻塞
		if !state.client.Choked {
			//积压的工作小于最大积压数，请求的数据小于最终数据
			for state.backlog < MaxBacklog && state.requested < work.length {
				blockSize := MaxBlockSize
				if work.length-state.requested < blockSize {
					blockSize = work.length - state.requested
				}
				err := c.SendRequest(work.index, state.requested, blockSize)
				if err != nil {
					return nil, err
				}
				state.backlog++
				state.requested += blockSize
			}
		}
		err := state.readMessage()
		if err != nil {
			return nil, err
		}
	}
	return state.buf, nil
}

//接收msg
func (state *pieceProgress) readMessage() error {
	msg, err := state.client.Read() // this call blocks
	if err != nil {
		return err
	}

	if msg == nil { // keep-alive
		return nil
	}
	switch msg.ID {
	case MsgUnchoke:
		//标识没有阻塞
		state.client.Choked = false
	case MsgChoke:
		//标识阻塞
		state.client.Choked = true
	case MsgHave:
		//标识存在该piece
		index, err := ParseHave(msg)
		if err != nil {
			return err
		}
		state.client.Bitfield.SetPiece(index)
	case MsgPiece:
		//接收piece
		n, err := ParsePiece(state.index, state.buf, msg)
		if err != nil {
			return err
		}
		state.downloaded += n
		state.backlog--
	}
	return nil
}
