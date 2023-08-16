package handshake

import "slices"

//go:generate msgp -unexported -v -io=false

//msgp:tuple _Hello1_Payload
type _Hello1_Payload struct {
	Versions    [4]uint8
	ICanEncrypt bool
}

func (p *_Hello1_Payload) Pack(ICanEncrypt bool) {
	n := min(len(p.Versions), len(protocolVersions))
	copy(p.Versions[:n], protocolVersions[:n])
	p.ICanEncrypt = ICanEncrypt
}

func (p *_Hello1_Payload) ChooseVersion() byte {
	var selected byte = 0
	for _, v := range p.Versions {
		if v == 0 {
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
	ChosenVersion uint8
	UseEncryption bool
}

func (p *_Resp1_Payload) VerifyVersion(hello *_Hello1_Payload) bool {
	v := p.ChosenVersion
	return v != 0 && slices.Contains(hello.Versions[:], v)
}
