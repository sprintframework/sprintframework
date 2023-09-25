/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package sprintutils

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"github.com/golang-jwt/jwt"
	"github.com/pkg/errors"
	"github.com/codeallergy/base62"
	"github.com/sprintframework/sprint"
	"io"
	"strconv"
	"strings"
)

var (
	DefaultTokenSize = 32 // 256-bit AES key
	DefaultLongIdSize = 16 // 128-bit

	NodeIdBits = 48
	NodeIdSize = NodeIdBits / 8

	Encoding = base64.RawURLEncoding
)

func GenerateLongId() (string, error) {
	nonce := make([]byte, DefaultLongIdSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err == nil {
		return base62.StdEncoding.EncodeToString(nonce), nil
	} else {
		return "", err
	}
}

func EncodeLongId(id []byte) string {
	return base62.StdEncoding.EncodeToString(id)
}

func DecodeLongId(base62str string) ([]byte, error) {
	return base62.StdEncoding.DecodeString(base62str)
}

func EncodeId(id uint64) string {
	return base62.StdEncoding.EncodeUint64(id)
}

func DecodeId(base62str string) (uint64, error) {
	return base62.StdEncoding.DecodeToUint64(base62str)
}

func GenerateToken() (string, error) {
	nonce := make([]byte, DefaultTokenSize)
	for {
		if _, err := io.ReadFull(rand.Reader, nonce); err == nil {
			key := Encoding.EncodeToString(nonce)
			if !strings.ContainsAny(key, "-_") {
				return key, nil
			}
		} else {
			return "", err
		}
	}
}

func ParseToken(base64key string) ([]byte, error) {
	key, err := Encoding.DecodeString(base64key)
	if err != nil {
		return key, err
	}
	if len(key) != DefaultTokenSize {
		return key, errors.Errorf("wrong token size %d, expected %d", len(key), DefaultTokenSize)
	}
	return key, nil
}

func GenerateNodeId() (string, error) {

	blob := make([]byte, NodeIdSize)
	if _, err := io.ReadFull(rand.Reader, blob); err == nil {
		return "0x" + hex.EncodeToString(blob), nil
	} else {
		return "", err
	}

}

func ParseNodeId(nodeId string) (uint64, error) {
	if strings.HasPrefix(nodeId, "0x") {
		nodeId = nodeId[2:]
	}
	return strconv.ParseUint(nodeId, 16, NodeIdBits)
}

type UserClaims struct {
	Roles     []string            `json:"roles"`
	Context   map[string]string   `json:"ctx"`
	jwt.StandardClaims
}

func GenerateAuthToken(secretKey []byte, user *sprint.AuthorizedUser) (string, error) {

	if secretKey == nil {
		return "", errors.New("empty secretKey")
	}

	if user == nil {
		return "", errors.New("empty user")
	}

	var roles []string
	for key, _ := range user.Roles {
		roles = append(roles, key)
	}

	claims := &UserClaims{
		StandardClaims: jwt.StandardClaims{
			Id:        user.Username,
			ExpiresAt: user.ExpiresAt,
		},
		Roles:   roles,
		Context: user.Context,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)

}

func VerifyAuthToken(secretKey []byte, jwtToken string) (*sprint.AuthorizedUser, error) {

	if secretKey == nil {
		return nil, errors.New("empty secretKey")
	}

	var claims UserClaims

	token, err := jwt.ParseWithClaims(jwtToken, &claims, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	decodedToken, _ := Encoding.DecodeString(jwtToken)

	if err != nil {
		return nil, errors.Errorf("wrong jwt token '%s'", decodedToken)
	}

	if !token.Valid {
		return nil, errors.Errorf("expired jwt token '%s'", decodedToken)
	}

	indexedRoles := make(map[string]bool)
	for _, role := range claims.Roles {
		indexedRoles[role] = true
	}

	return &sprint.AuthorizedUser{
		Username:  claims.Id,
		Roles:     indexedRoles,
		Context:   claims.Context,
		ExpiresAt: claims.ExpiresAt,
		Token:     jwtToken,
	}, nil
}
