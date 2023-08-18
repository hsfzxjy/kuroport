package handshake

import (
	"kstack/negotiator/core"
	sec "kstack/security"
	"slices"
)

//go:generate msgp -unexported -v -io=false

//msgp:tuple _Version
type _Version byte

func (v _Version) IsValid() bool { return v != 0 }

//msgp:tuple _Hello1_Payload
type _Hello1_Payload struct {
	Versions [4]byte
	sec.SecLevel
	sec.AuthLevel
	FirstTime bool
}

func (p *_Hello1_Payload) PackVersions() {
	n := min(len(p.Versions), len(protocolVersions))
	copy(p.Versions[:n], []byte(protocolVersions[:n]))
}

func (p *_Hello1_Payload) VerifyResp(resp *_Resp1_Payload) error {
	v := resp.ChosenVersion
	if !v.IsValid() || !slices.Contains(p.Versions[:], byte(v)) {
		return ErrUnsupportedVersion
	}
	if !p.SecLevel.Accept(resp.UseEncryption) {
		return ErrBadOption
	}
	if !p.AuthLevel.Accept(resp.UseAuth) {
		return ErrBadOption
	}
	return nil
}

//msgp:tuple _Resp1_Payload
type _Resp1_Payload struct {
	ChosenVersion _Version
	UseEncryption bool
	UseAuth       bool
}

func (resp *_Resp1_Payload) Handle(hello *_Hello1_Payload, cfg *core.Config) error {
	var selected _Version
	for _, v := range hello.Versions {
		v := _Version(v)
		if !v.IsValid() {
			continue
		}
		if _, ok := protocols[v]; ok && selected < v {
			selected = v
		}
	}
	if !selected.IsValid() {
		return ErrUnsupportedVersion
	}
	resp.ChosenVersion = selected
	if useE, ok := cfg.SecCap.Decide(hello.SecLevel); !ok {
		return ErrBadOption
	} else {
		resp.UseEncryption = useE
	}
	if useA, ok := cfg.AuthCap.Decide(hello.AuthLevel); !ok {
		return ErrBadOption
	} else {
		resp.UseAuth = useA
	}
	if resp.UseEncryption && !resp.UseAuth {
		return ErrBadOption
	}
	return nil
}
