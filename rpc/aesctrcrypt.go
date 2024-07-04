package rpc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"math/rand"
	"strconv"
)

/*
	textStr := []byte("we are ro team")

	aesPass := []byte("wenting123456789")
	retStr, err := rpc.AESCtrEncrypt(textStr, aesPass)
	if err != nil {
		log.Printf("err:%v", err)
		return
	}
	retStr1 := base64.StdEncoding.EncodeToString(retStr)
	log.Printf("en:%v", retStr1)

	plainText,err := rpc.AESCtrDecrypt(retStr, aesPass)
	if err != nil {
		log.Printf("err:%v", err)
		return
	}
	log.Printf("de:%v", string(plainText))
*/

var ErrKeyLen = errors.New("a sixteen or twenty-four or thirty-two length secret key is required")
var ErrIvAes = errors.New("a sixteen-length ivaes is required")
var ErrPaddingSize = errors.New("padding size error please check the secret key or iv")

var ivaes = "wenting123456789"

func GetAESKey(len int) string {
	//key := make([]byte, len)
	//for idx := 0; idx < len; idx++ {
	//	key[idx] = byte(rand.Int31n(256))
	//
	//}
	//retStr := base64.StdEncoding.EncodeToString(key)

	//return "wenting123456789"
	retStr := "wt"
	for idx := 0; idx < len-2; idx++ {
		retStr += strconv.Itoa(int(rand.Int31n(10)))

	}
	ivaes = retStr

	return retStr
}

func AESCtrEncrypt(textStr, key []byte, ivAes ...byte) ([]byte, error) {
	return AESCbcEncrypt(textStr, key, ivAes...)
	//if len(key) != 16 && len(key) != 24 && len(key) != 32 {
	//	return nil, ErrKeyLen
	//}
	//if len(ivAes) != 0 && len(ivAes) != 16 {
	//	return nil, ErrIvAes
	//}
	//
	//block, err := aes.NewCipher(key)
	//if err != nil {
	//	return nil, err
	//}
	//
	//var iv []byte
	//if len(ivAes) != 0 {
	//	iv = ivAes
	//} else {
	//	iv = []byte(ivaes)
	//}
	//
	//data := cipher.NewCTR(block, iv)
	//retText := make([]byte, len(textStr))
	//data.XORKeyStream(retText, textStr)
	//
	//return retText, nil
}

func AESCtrDecrypt(cryptText, key []byte, ivAes ...byte) ([]byte, error) {
	return AESCbcDecrypt(cryptText, key, ivAes...)
	//
	//if key != nil && len(key) != 16 && len(key) != 24 && len(key) != 32 {
	//	log.Println("AESCtrDecrypt key:", key)
	//	return nil, ErrKeyLen
	//}
	//if len(ivAes) != 0 && len(ivAes) != 16 {
	//	return nil, ErrIvAes
	//}
	//
	//block, err := aes.NewCipher(key)
	//if err != nil {
	//	return nil, err
	//}
	//var iv []byte
	//if len(ivAes) != 0 {
	//	iv = ivAes
	//} else {
	//	iv = []byte(ivaes)
	//}
	//
	//data := cipher.NewCTR(block, iv)
	//retText := make([]byte, len(cryptText))
	//data.XORKeyStream(retText, cryptText)
	//return retText, nil
}

func AESCbcEncrypt(plainText, key []byte, ivAes ...byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrKeyLen
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	paddingText := PKCS5Padding(plainText, block.BlockSize())

	var iv []byte
	if len(ivAes) != 0 {
		if len(ivAes) != 16 {
			return nil, ErrIvAes
		} else {
			iv = ivAes
		}
	} else {
		iv = []byte(ivaes)
	} // To initialize the vector, it needs to be the same length as block.blocksize
	blockMode := cipher.NewCBCEncrypter(block, iv)
	cipherText := make([]byte, len(paddingText))
	blockMode.CryptBlocks(cipherText, paddingText)
	return cipherText, nil
}

// decrypt
func AESCbcDecrypt(cipherText, key []byte, ivAes ...byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrKeyLen
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	var iv []byte
	if len(ivAes) != 0 {
		if len(ivAes) != 16 {
			return nil, ErrIvAes
		} else {
			iv = ivAes
		}
	} else {
		iv = []byte(ivaes)
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	paddingText := make([]byte, len(cipherText))
	blockMode.CryptBlocks(paddingText, cipherText)

	plainText, err := PKCS5UnPadding(paddingText)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

func PKCS5Padding(plainText []byte, blockSize int) []byte {
	if blockSize <= 0 {
		return nil
	}
	padding := blockSize - (len(plainText) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	//todo...
	// runtime.slicecopy()导致DATA RACE
	len1 := len(plainText)
	len2 := len(padText)
	newText := make([]byte, len1+len2)
	copy(newText[0:], plainText[:len1])
	copy(newText[len1:], padText[:len2])
	//newText := append(plainText, padText...)
	return newText
}

func PKCS5UnPadding(plainText []byte) ([]byte, error) {
	length := len(plainText)
	if length <= 0 {
		return plainText, nil
	}
	number := int(plainText[length-1])
	if number > length {
		return nil, ErrPaddingSize
	}
	return plainText[:length-number], nil
}
