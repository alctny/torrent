package torrent

import (
	"errors"
	"os"

	"github.com/alctny/torrent/bencode"
)

var (
	ErrPiecesLength = errors.New("pieces length is not a multiple of 20")
)

type File struct {
	Length int64    `bencode:"length"`
	Path   []string `bencode:"path"`
}

type Info struct {
	Pieces      string   `bencode:"pieces"`
	Hash        [][]byte `bencode:"-"`
	Name        string   `bencode:"name"`
	PieceLength *int64   `bencode:"piece length"`
	Files       []File   `bencode:"files"`
}

type Torrent struct {
	UrlList []string `bencode:"url-list"`
	Info    Info     `bencode:"info"`
}

func (in *Info) picess2Hash() error {
	if len(in.Pieces)%20 != 0 {
		return ErrPiecesLength
	}

	count := len(in.Pieces) / 20
	in.Hash = make([][]byte, count)
	for i := 0; i < count; i++ {
		in.Hash[i] = []byte(in.Pieces[i*20 : i*20+20])
	}

	return nil
}

// Parser parse torrent file and return Torrent struct
func Parser(f string) (*Torrent, error) {
	data, err := os.ReadFile(f)
	if err != nil {
		return nil, err
	}

	var t Torrent
	err = bencode.Unmarshal(data, &t)
	if err != nil {
		return nil, err
	}
	err = t.Info.picess2Hash()
	return &t, err
}
