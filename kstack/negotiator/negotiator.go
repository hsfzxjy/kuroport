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
type OutboundOption = core.OutboundOption
type PassCode = core.PassCode
type IStore = core.IStore

type INegotiator interface {
	HandleInbound(ctx context.Context, conn kstack.IConn) (kstack.IConn, error)
	HandleOutbound(ctx context.Context, conn kstack.IConn, oopt OutboundOption) (kstack.IConn, error)
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

func (n *_Negotiator) HandleOutbound(ctx context.Context, conn internal.IConn, oopt core.OutboundOption) (internal.IConn, error) {
	r, err := handshake.Initiate(ctx, &n.cfg, oopt, n.store, conn)
	if err != nil {
		return nil, err
	}
	return nc.New(&r), nil
}
