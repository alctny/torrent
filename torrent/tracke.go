package torrent

import (
	"fmt"
)

// TrackerResp  与 tracker 通信的响应，包含 Perrs 的信息
type TrackerResp struct {
	Interval    int64  `bencode:"interval"`
	MinInterval int64  `bencode:"min interval"`
	Peers       []byte `bencode:"peers"`
}

func (t *TrackerResp) ParserPeers() ([]Node, error) {
	if len(t.Peers)%6 != 0 {
		return nil, fmt.Errorf("invalid peer length, should be multiple of 6, but got %d", len(t.Peers))
	}
	len := len(t.Peers) / 6
	nodes := make([]Node, len)
	for i := 0; i < len; i++ {
		peer := t.Peers[i*6 : (i+1)*6]
		ip := fmt.Sprintf("%d.%d.%d.%d", peer[0], peer[1], peer[2], peer[3])
		port := int64(peer[4])<<8 + int64(peer[5])
		nodes[i] = Node{IP: ip, Port: port}
	}
	return nodes, nil
}
