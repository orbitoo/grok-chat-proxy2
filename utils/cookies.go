package utils

import "net/http"

func ParseCookies(cookieString string) []*http.Cookie {
	header := http.Header{}
	header.Add("Cookie", cookieString)
	req := http.Request{Header: header}
	return req.Cookies()
}