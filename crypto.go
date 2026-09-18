package snow

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func newCryptoModule(i *Interp, name string) *Module {
	m := &Module{Name: name, Dict: NewDict(8)}
	m.Dict.Set("sha256", Native(cryptoSha256))
	m.Dict.Set("sha1", Native(cryptoSha1))
	m.Dict.Set("md5", Native(cryptoMd5))
	m.Dict.Set("random_token", Native(cryptoRandomToken))
	m.Dict.Set("jwt_sign", Native(cryptoJWTSign))
	m.Dict.Set("jwt_verify", Native(cryptoJWTVerify))
	return m
}

func cryptoSha256(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("crypto.sha256 expects 1 string argument")
	}
	s, err := strArg(args[0], "crypto.sha256")
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256([]byte(s))
	return []Val{Str(hex.EncodeToString(h[:]))}, nil
}

func cryptoSha1(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("crypto.sha1 expects 1 string argument")
	}
	s, err := strArg(args[0], "crypto.sha1")
	if err != nil {
		return nil, err
	}
	h := sha1.Sum([]byte(s))
	return []Val{Str(hex.EncodeToString(h[:]))}, nil
}

func cryptoMd5(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 1 {
		return nil, fmt.Errorf("crypto.md5 expects 1 string argument")
	}
	s, err := strArg(args[0], "crypto.md5")
	if err != nil {
		return nil, err
	}
	h := md5.Sum([]byte(s))
	return []Val{Str(hex.EncodeToString(h[:]))}, nil
}

func cryptoRandomToken(i *Interp, args []Val) ([]Val, error) {
	n := 16 // 16 bytes = 32 hex chars by default
	if len(args) >= 1 {
		if v, ok := args[0].(Int); ok && v > 0 {
			n = int((v + 1) / 2)
		}
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	out := hex.EncodeToString(b)
	if len(args) >= 1 {
		if v, ok := args[0].(Int); ok && int(v) < len(out) {
			out = out[:int(v)]
		}
	}
	return []Val{Str(out)}, nil
}

// crypto.jwt_sign(payload_dict, secret) -> token
func cryptoJWTSign(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("crypto.jwt_sign expects (payload, secret)")
	}
	secret, err := strArg(args[1], "crypto.jwt_sign secret")
	if err != nil {
		return nil, err
	}
	headerJSON := `{"alg":"HS256","typ":"JWT"}`
	headerB64 := base64.RawURLEncoding.EncodeToString([]byte(headerJSON))

	payloadJSON, err := EncodeJSON(args[0])
	if err != nil {
		return nil, fmt.Errorf("crypto.jwt_sign invalid payload: %v", err)
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString([]byte(payloadJSON))

	signingInput := headerB64 + "." + payloadB64
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	sigB64 := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	token := signingInput + "." + sigB64
	return []Val{Str(token)}, nil
}

// crypto.jwt_verify(token, secret) -> payload or nil
func cryptoJWTVerify(i *Interp, args []Val) ([]Val, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("crypto.jwt_verify expects (token, secret)")
	}
	tokenStr, err := strArg(args[0], "crypto.jwt_verify token")
	if err != nil {
		return nil, err
	}
	secret, err := strArg(args[1], "crypto.jwt_verify secret")
	if err != nil {
		return nil, err
	}

	parts := strings.Split(string(tokenStr), ".")
	if len(parts) != 3 {
		return []Val{Nil}, nil
	}

	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return []Val{Nil}, nil
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingInput))
	expectedSig := mac.Sum(nil)

	if !hmac.Equal(sig, expectedSig) {
		return []Val{Nil}, nil
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return []Val{Nil}, nil
	}

	val, err := DecodeJSON(string(payloadBytes))
	if err != nil {
		return []Val{Nil}, nil
	}

	// Check exp claim if present in dict
	if d, ok := val.(*Dict); ok {
		if expVal, hasExp := d.Get("exp"); hasExp {
			var expSec int64
			switch ev := expVal.(type) {
			case Int:
				expSec = int64(ev)
			case Float:
				expSec = int64(ev)
			}
			if expSec > 0 && time.Now().Unix() > expSec {
				return []Val{Nil}, nil // Expired
			}
		}
	}

	return []Val{val}, nil
}
