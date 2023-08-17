package nego

import (
	"context"
	"kstack"
	"kstack/internal"
	"kstack/negotiator/core"
	"kstack/negotiator/handshake"
	nc "kstack/negotiator/negotiated_conn"
)

type Config = core.Config
type HSOpt = core.HSOpt
type PassCode = core.PassCode
type IStore = core.IStore

type INegotiator interface {
	HandleInbound(ctx context.Context, conn kstack.IConn) (kstack.IConn, error)
	HandleOutbound(ctx context.Context, conn kstack.IConn, oopt HSOpt) (kstack.IConn, error)
}

type _Negotiator struct {
	cfg   Config
	store IStore
}

func New(cfg Config, store IStore) INegotiator {
	return &_Negotiator{
		cfg:   cfg,
		store: store,
	}
}

func (n *_Negotiator) HandleInbound(ctx context.Context, conn internal.IConn) (internal.IConn, error) {
	r, err := handshake.Respond(ctx, &n.cfg, n.store, conn)
	if err != nil {
		return nil, err
	}
	return nc.New(&r), nil
}

func (n *_Negotiator) HandleOutbound(ctx context.Context, conn internal.IConn, hsopt core.HSOpt) (internal.IConn, error) {
	r, err := handshake.Initiate(ctx, &n.cfg, hsopt, n.store, conn)
	if err != nil {
		return nil, err
	}
	return nc.New(&r), nil
}
