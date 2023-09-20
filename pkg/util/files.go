/*
 * Copyright (c) 2023 Zander Schwid & Co. LLC.
 * SPDX-License-Identifier: BUSL-1.1
 */

package util

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"os"
	"strings"
)

/**
Parses only os.Unix file mode with 0777 mask
 */
func ParseFileMode(s string) os.FileMode {

	var m uint32

	const rwx = "rwxrwxrwx"
	off := len(s) - len(rwx)
	if off < 0 {
		buf := []byte("---------")
		copy(buf[-off:], s)
		s = string(buf)
	} else {
		s = s[off:]
	}

	for i, c := range rwx {

		if byte(c) == s[i] {
			m |= 1<<uint(9-1-i)
		}

	}

	return os.FileMode(m)
}

func CreateFileIfNeeded(fileName string, fileperm os.FileMode) error {

	_, err := os.Stat(fileName)
	exist := err == nil
	if exist {
		return nil
	}

	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fileperm)
	if err != nil {
		return err
	}
	file.Close()

	return os.Chmod(fileName, fileperm)
}

func CreateDirIfNeeded(dir string, perm os.FileMode) error {
	if _, err := os.Stat(dir); err != nil {
		if err = os.Mkdir(dir, perm); err != nil {
			return errors.Errorf("unable to create dir '%s' with permissions %x, %v", dir, perm ,err)
		}
		if err = os.Chmod(dir, perm); err != nil {
			return errors.Errorf("unable to chmod dir '%s' with permissions %x, %v", dir, perm ,err)
		}
	}
	return nil
}

func RemoveFileIfExist(filePath string) error {
	_, err := os.Stat(filePath)
	exist := err == nil
	if !exist {
		return nil
	}
	return os.Remove(filePath)
}

func copyFile(src string, dst string, perm os.FileMode) (int64, error) {
	inFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer inFile.Close()
	outFile, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return 0, err
	}
	defer outFile.Close()
	return io.Copy(outFile, inFile)
}

func CopyFile(src string, dst string, perm os.FileMode) (int64, error) {
	cnt, err := copyFile(src, dst, perm)
	if err != nil {
		return 0, err
	}
	return cnt, os.Chmod(dst, perm)
}

func IsFileLocked(filePath string) bool {
	if file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_EXCL, 0); err != nil {
		return true
	} else {
		file.Close()
		return false
	}
}

func UnpackGzipFile(inputFilePath, outputFilePath string, filePerm os.FileMode) (int64, error) {
	cnt, err := doUnpackGzipFile(inputFilePath, outputFilePath, filePerm)
	if err != nil {
		return 0, err
	}
	return cnt, os.Chmod(outputFilePath, filePerm)
}

func doUnpackGzipFile(inputFilePath, outputFilePath string, filePerm os.FileMode) (int64, error) {
	inFile, err := os.Open(inputFilePath)
	if err != nil {
		return 0, err
	}
	defer inFile.Close()
	zr, err := gzip.NewReader(inFile)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	outFile, err := os.OpenFile(outputFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm)
	if err != nil {
		return 0, err
	}
	defer outFile.Close()
	return io.Copy(outFile, zr)
}

func UnpackTarGzipFile(fileName string, inputFilePath, outputFilePath string, filePerm os.FileMode) (int64, error) {
	inFile, err := os.Open(inputFilePath)
	if err != nil {
		return 0, err
	}
	defer inFile.Close()
	zr, err := gzip.NewReader(inFile)
	if err != nil {
		return 0, err
	}
	defer zr.Close()

	tarReader := tar.NewReader(zr)
	currentDir := ""
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			return 0, errors.Errorf("file '%s' not found in tar.gz archive '%s'", fileName, inputFilePath)
		}

		if err != nil {
			return 0, err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			currentDir = header.Name
		case tar.TypeReg:
			if strings.TrimPrefix(header.Name, currentDir) == fileName {
				outFile, err := os.OpenFile(outputFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, filePerm)
				if err != nil {
					return 0, err
				}
				written, err := io.CopyN(outFile, zr, header.Size)
				outFile.Close()
				if err != nil {
					return 0, err
				}
				return written, os.Chmod(outputFilePath, filePerm)
			}
		}

	}
}

func BackupFile(filePath string, filePerm os.FileMode) (int64, error) {
	_, err := os.Stat(filePath)
	exist := err == nil
	if !exist {
		return 0, nil
	}
	return CopyFile(filePath, filePath+".bak", filePerm)
}

func AppendFile(filename string, filePerm os.FileMode, format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_RDWR, filePerm)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.WriteString(file, message)
	return err
}

func PackGzipData(plain []byte) ([]byte, error) {

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)

	if _, err := zw.Write(plain); err != nil {
		return nil, err
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func UnpackGzipData(compressed []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewBuffer(compressed))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, zr); err != nil {
		return nil, err
	}
	if err := zr.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
