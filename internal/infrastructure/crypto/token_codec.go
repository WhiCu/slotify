package crypto

import (
	"errors"
	"time"

	"aidanwoods.dev/go-paseto"
)

var (
	ErrTokenMalformed = errors.New("crypto: challenge token malformed")
	ErrTokenExpired   = errors.New("crypto: challenge token expired")
	ErrKeyNotFound    = errors.New("crypto: key not found")
	ErrTypeMismatch   = errors.New("crypto: type mismatch")
)

type TokenCodec struct {
	privateKey paseto.V4SymmetricKey
}

func NewTokenCodec(privateKey paseto.V4SymmetricKey) *TokenCodec {
	return &TokenCodec{privateKey: privateKey}
}

func (sc *TokenCodec) Encode(payload map[string]any, ttl time.Duration) (string, error) {
	token := paseto.NewToken()
	if ttl > 0 {
		token.SetExpiration(time.Now().Add(ttl))
	}

	for key, value := range payload {
		err := token.Set(key, value)
		if err != nil {
			return "", err
		}
	}

	return token.V4Encrypt(sc.privateKey, nil), nil
}

func (sc *TokenCodec) Decode(tokenStr string) (map[string]any, error) {
	parser := paseto.NewParser()

	token, err := parser.ParseV4Local(sc.privateKey, tokenStr, nil)
	if err != nil {
		if err, ok := errors.AsType[paseto.RuleError](err); ok {
			return nil, errors.Join(ErrTokenExpired, err)
		}
		return nil, ErrTokenMalformed
	}

	return token.Claims(), nil
}

func (sc *TokenCodec) DecodeType[T any](tokenStr string, key string) (T, error) {
	var zero T
	claims, err := sc.Decode(tokenStr)
	if err != nil {
		return zero, err
	}
	value, ok := claims[key]
	if !ok {
		return zero, ErrKeyNotFound
	}
	t, ok := value.(T)
	if !ok {
		return zero, ErrTypeMismatch
	}
	return t, nil
}

func (sc *TokenCodec) Decode2Type[T1, T2 any](tokenStr string, key1, key2 string) (T1, T2, error) {
	var zero1 T1
	var zero2 T2
	claims, err := sc.Decode(tokenStr)
	if err != nil {
		return zero1, zero2, err
	}

	var t1 T1
	{
		value, ok := claims[key1]
		if !ok {
			return zero1, zero2, errors.Join(ErrKeyNotFound, errors.New("key: "+key1))
		}
		t1, ok = value.(T1)
		if !ok {
			return zero1, zero2, errors.Join(ErrTypeMismatch, errors.New("key: "+key1))
		}
	}

	var t2 T2
	{
		value, ok := claims[key2]
		if !ok {
			return zero1, zero2, errors.Join(ErrKeyNotFound, errors.New("key: "+key2))
		}
		t2, ok = value.(T2)
		if !ok {
			return zero1, zero2, errors.Join(ErrTypeMismatch, errors.New("key: "+key2))
		}
	}
	return t1, t2, nil
}
