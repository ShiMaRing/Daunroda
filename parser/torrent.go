package parser

import (
	"bitDownloader/downloader"
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/jackpal/bencode-go"
	"io"
	"math/rand"
	"net/url"
	"os"
	"strconv"
)

//提供种子文件的解析工作

// BencodeInfo 解析结构体
type BencodeInfo struct {
	//各个种子信息字段
	Pieces      string `bencode:"pieces"`      //binary blob of the hashes of each piece
	PieceLength int    `bencode:"pieceLength"` //分片长度
	Length      int    `bencode:"length"`      //总长度
	Name        string `bencode:"mame"`
}

//使用sha1 获取编码
func (i *BencodeInfo) hash() ([20]byte, error) {
	var buf bytes.Buffer
	err := bencode.Marshal(&buf, *i)
	if err != nil {
		//返回空
		return [20]byte{}, err
	}
	return sha1.Sum(buf.Bytes()), nil
}

func (i *BencodeInfo) splitPieceHashes() ([][20]byte, error) {
	//进行分割操作
	hashLength := 20
	//先进行转化
	buf := []byte(i.Pieces)

	if len(buf)%hashLength != 0 {
		return nil, fmt.Errorf("Received malformed pieces of length %d", len(buf))
	}
	numHashs := len(buf) / hashLength

	tmp := make([][20]byte, numHashs)

	for i := 0; i < numHashs; i++ {
		copy(tmp[i][:], buf[i*hashLength:(i+1)*hashLength])
	}
	return tmp, nil
}

// BencodeTorrent 解析种子
type BencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     BencodeInfo `bencode:"info"`
}

// TorrentFile 标识结构体
type TorrentFile struct {
	Announce    string     //发布者
	InfoHash    [20]byte   //当前info的哈希
	PieceHashes [][20]byte //全部的哈希
	PieceLength int        //某块长度
	Length      int        //完整长度
	Name        string     //资源名称
}

// Open 由输入流中读取输入
func Open(r io.Reader) (*BencodeTorrent, error) {
	b := &BencodeTorrent{}
	err := bencode.Unmarshal(r, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

//转化方法
func (bto BencodeTorrent) ToTorrentFile() (TorrentFile, error) {

	hash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	//接下来进行分割
	pieceHashes, err := bto.Info.splitPieceHashes()

	if err != nil {
		return TorrentFile{}, err
	}

	t := TorrentFile{
		Announce:    bto.Announce,
		InfoHash:    hash,
		PieceHashes: pieceHashes,
		PieceLength: bto.Info.PieceLength,
		Length:      bto.Info.Length,
		Name:        bto.Info.Name,
	}
	return t, nil
}

// DownloadToFile downloads a torrent and writes it to a file
func (t *TorrentFile) DownloadToFile(path string) error {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return err
	}

	peers, err := t.requestPeers(peerID, 6881)
	if err != nil {
		return err
	}

	torrent := downloader.Torrent{
		Peers:       peers,
		PeerID:      peerID,
		InfoHash:    t.InfoHash,
		PieceHashes: t.PieceHashes,
		PieceLength: t.PieceLength,
		Length:      t.Length,
		Name:        t.Name,
	}
	buf, err := torrent.Download()
	if err != nil {
		return err
	}

	outFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer outFile.Close()
	_, err = outFile.Write(buf)
	if err != nil {
		return err
	}
	return nil
}

//接下来需要向服务器声明作为一个种子接收者，并且需要发送get请求，携带相关参数
//peerID 随机生成的20字节 port端口，填充参数
func (t *TorrentFile) buildTrackerURL(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", err
	}
	//type Values map[string][]string
	paras := url.Values{
		//携带部分参数
		"info_hash":  []string{string(t.InfoHash[:])},
		"peer_id":    []string{string(peerID[:])},
		"port":       []string{strconv.Itoa(int(port))},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(t.Length)},
	}
	base.RawQuery = paras.Encode()

	return base.String(), nil
}
