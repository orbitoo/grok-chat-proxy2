package utils

import (
	"bufio"
	"errors"
	"os"
)

func MakeDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}

	}
	return nil
}

func ReadCookies() ([]string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	cookieFile := cwd + "/cookies"
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		errMsg := "cookies file not found"
		return nil, errors.New(errMsg)
	}
	file, err := os.Open(cookieFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var cookies []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cookie := scanner.Text()
		if cookie == "" {
			continue
		}
		cookies = append(cookies, cookie)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cookies, nil
}
