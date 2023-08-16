package nc

import (
	"encoding/binary"
	"kstack/negotiator/handshake"
	"kstack/negotiator/rw"
	"kstack/peer"
	"sync"

	"github.com/flynn/noise"
	pool "github.com/libp2p/go-buffer-pool"
	"golang.org/x/crypto/chacha20poly1305"
)

type _SecureConn struct {
	rw.RW
	remoteID peer.ID

	rLock, wLock sync.Mutex

	qseek int    // queued bytes seek value.
	qbuf  []byte // queued bytes buffer.

	enc *noise.CipherState
	dec *noise.CipherState
}

func _NewSecureConn(r *handshake.Result) *_SecureConn {
	c := new(_SecureConn)
	c.RW = r.RW
	c.enc = r.Enc
	c.dec = r.Dec
	c.remoteID = r.RemoteID
	return c
}

func (c *_SecureConn) Read(buf []byte) (int, error) {
	c.rLock.Lock()
	defer c.rLock.Unlock()

	if c.qbuf != nil {
		// we have queued bytes; copy as much as we can.
		copied := copy(buf, c.qbuf[c.qseek:])
		c.qseek += copied
		if c.qseek == len(c.qbuf) {
			// queued buffer is now empty, reset and release.
			pool.Put(c.qbuf)
			c.qseek, c.qbuf = 0, nil
		}
		return copied, nil
	}

	// length of the next encrypted message.
	nextMsgLen, err := c.ReadNextInsecureMsgLen()
	if err != nil {
		return 0, err
	}

	// If the buffer is atleast as big as the encrypted message size,
	// we can read AND decrypt in place.
	if len(buf) >= nextMsgLen {
		if err := c.ReadNextMsgInsecure(buf[:nextMsgLen]); err != nil {
			return 0, err
		}

		dbuf, err := c.decrypt(buf[:0], buf[:nextMsgLen])
		if err != nil {
			return 0, err
		}

		return len(dbuf), nil
	}

	// otherwise, we get a buffer from the pool so we can read the message into it
	// and then decrypt in place, since we're retaining the buffer (or a view thereof).
	cbuf := pool.Get(nextMsgLen)
	if err := c.ReadNextMsgInsecure(cbuf); err != nil {
		return 0, err
	}

	if c.qbuf, err = c.decrypt(cbuf[:0], cbuf); err != nil {
		return 0, err
	}

	// copy as many bytes as we can; update seek pointer.
	c.qseek = copy(buf, c.qbuf)

	return c.qseek, nil
}

func (s *_SecureConn) Write(data []byte) (int, error) {
	s.wLock.Lock()
	defer s.wLock.Unlock()

	var (
		written int
		cbuf    []byte
		total   = len(data)
	)

	if total < rw.MaxPlaintextLength {
		cbuf = pool.Get(total + chacha20poly1305.Overhead + rw.LengthPrefixLength)
	} else {
		cbuf = pool.Get(rw.MaxTransportMsgLength + rw.LengthPrefixLength)
	}

	defer pool.Put(cbuf)

	for written < total {
		end := written + rw.MaxPlaintextLength
		if end > total {
			end = total
		}

		b, err := s.encrypt(cbuf[:rw.LengthPrefixLength], data[written:end])
		if err != nil {
			return 0, err
		}

		binary.BigEndian.PutUint16(b, uint16(len(b)-rw.LengthPrefixLength))

		err = s.WriteMsgInsecure(b)
		if err != nil {
			return written, err
		}
		written = end
	}
	return written, nil
}

func (c *_SecureConn) decrypt(out, ciphertext []byte) ([]byte, error) {
	return c.dec.Decrypt(out, nil, ciphertext)
}

func (c *_SecureConn) encrypt(out, plaintext []byte) ([]byte, error) {
	return c.enc.Encrypt(out, nil, plaintext)
}
