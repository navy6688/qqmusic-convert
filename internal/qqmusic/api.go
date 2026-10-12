package qqmusic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"qqmusic-convert/internal/audio"
)

const apiURL = "https://u.y.qq.com/cgi-bin/musicu.fcg"

type credentials struct{ uin, token, loginType string }

// Client retains only this batch's validated session in memory.
type Client struct {
	http    *http.Client
	session *credentials
}

func NewClient() *Client {
	return &Client{http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("QQ 音乐接口发生重定向，认证信息未转发")
	}}}
}

func (client *Client) Clear() { client.session = nil }

func (client *Client) Key(ctx context.Context, footer audio.Footer) (string, error) {
	if client.session != nil {
		key, status, err := client.request(ctx, footer, *client.session)
		if err != nil {
			return "", err
		}
		if key != "" {
			return key, nil
		}
		return "", fmt.Errorf("资源 %s 未返回密钥（%s）；请检查登录状态和歌曲权限", footer.Resource, status)
	}
	uin, err := currentAccount()
	if err != nil {
		return "", err
	}
	pids, err := processIDs()
	if err != nil {
		return "", err
	}
	if len(pids) == 0 {
		return "", errors.New("请先打开桌面版 QQ 音乐并登录，再开始转换")
	}
	var accepted *credentials
	var result string
	for _, pid := range pids {
		err = scanTokens(ctx, pid, func(token string) (bool, error) {
			for _, loginType := range []string{"3", "1"} {
				candidate := credentials{uin: uin, token: token, loginType: loginType}
				key, _, err := client.request(ctx, footer, candidate)
				if err != nil {
					return false, err
				}
				if key != "" {
					accepted = &candidate
					result = key
					return true, nil
				}
			}
			return false, nil
		})
		if err != nil {
			return "", err
		}
		if accepted != nil {
			client.session = accepted
			return result, nil
		}
	}
	return "", fmt.Errorf("无法取得资源 %s 的密钥；请重新登录 QQ 音乐或检查客户端版本兼容性", footer.Resource)
}

type apiResponse struct {
	Request struct {
		Code int `json:"code"`
		Song struct {
			Retcode int `json:"retcode"`
			Entries []struct {
				Key     string `json:"ekey"`
				Result  int    `json:"result"`
				Errcode int    `json:"errcode"`
			} `json:"midurlinfo"`
		} `json:"data"`
	} `json:"req_1"`
}

func (client *Client) request(ctx context.Context, footer audio.Footer, auth credentials) (string, string, error) {
	payload := map[string]any{
		"comm": map[string]string{"authst": auth.token, "ct": "19", "cv": "1859", "uin": auth.uin, "tmeLoginType": auth.loginType},
		"req_1": map[string]any{"module": "music.vkey.GetEVkey", "method": "CgiGetEVkey", "param": map[string]any{
			"filename": []string{footer.Resource}, "guid": "10000", "songmid": []string{footer.MID}, "songtype": []int{1},
			"uin": auth.uin, "loginflag": 1, "platform": "27", "ctx": 1,
		}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", errors.New("无法生成 QQ 音乐密钥请求")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return "", "", errors.New("无法建立 QQ 音乐密钥请求")
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "QQMusic/20 QMUnlock")
	response, err := client.http.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
		// Never expose a raw request, response, token or auth-bearing transport error.
		return "", "", errors.New("连接 QQ 音乐密钥接口失败；请检查网络连接（认证信息已隐藏）")
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return "", "", fmt.Errorf("QQ 音乐密钥接口返回 HTTP %d", response.StatusCode)
	}
	const maximum = 2 * 1024 * 1024
	reply, err := io.ReadAll(io.LimitReader(response.Body, maximum+1))
	if err != nil {
		return "", "", errors.New("读取 QQ 音乐接口响应失败")
	}
	if len(reply) > maximum {
		return "", "", errors.New("QQ 音乐接口响应超过 2 MiB")
	}
	var decoded apiResponse
	if err = json.Unmarshal(reply, &decoded); err != nil {
		return "", "", errors.New("QQ 音乐接口响应结构无效（原始响应已隐藏）")
	}
	if len(decoded.Request.Song.Entries) > 0 {
		key := decoded.Request.Song.Entries[0].Key
		if len(key) > 16384 {
			return "", "", errors.New("QQ 音乐返回的密钥长度异常（内容已隐藏）")
		}
		if key != "" {
			return key, "", nil
		}
	}
	return "", fmt.Sprintf("请求状态 %d，曲目状态 %d", decoded.Request.Code, decoded.Request.Song.Retcode), nil
}
