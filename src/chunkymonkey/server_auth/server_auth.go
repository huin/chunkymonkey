package server_auth

import (
	"expvar"
	"net/http"
	"time"
	"net/url"
	"io"
)

var (
	expVarServerAuthSuccessCount *expvar.Int
	expVarServerAuthFailCount    *expvar.Int
	expVarServerAuthTimeNs       *expvar.Int
)

func init() {
	expVarServerAuthSuccessCount = expvar.NewInt("server-auth-success-count")
	expVarServerAuthFailCount = expvar.NewInt("server-auth-fail-count")
	expVarServerAuthTimeNs = expvar.NewInt("server-auth-time-ns")
}

// An IAuthenticator takes a serverId and a user string and attempts to
// authenticate against a server. This interface allows for the use of a dummy
// authentication server for testing purposes.
type IAuthenticator interface {
	Authenticate(string, string) (bool, error)
}

// DummyAuth is a no-op authentication server, always returning the value of
// 'Valid'.
type DummyAuth struct {
	Result bool
}

// Authenticate implements the IAuthenticator.Authenticate method
func (d *DummyAuth) Authenticate(serverId, user string) (authenticated bool, err error) {
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

// Build a URL+query string based on a given server URL, serverId and user
// input
func (s *ServerAuth) BuildQuery(serverId, user string) (query string) {
	queryValues := url.Values{
		"serverId": {serverId},
		"user":     {user},
	}

	queryUrl := s.baseUrl
	queryUrl.RawQuery = queryValues.Encode()

	return queryUrl.String()
}

// Authenticate implements the IAuthenticator.Authenticate method
func (s *ServerAuth) Authenticate(serverId, user string) (authenticated bool, err error) {
	before := time.Now()
	defer func() {
		after := time.Now()
		expVarServerAuthTimeNs.Add(after.Unix() - before.Unix())
		if authenticated {
			expVarServerAuthSuccessCount.Add(1)
		} else {
			expVarServerAuthFailCount.Add(1)
		}
	}()

	authenticated = false

	url_ := s.BuildQuery(serverId, user)

	response, err := http.Get(url_)
	if err != nil {
		return
	}

	if response.StatusCode == 200 {
		// We only need to read up to 3 bytes for "YES" or "NO"
		buf := make([]byte, 3)
		bufferPos := 0
		var numBytesRead int

		for err == nil && bufferPos < 3 {
			numBytesRead, err = response.Body.Read(buf[bufferPos:])
			if err != nil && err != io.EOF {
				return
			}
			bufferPos += numBytesRead
		}

		result := string(buf[0:bufferPos])
		authenticated = (result == "YES")
	} else {
		return
	}

	return
}
