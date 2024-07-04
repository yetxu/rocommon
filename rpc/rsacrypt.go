package rpc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"os"
)

const (
	privateKey = `-----BEGIN RSA PRIVATE KEY-----
MIICWwIBAAKBgQCTnWuCTLNtDiqCt6fEfnLRUGT4zrPPgL1l8alZBcgdIC8ErtqxOZLFjVTYqxE8dqnkyBhW9pjv2WodAf9o0D5Em0Ysx2I8yurWBGmvpxLIaMaqqIPuKBYJSzJkv2wht5eXrUpCJBxn/0kFSBBLvvq/9NWWDniVA71NQaGPUal/DQIBAwKBgBiaPJW3czzXscBz8UtqaHg4ENQic01AH5D9nDmA9q+FXStyecg0QyDs43lx2DS+caYhWWPTxCf5ka+AVTwitQsuDkQ/f9bLvxaqCqhZck2ph0Bb/N+CYKU5jgy88BNZjLvjdLBTjBeVQjk57ofLS6r9mn+QXF4z+fpnIEJrbX7LAkEA1/DMrghmNYuVWK5BKQWJzBkKS4k/ef7Gh8QFNyQ8pV+xExNK2T0BjmZH+uA6Sigkn3otqj7fnB3AtbNB5SDifQJBAK7/xpxazc7kuK97fGVfbKOCHjUNcZ/TY2oaExqncPjrf0V61VWW2PFVZfGY4rEMmWO8awIPgC/DriEsvuf3o9ECQQCP9d3JWu7OXQ47HtYbWQaIELGHsNT7/y8FLVjPbX3DlSC3YjHmKKu0RC/8lXwxcBhqUXPG1JUSvoB5IivuFexTAkB0qoRoPIk0mHsfp6hDlPMXrBQjXku/4kJGvAy8b6Cl8lTY/I45DztLjkP2Zex2CGZCfZysClV1LR7AyH9FT8KLAkEAvxsN59kyXjRrbyRcMzSPrBcVFgLfmFyPQZKc8+BgRENtxPM8+WRLIMgMzVh3Sh175kKNKUDeacpzu1uiaHt6VA==
-----END RSA PRIVATE KEY-----
`
	publicKey = `-----BEGIN PUBLIC KEY-----
MIGdMA0GCSqGSIb3DQEBAQUAA4GLADCBhwKBgQCTnWuCTLNtDiqCt6fEfnLRUGT4zrPPgL1l8alZBcgdIC8ErtqxOZLFjVTYqxE8dqnkyBhW9pjv2WodAf9o0D5Em0Ysx2I8yurWBGmvpxLIaMaqqIPuKBYJSzJkv2wht5eXrUpCJBxn/0kFSBBLvvq/9NWWDniVA71NQaGPUal/DQIBAw==
-----END PUBLIC KEY-----
`
	//私钥是用PKCS8来生成的
	privateClientKey = `-----BEGIN PRIVATE KEY-----
MIICdQIBADANBgkqhkiG9w0BAQEFAASCAl8wggJbAgEAAoGBAI0sUcw92U5v59lbWrOBNj6kNNG1bFjLJ3eKrMjBpR7vZY6yDkH/qQciXLKqYZdF2slUrgHNKLEnoQOaQEFfVw0uyc2juzb4gDGl+P0TCIhLJzlzyWP83d7p/ehCpcp34i21FmjDTEE/2O6k5FfytxbKFm33iaxd0CtNGl0TvAHXAgEDAoGAF4di91+kN71RTuSPHereX8YIzZ48uXcxPpcczCBGL9KQ7R2tCv/xgTBkyHG67oukduNyVaIxctvwK0RgCuU5LJxM4phwr+YOUgfiv0qjEqzrj10YqQ4C5qc1JQPs/q1BhSJp8nY5AgrUtBvZWsyM1i+A9nDlZqpRAn2y6+LaXGMCQQDrkcn4n/clnCmnt84IWhtO6quWspPy2qYf4pirvAd2w019WQdD1IFbptYjIzMj7x4HXYEtUob5apB6CFzuJuNvAkEAmWq0GHcgbo4bzvCzStv87rcxroLZtfHQ3txxf1vK8ZXBaUGyuiGzozXf2qkEA3rzpbMmJNa9Zn+L4OB41Hb0GQJBAJ0L2/sVT25oG8UlNAWRZ4nxx7nMYqHnGWqXEHJ9Wk8s3lOQr4KNq5JvOWzCIhf0vq+Tq3OMWfucYFFa6J7El58CQGZHIrr6FZ8JZ99LIjHn/fR6IR8B5nlL4JSS9lTn3KEOgPDWdybBImzOlTxwrVenTRkiGW3kfkRVB+tAUI2korsCQQCbEQHa0XbFjA230nejo8y1umltCvtD1eeomzblXLSLPqwenqd380B1vkZEUaSDafmo248THmWOfDom6T/hmvvW
-----END PRIVATE KEY-----`
	publicClientKey = `-----BEGIN PUBLIC KEY-----
MIGdMA0GCSqGSIb3DQEBAQUAA4GLADCBhwKBgQCNLFHMPdlOb+fZW1qzgTY+pDTRtWxYyyd3iqzIwaUe72WOsg5B/6kHIlyyqmGXRdrJVK4BzSixJ6EDmkBBX1cNLsnNo7s2+IAxpfj9EwiISyc5c8lj/N3e6f3oQqXKd+IttRZow0xBP9jupORX8rcWyhZt94msXdArTRpdE7wB1wIBAw==
-----END PUBLIC KEY-----`

	privateServerKey = ``

	publicServerKey = ``
)

const (
	privateKeyPrefix = "WT RSA PRIVATE KEY "
	publicKeyPrefix  = " WT  RSA PUBLIC KEY "
)

var PublicKey []byte
var PrivateKey []byte

var PrivateClientKey []byte
var PublicClientKey []byte
var PrivateServerKey []byte
var PublicServerKey []byte

func init() {
	PublicKey = []byte(publicKey)
	PrivateKey = []byte(privateKey)

	PrivateClientKey = []byte(privateClientKey)
	PublicClientKey = []byte(publicClientKey)
	PrivateServerKey = []byte(privateServerKey)
	PublicServerKey = []byte(publicServerKey)
}

func GetRSAKey(prefix string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return err
	}

	//客户端需要PKCS8格式的私钥
	x509PrivateKey, err := Marsha1PKCS8PrivateKey(privateKey)
	if err != nil {
		return err
	}
	//x509PrivateKey := x509.MarshalPKCS1PrivateKey(privateKey)
	privateFile, err := os.Create("./private" + prefix + ".pem")
	if err != nil {
		return err
	}
	defer privateFile.Close()
	privateBlock := pem.Block{
		Type:  privateKeyPrefix,
		Bytes: x509PrivateKey,
	}

	if err = pem.Encode(privateFile, &privateBlock); err != nil {
		return err
	}

	publicKey := privateKey.PublicKey
	x509PublicKey, err := x509.MarshalPKIXPublicKey(&publicKey)
	if err != nil {
		panic(err)
	}
	publicFile, err := os.Create("./public" + prefix + ".pem")
	if err != nil {
		return err
	}
	defer publicFile.Close()
	publicBlock := pem.Block{
		Type:  publicKeyPrefix,
		Bytes: x509PublicKey,
	}
	if err = pem.Encode(publicFile, &publicBlock); err != nil {
		return err
	}

	return nil
}

func RSAEncrypt(textStr, key []byte) (cryptText []byte, err error) {
	block, _ := pem.Decode(key)

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	tmpPublicKey := publicKeyInterface.(*rsa.PublicKey)

	tmpRetText, err := rsa.EncryptPKCS1v15(rand.Reader, tmpPublicKey, textStr)
	if err != nil {
		return nil, err
	}

	retText := base64.StdEncoding.EncodeToString(tmpRetText)

	return []byte(retText), nil
}

func RSADecrypt(cryptText, key []byte) ([]byte, error) {
	block, _ := pem.Decode(key)
	tmpPrivateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	//block, _ := pem.Decode(key)
	//tmpPrivateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	//if err != nil {
	//	return nil, err
	//}
	tmpCryptText, err := base64.StdEncoding.DecodeString(string(cryptText))
	if err != nil {
		return nil, err
	}

	retText, err := rsa.DecryptPKCS1v15(rand.Reader, tmpPrivateKey.(*rsa.PrivateKey), tmpCryptText)
	if err != nil {
		return nil, err
	}
	return retText, nil
}

type pkcs8Key struct {
	Version             int
	PrivateKeyAlgorithm []asn1.ObjectIdentifier
	PrivateKey          []byte
}

func Marsha1PKCS8PrivateKey(key *rsa.PrivateKey) ([]byte, error) {
	var pkey pkcs8Key
	pkey.Version = 0
	pkey.PrivateKeyAlgorithm = make([]asn1.ObjectIdentifier, 1)
	pkey.PrivateKeyAlgorithm[0] = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}
	pkey.PrivateKey = x509.MarshalPKCS1PrivateKey(key)

	return asn1.Marshal(pkey)
}
