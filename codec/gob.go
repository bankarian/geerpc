package codec

import (
	"encoding/gob"
	"io"
	"log"
)

type GobCodec struct {
	enc  *gob.Encoder
	dec  *gob.Decoder
	conn io.ReadWriteCloser
}

func NewGobCodec(conn io.ReadWriteCloser) ICodec {
	return &GobCodec{
		conn: conn,
		enc:  gob.NewEncoder(conn),
		dec:  gob.NewDecoder(conn),
	}
}

var _ ICodec = (*GobCodec)(nil)

func (g *GobCodec) Close() error {
	return g.conn.Close()
}

func (g *GobCodec) ReadHeader(h *Header) error {
	return g.dec.Decode(h)
}

func (g *GobCodec) ReadBody(b interface{}) error {
	return g.dec.Decode(b)
}

func (g *GobCodec) Write(h *Header, b interface{}) error {
	// encode h to conn, encode b to conn
	if err := g.enc.Encode(h); err != nil {
		log.Println("codec: encode header error", err)
		return err
	}
	if err := g.enc.Encode(b); err != nil {
		log.Println("codec: encode body error", err)
		return err
	}
	return nil
}
