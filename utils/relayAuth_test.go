package utils

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestRelayAuthSign(t *testing.T) {
	var pkString = "ftFuDNBFm8-kPCoCaaWMio_mJYC2txJuCtwSeHn2vv0"
	var skString = "uZUtRrryN8jybTTOjbs5EDfqWNwyDfEng4TSRa6Ifhs"
	var data = "hello"
	var timestamp = time.Date(2022, 10, 10, 10, 10, 10, 0, time.UTC)
	// this was generated by relay-auth.rs (relay library) for the above data
	var expectedSignature = "fI9HUkBnG_spOO3GuflscY0LXNuaMxxELsaaPo0KTrfnKfoXaHUibfFto-JvAU8ySbjKVA_Gmi1kw1AjnDsvAw.eyJ0IjoiMjAyMi0xMC0xMFQxMDoxMDoxMFoifQ"

	var privateKey, err = PrivateKeyFromString(pkString, skString)

	if err != nil {
		t.Fatalf("Could not decode private key: %v", err)
	}
	var dataRaw = []byte(data)
	signature, err := RelayAuthSign(privateKey, dataRaw, timestamp)
	if err != nil {
		t.Fatalf("Could not sign data: %v", err)
	}
	if signature != expectedSignature {
		t.Fatalf("Signature does not match expected: %s", signature)
	}
}

// TestAuthFunctionality is a pseudo test that breaks the signing functionality into pieces
// so that it can be checked against Relay (DO NOT DELETE).
func TestAuthFunctionality(t *testing.T) {
	fmt.Println("Go sign test")
	var pkString = "ftFuDNBFm8-kPCoCaaWMio_mJYC2txJuCtwSeHn2vv0"
	var skString = "uZUtRrryN8jybTTOjbs5EDfqWNwyDfEng4TSRa6Ifhs"

	fmt.Printf("public key string: %s\n", pkString)
	fmt.Printf("private key string: %s\n", skString)

	var err error
	var pkDecoded []byte
	pkDecoded, err = base64.RawURLEncoding.DecodeString(pkString)

	if err != nil {
		t.Error("Could not decode public key")
	}

	var skDecoded []byte
	skDecoded, err = base64.RawURLEncoding.DecodeString(skString)

	fmt.Printf("public key decoded: %v\n", pkDecoded)
	fmt.Printf("private key decoded: %v\n", skDecoded)

	var pk ed25519.PublicKey = pkDecoded
	var sk ed25519.PrivateKey = append(skDecoded, pkDecoded...)

	fmt.Printf("private key: %v\n", sk)
	fmt.Printf("public key: %v\n", pk)
	fmt.Printf("public key from private key: %v\n", sk.Public())

	pk2, sk2, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Error("Could not generate key")
	}
	fmt.Printf("generated private len=%d key:  %v\n", len(sk2), sk2)
	fmt.Printf("generated public len=%d key: %v\n", len(pk2), pk2)
	fmt.Printf("public generated from private len=%d key: %v\n", len(sk), sk2.Public())

	data := "hello"
	dataRaw := []byte(data)
	date := time.Date(2022, 10, 10, 10, 10, 10, 0, time.UTC)
	dateStr := date.Format(time.RFC3339)
	fmt.Printf("date: %s\n", dateStr)
	header := struct {
		T string `json:"t"`
	}{T: dateStr}
	headerStr, _ := json.Marshal(header)
	fmt.Printf("Header packed as string: %s\n", headerStr)
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerStr)
	fmt.Printf("Header encoded as string: %s\n", headerEncoded)
	var messageRaw []byte = []byte(headerEncoded)
	messageRaw = append(headerStr, '\x00')
	messageRaw = append(messageRaw, dataRaw...)
	fmt.Printf("Message raw %v\n", messageRaw)
	signedMessage := ed25519.Sign(sk, messageRaw)
	fmt.Printf("Signed message: %v\n", signedMessage)
	signedEncoded := base64.RawURLEncoding.EncodeToString(signedMessage)
	fmt.Printf("Signed message encoded: %s.%s\n", signedEncoded, headerEncoded)
}

/*
// Equivalent Rust test, please DO NOT DELETE.
use std::str;
#[test]
fn test_relay_auth_signature_generation() {
    println!("Rust sign test:");
    let pk_string = "ftFuDNBFm8-kPCoCaaWMio_mJYC2txJuCtwSeHn2vv0";
    let sk_string = "uZUtRrryN8jybTTOjbs5EDfqWNwyDfEng4TSRa6Ifhs";
    println!("public key string: {}", pk_string);
    println!("secret key string: {}", sk_string);

    let pk_decoded = base64::decode_config(pk_string, base64::URL_SAFE_NO_PAD).unwrap();
    let sk_decoded = base64::decode_config(sk_string, base64::URL_SAFE_NO_PAD).unwrap();
    println!("public key decoded: {:?}", pk_decoded);
    println!("secret key decoded: {:?}", sk_decoded);

    let pk = PublicKey::from_str(pk_string).unwrap();
    let sk = SecretKey::from_str(sk_string).unwrap();
    println!("public key as bytes: {:?}", pk.inner.to_bytes());
    println!("secret key as bytes: {:?}", sk.inner.to_bytes());

    let data = "hello";
    let some_date = NaiveDate::from_ymd(2022, 10, 10).and_hms(10, 10, 10);
    let utc_date = DateTime::<Utc>::from_utc(some_date, Utc);
    let header = SignatureHeader {
        timestamp: Some(utc_date),
    };

    let signature = sk.sign_with_header(data.as_bytes(), &header);
    println!("signed message signature: {}", signature);
    println!("Doing manual signature");
    let mut header_raw =
        serde_json::to_vec(&header).expect("attempted to pack non json safe header");
    println!(
        "Header packed as string: {}",
        str::from_utf8(&header_raw).unwrap()
    );
    let header_encoded = base64::encode_config(&header_raw[..], base64::URL_SAFE_NO_PAD);
    println!("Header encoded:{}", header_encoded);
    header_raw.push(b'\x00');
    header_raw.extend_from_slice(data.as_bytes());
    println!("Message raw {:?}", header_raw);
    let sig = sk.inner.sign::<Sha512>(&header_raw);
    println!("Signature {:?}", sig);
    let mut sig_encoded = base64::encode_config(&sig.to_bytes()[..], base64::URL_SAFE_NO_PAD);
    sig_encoded.push('.');
    sig_encoded.push_str(&header_encoded);
    println!("manual signature: {}", sig_encoded)
}
*/
