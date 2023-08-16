package handshake

import (
	"encoding/binary"
	"kstack/negotiator/rw"
	ku "kutil"

	"github.com/flynn/noise"
	pool "github.com/libp2p/go-buffer-pool"
	"github.com/tinylib/msgp/msgp"
)

const lengthPrefixLength = rw.LengthPrefixLength

type _RW struct {
	rw.RW
	wbuf []byte

	setCipherStates func(cs1, cs2 *noise.CipherState)
}

func (rw *_RW) Init(s *_Session, bufSize int) (dispose ku.F) {
	rw.RW.Init(s.conn)
	rw.setCipherStates = s.setCipherStates
	buf := pool.Get(bufSize)
	rw.wbuf = buf
	return func() {
		pool.Put(buf)
	}
}

func (rw *_RW) ReadMessage(um msgp.Unmarshaler) error {
	n, err := rw.ReadNextInsecureMsgLen()
	if err != nil {
		return err
	}
	if um == nil {
		if n != 0 {
			panic("")
		}
		return nil
	}
	buf := pool.Get(n)
	defer pool.Put(buf)

	if err := rw.ReadNextMsgInsecure(buf[:n]); err != nil {
		return err
	}

	_, err = um.UnmarshalMsg(buf[:n])
	return err
}

func (rw *_RW) ReadAndDecryptMessage(um msgp.Unmarshaler, hs *noise.HandshakeState) error {
	n, err := rw.ReadNextInsecureMsgLen()
	if err != nil {
		return err
	}
	buf := pool.Get(n)
	defer pool.Put(buf)

	if err := rw.ReadNextMsgInsecure(buf[:n]); err != nil {
		return err
	}

	msg, cs1, cs2, err := hs.ReadMessage(nil, buf[:n])
	if err != nil {
		return err
	}
	if um == nil {
		return nil
	}

	_, err = um.UnmarshalMsg(msg)
	if err != nil {
		return err
	}

	if cs1 != nil && cs2 != nil {
		rw.setCipherStates(cs1, cs2)
	}

	return nil
}

// msg must be buf[2:x]
func (rw *_RW) WriteBytes(msg []byte) error {
	n := len(msg)
	binary.BigEndian.PutUint16(rw.wbuf[:], uint16(n))
	return rw.WriteMsgInsecure(rw.wbuf[:n+lengthPrefixLength])
}

func (rw *_RW) WriteMessage(m msgp.Marshaler) error {
	var encoded []byte
	if m != nil {
		var err error
		encoded, err = m.MarshalMsg(rw.wbuf[lengthPrefixLength:lengthPrefixLength])
		if err != nil {
			return err
		}
	}
	return rw.WriteBytes(encoded)
}

func (rw *_RW) EncryptAndWriteMessage(m msgp.MarshalSizer, hs *noise.HandshakeState) error {
	var payload []byte
	if m != nil {
		var err error
		msgbuf := pool.Get(m.Msgsize())
		defer pool.Put(msgbuf)
		payload, err = m.MarshalMsg(msgbuf[:0])
		if err != nil {
			return err
		}
	}
	encoded, cs1, cs2, err := hs.WriteMessage(rw.wbuf[lengthPrefixLength:lengthPrefixLength], payload)
	if err != nil {
		return err
	}
	if err := rw.WriteBytes(encoded); err != nil {
		return err
	}
	if cs1 != nil && cs2 != nil {
		rw.setCipherStates(cs1, cs2)
	}
	return nil
}
