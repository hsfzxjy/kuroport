package handshake

import "slices"

//go:generate msgp -unexported -v -io=false

//msgp:tuple _Version
type _Version byte

func (v _Version) IsValid() bool { return v != 0 }

//msgp:tuple _Hello1_Payload
type _Hello1_Payload struct {
	Versions    [4]byte
	ICanEncrypt bool
}

func (p *_Hello1_Payload) Pack(ICanEncrypt bool) {
	n := min(len(p.Versions), len(protocolVersions))
	copy(p.Versions[:n], []byte(protocolVersions[:n]))
	p.ICanEncrypt = ICanEncrypt
}

func (p *_Hello1_Payload) ChooseVersion() _Version {
	var selected _Version
	for _, v := range p.Versions {
		v := _Version(v)
		if !v.IsValid() {
			continue
		}
		if _, ok := protocols[v]; ok && selected < v {
			selected = v
		}
	}
	return selected
}

//msgp:tuple _Resp1_Payload
type _Resp1_Payload struct {
	ChosenVersion _Version
	UseEncryption bool
}

func (p *_Resp1_Payload) VerifyVersion(hello *_Hello1_Payload) bool {
	v := p.ChosenVersion
	return v.IsValid() && slices.Contains(hello.Versions[:], byte(v))
}
