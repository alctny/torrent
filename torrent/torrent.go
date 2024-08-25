package torrent

import (
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/alctny/torrent/bencode"
	"github.com/go-resty/resty/v2"
)

const SHALEN = 20

var (
	ErrPiecesLength    = errors.New("pieces length is not a multiple of 20")
	ErrNodersLength    = errors.New("nodes length is not a multiple of 2")
	ErrNodeFormat      = errors.New("node format error")
	ErrParserDHP       = errors.New("parser dhp error")
	ErrNoPeers         = errors.New("no peers ant trackers")
	ErrNetwork         = errors.New("network error")
	ErrTrackerInvalide = errors.New("tracker invalide")
)

var (
	PeerID [SHALEN]byte
)

func init() {
	_, err := rand.Read(PeerID[:])
	if err != nil {
		panic(err)
	}
}

type Torrent struct {
	// 原始数据
	file string `bencode:"-"`
	data []byte `bencode:"-"`
	// 资源信息
	Raw     *RawTorrent  `bencode:"-"`
	Base    *FileInfo    `bencode:"-"`
	Tracker *TrackerInfo `bencode:"-"`
	Peer    *PeerInfo    `bencode:"-"`
	// 当前正在使用的 tracker
	trackerIndex int `bencode:"-"`
}

type PeerInfo struct {
	Peers []Node `bencode:"-"`
}

type TrackerInfo struct {
	Trackers    []string `bencode:"-"`
	HttpSeed    []string `bencode:"-"`
	Interval    int64    `bencode:"-"`
	MinInterval int64    `bencode:"-"`
}

type FileInfo struct {
	Sha1    [SHALEN]byte   `bencode:"-"`
	Name    string         `bencode:"-"`
	Size    int64          `bencode:"-"`
	Ed2k    string         `bencode:"-"`
	Comment string         `bencode:"-"`
	Pieces  [][SHALEN]byte `bencode:"-"`
	FileSha [SHALEN]byte   `bencode:"-"`
}

type RawTorrent struct {
	Anonunce     string     `bencode:"announce"`
	AnnounceList [][]string `bencode:"announce-list"`
	UrlList      []string   `bencode:"url-list"`
	Node         [][2]any   `bencode:"nodes"`
	Comment      string     `bencode:"comment"`
	CreateAt     int64      `bencode:"creation date"`
	CreateBy     string     `bencode:"created by"`
	HttpSeed     []string   `bencode:"httpseeds"`
	Encoding     string     `bencode:"encoding"`
	Info         RawInfo    `bencode:"info"`
}

type RawInfo struct {
	Files       []RawFile `bencode:"files"`
	Lnegth      int64     `bencode:"length"`
	Name        string    `bencode:"name"`
	PieceLength int64     `bencode:"piece length"`
	Pieces      string    `bencode:"pieces"`
	Pieces6     string    `bencode:"pieces6"`
	NameUTF8    string    `bencode:"name.utf-8"`
	Ed2K        string    `bencode:"ed2k"`
	FileHash    []byte    `bencode:"filehash"`
}

type RawFile struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
}

type Node struct {
	IP   string `bencode:"-"`
	Port int64  `bencode:"-"`
}

// NewTorrent 从 .torrent 文件创建 Torrent 结构
func NewTorrent(file string) (*Torrent, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var raw RawTorrent
	err = bencode.Unmarshal(data, &raw)
	if err != nil {
		return nil, err
	}

	// 资源唯一标识
	infoRaw, err := bencode.GetRaw(data, "info")
	if err != nil {
		return nil, err
	}

	// pices
	pieces, err := PiecesSplit(raw.Info.Pieces)
	if err != nil {
		return nil, err
	}

	// DHP 信息
	node, err := ParserTupeNodes(raw.Node)
	if err != nil {
		return nil, err
	}

	// lengthSum
	lengthSum := raw.Info.Lnegth
	if lengthSum != 0 {
		for _, rif := range raw.Info.Files {
			lengthSum += rif.Length
		}
	}

	// tracker list
	tracker := []string{}
	if raw.Anonunce != "" {
		tracker = append(tracker, raw.Anonunce)
	}
	if raw.AnnounceList != nil {
		for _, al := range raw.AnnounceList {
			tracker = append(tracker, al...)
		}
	}
	if raw.UrlList != nil {
		tracker = append(tracker, raw.UrlList...)
	}

	// file sha1
	var fileSha [SHALEN]byte
	if len(raw.Info.FileHash) > 0 {
		fileSha = [SHALEN]byte(raw.Info.FileHash)
	}

	tor := &Torrent{
		file: file,
		data: data,
		Raw:  &raw,
		Base: &FileInfo{
			Sha1:    sha1.Sum(infoRaw),
			Name:    raw.Info.Name,
			Size:    lengthSum,
			Ed2k:    raw.Info.Ed2K,
			Comment: raw.Comment,
			Pieces:  pieces,
			FileSha: fileSha,
		},
		Tracker: &TrackerInfo{
			Trackers: tracker,
			HttpSeed: raw.HttpSeed,
		},
		Peer: &PeerInfo{
			Peers: node,
		},

		trackerIndex: -1,
	}

	if tor.Tracker.Trackers == nil && tor.Peer.Peers == nil {
		return nil, ErrNoPeers
	}

	return tor, nil
}

// TODO: 重试 并发 超时
// TryGetPeer 从 tracker 获取 peers
func (tor *Torrent) TryGetPeer() error {
	trackerUrl := tor.TryTracker()
	if trackerUrl == "" {
		return ErrNoPeers
	}

	params := url.Values{
		"info_hash":  []string{string(tor.Base.Sha1[:])},
		"peer_id":    []string{string(PeerID[:])},
		"port":       []string{strconv.Itoa(5000)},
		"uploaded":   []string{"0"},
		"downloaded": []string{"0"},
		"compact":    []string{"1"},
		"left":       []string{strconv.Itoa(int(tor.Base.Size))},
	}

	base, _ := url.Parse(trackerUrl)
	base.RawQuery = params.Encode()

	var body []byte
	client := resty.New()
	resp, err := client.
		SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36").
		R().
		SetBody(body).
		Get(base.String())
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return errors.Join(ErrNetwork, fmt.Errorf("status code: %d", resp.StatusCode()))
	}

	var res TrackerResp
	var buf []byte = resp.Body()
	err = bencode.Unmarshal(buf, &res)
	if err != nil {
		return err
	}
	nodes, err := res.ParserPeers()
	if err != nil {
		return err
	}

	// TODO: 去重
	tor.Peer.Peers = append(tor.Peer.Peers, nodes...)

	return nil
}

func (tor *Torrent) TryTracker() string {
	tor.trackerIndex++
	if tor.trackerIndex >= len(tor.Tracker.Trackers) {
		return ""
	}
	if tor.trackerIndex > 9 {
		return ""
	}
	return tor.Tracker.Trackers[tor.trackerIndex]
}

// ParserTupeNodes 解析 ("ip", port) 元组到 Node 结构
func ParserTupeNodes(raw [][2]any) ([]Node, error) {
	count := len(raw)

	t := make([]Node, count)
	for in := 0; in < count; in++ {
		ip, _ := raw[in][0].(string)
		port, _ := raw[in][1].(int64)

		t[in] = Node{IP: ip, Port: port}
	}

	return t, nil
}

// PiecesSplit 将 pieces 字符串切割成 [20]byte 切片
func PiecesSplit(pieces string) ([][SHALEN]byte, error) {
	if len(pieces)%SHALEN != 0 {
		return nil, ErrPiecesLength
	}
	count := len(pieces) / SHALEN
	sha1s := make([][SHALEN]byte, count)
	for in := 0; in < count; in++ {
		sha1s[in] = [SHALEN]byte([]byte(pieces[in*SHALEN : in*SHALEN+SHALEN]))
	}
	return sha1s, nil
}
