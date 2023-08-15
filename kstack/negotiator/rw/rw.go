package rw

import (
	"bufio"
	"encoding/binary"
	"io"
	"kstack"
)

const LEN_PREFIX_SIZE = 2

type conn = kstack.IConn

type RW struct {
	conn
	insecureReader *bufio.Reader

	buflen [LEN_PREFIX_SIZE]byte
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
	for {
		n, err := rw.conn.Write(data)
		if err != nil {
			return err
		}
		data = data[n:]
		if len(data) == 0 {
			return nil
		}
	}
}
