package cache

import (
	"bytes"
	"errors"
	// log "github.com/sirupsen/logrus"
)

// pkcs5Padding is a pkcs5 padding struct.
type pkcs5 struct{}

var (
	// ErrPaddingSize - represents padding errors
	ErrPaddingSize = errors.New("padding size error")
)
var (
	// PKCS5 represents pkcs5 struct
	PKCS5 = &pkcs5{}
)
var (
	// PKCS7 - difference with pkcs5 only block must be 8
	PKCS7 = &pkcs5{}
)

// Unpadding implements the Padding interface Unpadding method.
func (p *pkcs5) Unpadding(src []byte, blockSize int) ([]byte, error) {
	srcLen := len(src)
	paddingLen := int(src[srcLen-1])
	// log.Debugf("Unpadding: paddingLen %d >= %d srcLen  || paddingLen %d > %d blockSize", paddingLen, srcLen, paddingLen, blockSize)
	if paddingLen >= srcLen || paddingLen > blockSize {
		return nil, ErrPaddingSize
	}
	// log.Debug(src[:srcLen-paddingLen])
	return src[:srcLen-paddingLen], nil
}

// Padding implements the Padding interface Padding method.
func (p *pkcs5) Padding(src []byte, blockSize int) []byte {
	srcLen := len(src)
	padLen := blockSize - (srcLen % blockSize)
	padText := bytes.Repeat([]byte{byte(padLen)}, padLen)
	// log.Debugf("Padding: srcLen: %d", srcLen)
	// log.Debugf("Padding: padLen: %d", padLen)
	// log.Debugf("Padding: padTextLen: %d", len(padText))
	return append(src, padText...)
}
