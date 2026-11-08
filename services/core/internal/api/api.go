// Package api calls wormix-api's internal routes the way the C# server did: POST of a JSON body to
// ApiUrl + route, with X-TCP-SESSION and (when there is one) X-SESSION-KEY.
package api

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/andrij06062003/Wormix/services/core/internal/csx"
	"github.com/andrij06062003/Wormix/services/core/internal/jsonx"
)

// BaseURL is ApiUrl from the configuration; routes are appended to it as is.
var BaseURL string

var (
	// mainClient is used for the main server's commands (HttpClient's default 100 s timeout).
	mainClient = &http.Client{Timeout: 100 * time.Second, Transport: newTransport()}
	// pvpClient is used for the PvP calls (15 s timeout).
	pvpClient = &http.Client{Timeout: 15 * time.Second, Transport: newTransport()}
)

func newTransport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.DisableCompression = true // .NET's HttpClient doesn't ask for gzip
	return t
}

// StatusError is a non-200 answer from the API.
type StatusError struct {
	Status int
	Body   []byte
}

func (e *StatusError) Error() string { return fmt.Sprintf("API answered %d: %s", e.Status, e.Body) }

func post(client *http.Client, url string, body []byte, tcpSession string, token *string) (int, []byte, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("X-TCP-SESSION", tcpSession)
	if token != nil {
		req.Header.Set("X-SESSION-KEY", *token)
	}
	req.Header["User-Agent"] = []string{""} // no User-Agent, like .NET's HttpClient
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

// isNullOrWhiteSpace is string.IsNullOrWhiteSpace.
func isNullOrWhiteSpace(s *string) bool {
	if s == nil {
		return true
	}
	for _, r := range *s {
		switch r {
		case ' ', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000:
			continue
		}
		if r >= 0x2000 && r <= 0x200A {
			continue
		}
		return false
	}
	return true
}

// Post is HttpProcessor.PostRequest: a main-server call on behalf of a session. A non-200 answer or a body that
// isn't JSON panics (the C# server threw, which closed the connection). The session key header is sent only
// when the session has a non-blank token.
func Post(route string, body []byte, sessionID string, token *string) *jsonx.Value {
	if isNullOrWhiteSpace(token) {
		token = nil
	}
	status, data, err := post(mainClient, BaseURL+route, body, sessionID, token)
	if err != nil {
		csx.Throw(csx.Generic, "HTTP request to %s failed: %v", route, err)
	}
	if status != http.StatusOK {
		csx.Throw(csx.Generic, "Http Request Error [%d] from %s", status, route)
	}
	v, err := jsonx.Parse(data)
	if err != nil {
		csx.Throw(csx.Generic, "bad JSON from %s: %v", route, err)
	}
	return v
}

// PostRefusable is Post for a command the API may refuse (a 4xx answer such as StartBattle's "Not enough
// battles"): it returns refused=true instead of throwing, so the session can answer instead of dropping.
func PostRefusable(route string, body []byte, sessionID string, token *string) (v *jsonx.Value, refused bool) {
	if isNullOrWhiteSpace(token) {
		token = nil
	}
	status, data, err := post(mainClient, BaseURL+route, body, sessionID, token)
	if err != nil {
		csx.Throw(csx.Generic, "HTTP request to %s failed: %v", route, err)
	}
	if status >= 400 && status < 500 {
		return nil, true
	}
	if status != http.StatusOK {
		csx.Throw(csx.Generic, "Http Request Error [%d] from %s", status, route)
	}
	v, err = jsonx.Parse(data)
	if err != nil {
		csx.Throw(csx.Generic, "bad JSON from %s: %v", route, err)
	}
	return v, false
}

// PostPvP is PvpApi.Post: a call acting as one player, identified by the session token only. Returns a
// *StatusError for non-200 answers.
func PostPvP(route string, token string, body []byte) (*jsonx.Obj, error) {
	status, data, err := post(pvpClient, BaseURL+route, body, "pvp", &token)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, &StatusError{Status: status, Body: data}
	}
	v, err := jsonx.Parse(data)
	if err != nil {
		return nil, err
	}
	if v.Kind != jsonx.Object {
		return nil, fmt.Errorf("API answer is not a JSON object")
	}
	return v.Obj, nil
}
