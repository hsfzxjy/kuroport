package kc

import "errors"

//go:generate msgp

var ErrBadKeyType = errors.New("invalid or unsupported key type")

//msgp:tuple PubKeyMsgp
type PubKeyMsgp struct {
	Type Type
	Data []byte
}

// PubKeyUnmarshaller is a func that creates a PubKey from a given slice of bytes
type PubKeyUnmarshaller func(data []byte) (PubKey, error)

// PrivKeyUnmarshaller is a func that creates a PrivKey from a given slice of bytes
type PrivKeyUnmarshaller func(data []byte) (PrivKey, error)

// PubKeyUnmarshallers is a map of unmarshallers by key type
var PubKeyUnmarshallers = map[Type]PubKeyUnmarshaller{
	Type_Ed25519: UnmarshalEd25519PublicKey,
}

// PrivKeyUnmarshallers is a map of unmarshallers by key type
var PrivKeyUnmarshallers = map[Type]PrivKeyUnmarshaller{
	Type_Ed25519: UnmarshalEd25519PrivateKey,
}

// UnmarshalPublicKey converts a protobuf serialized public key into its
// representative object
func UnmarshalPublicKey(data []byte) (PubKey, error) {
	var msg PubKeyMsgp
	_, err := msg.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	return PublicKeyFromMsgp(msg)
}

// PublicKeyFromMsgp converts an unserialized protobuf PublicKey message
// into its representative object.
func PublicKeyFromMsgp(msg PubKeyMsgp) (PubKey, error) {
	um, ok := PubKeyUnmarshallers[msg.Type]
	if !ok {
		return nil, ErrBadKeyType
	}

	data := msg.Data

	pk, err := um(data)
	if err != nil {
		return nil, err
	}

	return pk, nil
}

// MarshalPublicKey converts a public key object into a protobuf serialized
// public key
func MarshalPublicKey(k PubKey) ([]byte, error) {
	msg, err := PublicKeyToMsgp(k)
	if err != nil {
		return nil, err
	}

	return msg.MarshalMsg(nil)
}

// PublicKeyToMsgp converts a public key object into an unserialized
// protobuf PublicKey message.
func PublicKeyToMsgp(k PubKey) (PubKeyMsgp, error) {
	data, err := k.Raw()
	if err != nil {
		return PubKeyMsgp{}, err
	}
	return PubKeyMsgp{
		Type: k.Type(),
		Data: data,
	}, nil
}
