package apns

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"time"
)

// https://developer.apple.com/library/content/documentation/NetworkingInternet/Conceptual/RemoteNotificationsPG/CommunicatingwithAPNs.html#//apple_ref/doc/uid/TP40008194-CH11-SW1

const jwtDefaultGrowSize = 256

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type jwtClaim struct {
	Iss string `json:"iss"`
	Iat int64  `json:"iat"`
}

type ecdsaSignature struct {
	R, S *big.Int
}

func CreateJWT(key []byte, kid string, teamID string, now time.Time) (string, error) {
	var b bytes.Buffer
	b.Grow(jwtDefaultGrowSize)

	header := jwtHeader{
		Alg: "ES256",
		Kid: kid,
	}
	headerJSON, err := json.Marshal(&header)
	if err != nil {
		return "", err
	}
	if err := writeAsBase64(&b, headerJSON); err != nil {
		return "", err
	}
	b.WriteByte(byte('.'))

	claim := jwtClaim{
		Iss: teamID,
		Iat: now.Unix(),
	}
	claimJSON, err := json.Marshal(&claim)
	if err != nil {
		return "", err
	}
	if err := writeAsBase64(&b, claimJSON); err != nil {
		return "", err
	}

	sig, err := createSignature(b.Bytes(), key)
	if err != nil {
		return "", err
	}
	b.WriteByte(byte('.'))

	if err := writeAsBase64(&b, sig); err != nil {
		return "", err
	}

	return b.String(), nil
}

func writeAsBase64(w io.Writer, byt []byte) error {
	enc := base64.NewEncoder(base64.RawURLEncoding, w)
	defer enc.Close()

	if _, err := enc.Write(byt); err != nil {
		return err
	}
	return nil
}

func createSignature(payload []byte, key []byte) ([]byte, error) {
	h := crypto.SHA256.New()
	if _, err := h.Write(payload); err != nil {
		return nil, err
	}
	msg := h.Sum(nil)

	block, _ := pem.Decode(key)
	p8key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	r, s, err := ecdsa.Sign(rand.Reader, p8key.(*ecdsa.PrivateKey), msg)
	if err != nil {
		return nil, err
	}

	sig, err := asn1.Marshal(ecdsaSignature{r, s})
	if err != nil {
		return nil, err
	}

	return sig, nil
}
