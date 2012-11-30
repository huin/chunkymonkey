package server_auth

import (
	"bufio"
	"errors"
	"expvar"
	"net/http"
	"net/url"
	"time"
)

var (
	expVarServerAuthSuccessCount = expvar.NewInt("server-auth-success-count")
	expVarServerAuthFailCount    = expvar.NewInt("server-auth-fail-count")
	expVarServerAuthTimeNs       = expvar.NewInt("server-auth-time-ns")
)

var (
	HttpResponseError     = errors.New("HTTP response error")
	ResponseTooLargeError = errors.New("HTTP response too large")
)

// An IAuthenticator takes a sessionId and a username string and attempts to
// authenticate against a server. This interface allows for the use of a dummy
// authentication server for testing purposes.
type IAuthenticator interface {
	Authenticate(sessionId, username string) (ok bool, err error)
}

// DummyAuth is a no-op authentication server, always returning the value of
// 'Result'.
type DummyAuth struct {
	Result bool
}

// Authenticate implements the IAuthenticator.Authenticate method
func (d *DummyAuth) Authenticate(sessionId, username string) (authenticated bool, err error) {
	return d.Result, nil
}

// ServerAuth represents authentication against a server, particularly the
// main minecraft server at http://www.minecraft.net/game/checkserver.jsp.
type ServerAuth struct {
	baseUrl url.URL
}

func NewServerAuth(baseUrlStr string) (s *ServerAuth, err error) {
	baseUrl, err := url.Parse(baseUrlStr)
	if err != nil {
		return
	}
	s = &ServerAuth{
		baseUrl: *baseUrl,
	}
	return
}

// buildQuery builds a URL+query string based on a given sessionId and username
// input.
func (s *ServerAuth) buildQuery(sessionId, username string) (query string) {
	queryValues := url.Values{
		// Despite it being called "serverId" in the HTTP request, it actually is a
		// per-connection ID.
		"serverId": {sessionId},
		"user":     {username},
	}

	queryUrl := s.baseUrl
	queryUrl.RawQuery = queryValues.Encode()

	return queryUrl.String()
}

// Authenticate implements the IAuthenticator.Authenticate method
func (s *ServerAuth) Authenticate(sessionId, username string) (authenticated bool, err error) {
	before := time.Now()
	defer func() {
		after := time.Now()
		expVarServerAuthTimeNs.Add(after.Sub(before).Nanoseconds())
		if authenticated {
			expVarServerAuthSuccessCount.Add(1)
		} else {
			expVarServerAuthFailCount.Add(1)
		}
	}()

	authenticated = false

	url_ := s.buildQuery(sessionId, username)

	response, err := http.Get(url_)
	if err != nil {
		return false, err
	}

	if response.StatusCode == 200 {
		lineReader := bufio.NewReader(response.Body)

		for {
			line, isPrefix, err := lineReader.ReadLine()
			if err != nil {
				return false, err
			} else if isPrefix {
				return false, ResponseTooLargeError
			} else if string(line) == "YES" {
				return true, nil
			}
		}
	} else {
		return false, HttpResponseError
	}

	return
}
