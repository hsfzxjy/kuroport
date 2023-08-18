package sec

//go:generate msgp

//go:generate stringer -type=SecLevel
type SecLevel byte

const (
	RequireCleartext SecLevel = 1 + iota
	SecWhatever
	RequireEncrypted
)

func (l SecLevel) Accept(useEncryption bool) bool {
	if useEncryption && l <= RequireCleartext ||
		!useEncryption && l >= RequireEncrypted {
		return false
	}
	return true
}

//go:generate stringer -type=AuthLevel
type AuthLevel byte

const (
	RequireNoAuth AuthLevel = 1 + iota
	AuthWhatever
	RequireAuth
)

func (l AuthLevel) Accept(useAuth bool) bool {
	if useAuth && l <= RequireNoAuth ||
		!useAuth && l >= RequireAuth {
		return false
	}
	return true
}
