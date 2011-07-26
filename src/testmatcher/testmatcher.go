// gomock.Matcher implementations for use in unit tests that use mocks.
package testmatcher

import (
	"fmt"
	"strings"
)

type StringPrefix struct {
	Prefix string
}

func (m *StringPrefix) Matches(x interface{}) bool {
	s, ok := x.(string)
	if !ok {
		return false
	}

	return strings.HasPrefix(s, m.Prefix)
}

func (m *StringPrefix) String() string {
	return fmt.Sprintf("starts with %q", m.Prefix)
}
