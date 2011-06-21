package server_auth

import (
	"expvar"
	"http"
	"os"
	"time"
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

// An Authenticator takes a serverId and a user string and attempts to
// authenticate against a server. This interface allows for the use of a dummy
// authentication server for testing purposes.
type Authenticator interface {
	Authenticate(string, string) (bool, os.Error)
}

// DummyAuth is a no-op authentication server, always returning the value of
// 'Valid'.
type DummyAuth struct {
	Result bool
}

// ServerAuth represents authentication against a server, particularly the
// main minecraft server at http://www.minecraft.net/game/checkserver.jsp.
type ServerAuth struct {
	Url string
}

// Authenticate implements the Authenticator.Authenticate method
func (d *DummyAuth) Authenticate(serverId, user string) (authenticated bool, err os.Error) {
	return d.Result, nil
}

// Build a URL+query string based on a given server URL, serverId and user
// input
func (s *ServerAuth) BuildQuery(serverId, user string) (query string) {
	query = s.Url + "?" + http.EncodeQuery(
		map[string][]string{
			"serverId": {serverId},
			"user":     {user},
		},
	)

	return
}

// Authenticate implements the Authenticator.Authenticate method
func (s *ServerAuth) Authenticate(serverId, user string) (authenticated bool, err os.Error) {
	before := time.Nanoseconds()
	defer func() {
		after := time.Nanoseconds()
		expVarServerAuthTimeNs.Add(after - before)
		if authenticated {
			expVarServerAuthSuccessCount.Add(1)
		} else {
			expVarServerAuthFailCount.Add(1)
		}
	}()

	authenticated = false

	url := s.BuildQuery(serverId, user)

	response, _, err := http.Get(url)
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
			if err != nil && err != os.EOF {
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
