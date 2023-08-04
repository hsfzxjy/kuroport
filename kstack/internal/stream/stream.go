package stream

import (
	"kstack/internal"
	ku "kutil"

	"github.com/hsfzxjy/smux"
)

type _MuxedStream struct {
	*smux.Stream
	id   internal.StreamID
	conn internal.IConn
}

func newMuxedStream(
	id internal.StreamID,
	ic internal.IConn,
	stream *smux.Stream,
	disposeSelf ku.F) *_MuxedStream {
	s := new(_MuxedStream)
	s.Stream = stream
	s.id = id
	s.conn = ic
	go s.runLoop(disposeSelf)
	return s
}

func (s *_MuxedStream) Conn() internal.IConn {
	return s.conn
}

func (s *_MuxedStream) ID() internal.StreamID {
	return s.id
}

func (s *_MuxedStream) runLoop(disposeSelf ku.F) {
	<-s.Stream.GetDieCh()
	disposeSelf.Do()
}

type _Stream struct {
	id internal.StreamID
	internal.IConn
}

func newStream(id internal.StreamID, ic internal.IConn, disposeSelf ku.F) *_Stream {
	s := &_Stream{id, ic}
	go s.runLoop(disposeSelf)
	return s
}

func (s *_Stream) runLoop(disposeSelf ku.F) {
	<-s.DiedCh()
	disposeSelf.Do()
}

func (s *_Stream) ID() internal.StreamID {
	return s.id
}

func (s *_Stream) Conn() internal.IConn {
	return s.IConn
}
