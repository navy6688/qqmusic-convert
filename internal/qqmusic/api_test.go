package qqmusic

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"qqmusic-convert/internal/audio"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestAPIRequestAndResponsePrivacy(t *testing.T) {
	client := NewClient()
	secret := strings.Repeat("private-auth-", 3)
	client.http.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != apiURL {
			t.Fatal("authentication sent to unexpected URL")
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), secret) || !strings.Contains(string(body), "resource.mflac") {
			t.Fatal("missing expected API fields")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"req_1":{"code":1,"data":{"retcode":405,"authst":"` + secret + `"}}}`))}, nil
	})
	key, status, err := client.request(context.Background(), audio.Footer{MID: "mid", Resource: "resource.mflac"}, credentials{uin: "123", token: secret, loginType: "3"})
	if err != nil || key != "" {
		t.Fatalf("unexpected response key=%d err=%v", len(key), err)
	}
	if strings.Contains(status, secret) {
		t.Fatal("API status leaked authentication")
	}
}

func TestMalformedAPIResponseHidesOriginalBody(t *testing.T) {
	client := NewClient()
	secret := "actual-secret-should-not-appear"
	client.http.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(secret))}, nil
	})
	_, _, err := client.request(context.Background(), audio.Footer{}, credentials{})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatal("malformed response leaked body or was accepted")
	}
}

func TestTokenTextFormats(t *testing.T) {
	token := strings.Repeat("A", 32)
	plain := `"authst":"` + token + `"`
	escaped := `\"authst\":\"` + token + `\"`
	wide := make([]byte, 0, len(plain)*2)
	for _, value := range []byte(plain) {
		wide = append(wide, value, 0)
	}
	for _, chunk := range [][]byte{[]byte(plain), []byte(escaped), wide} {
		found := tokenCandidates(chunk)
		if len(found) == 0 || found[0] != token {
			t.Fatal("token encoding not supported")
		}
	}
}
