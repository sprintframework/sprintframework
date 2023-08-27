/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"strings"
	"syscall"
)

func Promptf(request string, args ...interface{}) string {
	return Prompt(fmt.Sprintf(request, args...))
}

func Prompt(request string) string {
	reader := bufio.NewReader(os.Stdin)
	print(request)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "\"\"\"" { // """
		var out strings.Builder
		for {
			text, _ := reader.ReadString('\n')
			if out.Len() > 0 {
				out.WriteByte('\n')
			}
			text = strings.TrimSpace(text)
			if strings.HasSuffix(text, "\"\"\"") {
				out.WriteString(text[:len(text)-3])
				return out.String()
			} else {
				out.WriteString(text)
			}
		}
	} else {
		return text
	}
}

func PromptQuery(request string) string {
	reader := bufio.NewReader(os.Stdin)
	print(request)
	var out strings.Builder
	for {
		text, _ := reader.ReadString('\n')
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		text = strings.TrimSpace(text)
		if strings.HasSuffix(text, ";") {
			out.WriteString(text[:len(text)-1])
			return out.String()
		} else {
			out.WriteString(text)
		}
	}
}

var (
	pemStart     = "-----BEGIN "
	pemEnd       = "-----END "
	pemEndOfLine = "-----"
)

func PromptPEM(request string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	print(request)
	var out strings.Builder
	for i := 0; ; i++ {
		text, _ := reader.ReadString('\n')
		if i == 0 {
			if !strings.HasPrefix(text, pemStart) {
				return text, errors.Errorf("pem must start from '%s'", pemStart)
			}
		} else {
			out.WriteByte('\n')
		}
		if strings.HasPrefix(text, pemEnd) {
			out.WriteString(text)
			return out.String(), nil
		} else {
			out.WriteString(text)
		}
	}
}

func PromptPassword(request string) string {
	print(request)
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err == nil {
		println()
		password := string(bytePassword)
		return strings.TrimSpace(password)
	} else {
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		return strings.TrimSpace(text)
	}
}
