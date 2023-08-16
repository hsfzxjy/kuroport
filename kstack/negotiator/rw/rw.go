package rw

import (
	"bufio"
	"encoding/binary"
	"io"
	"kstack"

	"golang.org/x/crypto/chacha20poly1305"
)

const MaxTransportMsgLength = 0xffff
const MaxPlaintextLength = MaxTransportMsgLength - chacha20poly1305.Overhead
const LengthPrefixLength = 2

type conn = kstack.IConn

type RW struct {
	conn
	insecureReader *bufio.Reader

	buflen [LengthPrefixLength]byte
}

func (rw *RW) Conn() kstack.IConn {
	return rw.conn
}

func (rw *RW) Init(conn kstack.IConn) {
	rw.conn = conn
	rw.insecureReader = bufio.NewReader(conn)
}

func (rw *RW) ReadNextInsecureMsgLen() (int, error) {
	buflen := rw.buflen[:]
	_, err := io.ReadFull(rw.insecureReader, buflen)
	if err != nil {
		return 0, err
	}

	return int(binary.BigEndian.Uint16(buflen)), err
}

func (rw *RW) ReadNextMsgInsecure(buf []byte) error {
	_, err := io.ReadFull(rw.insecureReader, buf)
	return err
}

func (rw *RW) WriteMsgInsecure(data []byte) error {
	_, err := rw.conn.Write(data)
	return err
}
