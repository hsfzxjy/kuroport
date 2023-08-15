package peer_test

import (
	"crypto/rand"
	kc "kstack/crypto"
	"kstack/peer"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func Test_IdFromEd25519(t *testing.T) {
	for i := 0; i <= 10; i++ {
		priv, _, err := kc.GenerateEd25519Key(rand.Reader)
		require.ErrorIs(t, err, nil)
		id, err := peer.IDFromPrivateKey(priv)
		require.ErrorIs(t, err, nil)
		require.Len(t, id, 38)
		pub, err := id.ExtractPublicKey()
		require.ErrorIs(t, err, nil)
		require.True(t, priv.GetPublic().Equals(pub))
		require.False(t, id.IsAnonymous())
	}
}

func Test_IdFromUUID(t *testing.T) {
	for i := 0; i <= 10; i++ {
		uid := uuid.New()
		id := peer.IDFromUUID(uid)
		_, err := id.ExtractPublicKey()
		require.ErrorIs(t, err, peer.ErrNoPublicKey)
		require.True(t, id.IsAnonymous())
	}
}
