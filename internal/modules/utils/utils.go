package utils

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/url"
)

// Base64Encode Base64 编码
func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Decode Base64 解码
func Base64Decode(s string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// URLEncode URL 编码
func URLEncode(s string) string {
	return url.QueryEscape(s)
}

// URLDecode URL 解码
func URLDecode(s string) (string, error) {
	decoded, err := url.QueryUnescape(s)
	if err != nil {
		return "", err
	}
	return decoded, nil
}

// MD5Hash MD5 哈希
func MD5Hash(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// SHA256Hash SHA256 哈希
func SHA256Hash(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// HexEncode 十六进制编码
func HexEncode(s string) string {
	return hex.EncodeToString([]byte(s))
}

// HexDecode 十六进制解码
func HexDecode(s string) (string, error) {
	data, err := hex.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
