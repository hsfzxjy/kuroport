package handshake_test

import (
	"io"
	mocktcp "kstack/internal/mock/tcp"
	"kstack/negotiator/handshake"
	"ktest"
	"testing"

	"github.com/stretchr/testify/require"
)

func _Test_MI_Stage00_BadHello1(
	t *testing.T,
	f func(*handshake.Hello1_Payload),
	respErr error,
) {
	c := mocktcp.Pair()
	defer c.Close()

	scope := ktest.Scope()

	scope.Go(func() {
		_, err := handshake.M(c.A,
			func(s *handshake.MSession) error {
				var hello1 handshake.Hello1_Payload
				f(&hello1)
				s.Rw.WriteMessage(&hello1)

				return s.Rw.ReadMessage(nil)
			}, handshake.OOpt())
		require.ErrorIs(t, err, io.EOF)
	})

	scope.Go(func() {
		_, err := handshake.M(c.B, handshake.NormalRun)
		require.ErrorIs(t, err, respErr)
	})

	scope.Wait()
}

func Test_MI_Stage00_NoEncryption(t *testing.T) {
	_Test_MI_Stage00_BadHello1(t, func(h *handshake.Hello1_Payload) {
		h.Pack(false)
	}, handshake.ErrEncryptionRequired)
}

func Test_MI_Stage00_VersionAllZero(t *testing.T) {
	for _, c := range []struct {
		name      string
		encrypted bool
		respErr   error
	}{
		{"encrypted", true, handshake.ErrUnsupportedVersion},
		{"plaintext", false, handshake.ErrEncryptionRequired},
	} {
		t.Run(c.name, func(t *testing.T) {
			_Test_MI_Stage00_BadHello1(t, func(h *handshake.Hello1_Payload) {
				h.ICanEncrypt = c.encrypted
			}, c.respErr)
		})
	}
}

func Test_MI_Stage00_BadVersion(t *testing.T) {
	for _, c := range []struct {
		name      string
		encrypted bool
		respErr   error
	}{
		{"encrypted", true, handshake.ErrUnsupportedVersion},
		{"plaintext", false, handshake.ErrEncryptionRequired},
	} {
		t.Run(c.name, func(t *testing.T) {
			_Test_MI_Stage00_BadHello1(t, func(h *handshake.Hello1_Payload) {
				h.Versions[0] = 0xff
				h.ICanEncrypt = c.encrypted
			}, c.respErr)
		})
	}
}

func _Test_MR_Stage01_BadResp1(
	t *testing.T,
	f func(hello *handshake.Hello1_Payload) *handshake.Resp1_Payload,
	initErr error,
) {
	c := mocktcp.Pair()
	defer c.Close()

	scope := ktest.Scope()

	scope.Go(func() {
		_, err := handshake.M(c.A, handshake.NormalRun, handshake.OOpt())
		require.ErrorIs(t, err, initErr)
	})

	scope.Go(func() {
		_, err := handshake.M(c.B, func(s *handshake.MSession) error {
			var hello1 handshake.Hello1_Payload
			s.Rw.ReadMessage(&hello1)
			resp1 := f(&hello1)
			s.Rw.WriteMessage(resp1)
			return s.Rw.ReadMessage(nil)
		})
		require.ErrorIs(t, err, io.EOF)
	})

	scope.Wait()
}

func Test_MR_Stage01_NoEncryption(t *testing.T) {
	_Test_MR_Stage01_BadResp1(t, func(hello *handshake.Hello1_Payload) *handshake.Resp1_Payload {
		return &handshake.Resp1_Payload{
			ChosenVersion: hello.ChooseVersion(),
			UseEncryption: false,
		}
	}, handshake.ErrEncryptionRequired)
}
