package utils

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// PrivateKeyFromString creates a private key (and does some basic validation) from base64URL encoded public/private key pair.
// The public private key pair is in the same format as Relay's public/private key pair (found in credentials.json).
func PrivateKeyFromString(public string, private string) (ed25519.PrivateKey, error) {
	pk, err := base64.RawURLEncoding.DecodeString(public)
	if err != nil {
		return nil, fmt.Errorf("could not base64decode public key: %v", err)
	}
	sk, err := base64.RawURLEncoding.DecodeString(private)
	if err != nil {
		return nil, fmt.Errorf("could not base64decode private key: %v", err)
	}
	if len(pk) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: %d expected 32", len(pk))
	}
	// Relay serialises the private key without the public part, so we need to append it before we can use it
	if len(pk)+len(sk) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key length: %d expected 64", len(pk)+len(sk))
	}
	var retVal ed25519.PrivateKey = append(sk, pk...)
	return retVal, nil
}

// RelayAuthSign signs the given data for Relay authentication (the data should be the body of the request).
// The signature should be passed with the message in the Relay authentication header (X-Sentry-Relay-Signature).
func RelayAuthSign(privateKey ed25519.PrivateKey, data []byte, timestamp time.Time) (string, error) {
	timestampStr := timestamp.Format(time.RFC3339)
	header := struct {
		T string `json:"t"`
	}{T: timestampStr}
	headerStr, _ := json.Marshal(header)
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerStr)
	messageRaw := append([]byte(headerStr), '\x00')
	messageRaw = append(messageRaw, data...)

	signature := ed25519.Sign(privateKey, messageRaw)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	completeSignature := fmt.Sprintf("%s.%s", encodedSignature, headerEncoded)
	return completeSignature, nil
}
