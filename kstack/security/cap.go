package sec

//go:generate msgp

type SecCap bool

func (c SecCap) Normalize(level SecLevel) (normed SecLevel, ok bool) {
	if !c {
		if level >= RequireEncrypted {
			ok = false
		} else {
			normed = RequireCleartext
			ok = true
		}
		return
	} else {
		normed = level
		ok = true
		return
	}
}

func (c SecCap) Decide(peerLevel SecLevel) (useEncryption bool, ok bool) {
	if !c {
		if peerLevel >= RequireEncrypted {
			return false, false
		}
		return false, true
	} else {
		if peerLevel <= RequireCleartext {
			return false, true
		}
		return true, true
	}
}

type AuthCap bool

func (c AuthCap) Normalize(level AuthLevel) (normed AuthLevel, ok bool) {
	if !c {
		if level >= RequireAuth {
			ok = false
		} else {
			normed = RequireNoAuth
			ok = true
		}
		return
	} else {
		normed = level
		ok = true
		return
	}
}

func (c AuthCap) Decide(peerLevel AuthLevel) (useAuth bool, ok bool) {
	if !c {
		if peerLevel >= RequireAuth {
			return false, false
		}
		return false, true
	} else {
		if peerLevel <= RequireNoAuth {
			return false, true
		}
		return true, true
	}
}
