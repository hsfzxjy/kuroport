package handshake

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// MarshalMsg implements msgp.Marshaler
func (z *_Hello1_Payload) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendArrayHeader(o, uint32(4))
	for za0001 := range z.Versions {
		o = msgp.AppendUint8(o, z.Versions[za0001])
	}
	o = msgp.AppendBool(o, z.ICanEncrypt)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *_Hello1_Payload) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Versions")
		return
	}
	if zb0002 != uint32(4) {
		err = msgp.ArrayError{Wanted: uint32(4), Got: zb0002}
		return
	}
	for za0001 := range z.Versions {
		z.Versions[za0001], bts, err = msgp.ReadUint8Bytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "Versions", za0001)
			return
		}
	}
	z.ICanEncrypt, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ICanEncrypt")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *_Hello1_Payload) Msgsize() (s int) {
	s = 1 + msgp.ArrayHeaderSize + (4 * (msgp.Uint8Size)) + msgp.BoolSize
	return
}

// MarshalMsg implements msgp.Marshaler
func (z _Resp1_Payload) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 2
	o = append(o, 0x92)
	o = msgp.AppendUint8(o, z.ChosenVersion)
	o = msgp.AppendBool(o, z.UseEncryption)
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *_Resp1_Payload) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 2 {
		err = msgp.ArrayError{Wanted: 2, Got: zb0001}
		return
	}
	z.ChosenVersion, bts, err = msgp.ReadUint8Bytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "ChosenVersion")
		return
	}
	z.UseEncryption, bts, err = msgp.ReadBoolBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "UseEncryption")
		return
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z _Resp1_Payload) Msgsize() (s int) {
	s = 1 + msgp.Uint8Size + msgp.BoolSize
	return
}