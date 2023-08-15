package handshake

//go:generate msgp -unexported -v -io=false

//msgp:tuple _Hello1_Payload
type _Hello1_Payload struct {
	Versions    [4]uint8
	ICanEncrypt bool
}

//msgp:tuple _Resp1_Payload
type _Resp1_Payload struct {
	ChosenVersion uint8
	UseEncryption bool
}
