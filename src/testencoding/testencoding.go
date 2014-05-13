// The testencoding package is used in testing serialization and
// deserialization. It is particularly useful when some sub-sequences of bytes
// can acceptably be written in any order (e.g if they are generated from
// representations that do not provide order guaruntees like maps).
package testencoding

import (
	"bytes"
	"fmt"
	"strings"
)

type MatchError struct {
	FailedMatcher IBytesMatcher
	NextBytes     []byte
}

func (err *MatchError) Error() string {
	return fmt.Sprintf("failed to match %v on bytes: %x", err.FailedMatcher, err.NextBytes)
}

type TrailingBytesError struct {
	FailedMatcher IBytesMatcher
	TrailingBytes []byte
}

func (err *TrailingBytesError) Error() string {
	return fmt.Sprintf("trailing bytes when matching %v: %x", err.FailedMatcher, err.TrailingBytes)
}

// IBytesMatcher is the interface that tests if a sequence of bytes matches
// expectations.
type IBytesMatcher interface {
	// Match tests if the given sequence of bytes matches. It returns
	// matches=true if it matches correctly, and returns the number of bytes
	// matched as n.
	Match(b []byte) (n int, err error)

	// Write writes a sequence of bytes to the buffer that is an acceptable match
	// for the matcher..
	Write(writer *bytes.Buffer)

	String() string
}

// BytesLiteral matches a literal sequence of bytes.
type BytesLiteral struct {
	Bytes []byte
}

func LiteralString(s string) *BytesLiteral {
	return &BytesLiteral{[]byte(s)}
}

func (bm *BytesLiteral) Match(b []byte) (n int, err error) {
	if len(b) < len(bm.Bytes) {
		return 0, &MatchError{bm, b}
	}

	for i, v := range bm.Bytes {
		if v != b[i] {
			return 0, &MatchError{bm, b}
		}
	}

	return len(bm.Bytes), nil
}

func (bm *BytesLiteral) Write(writer *bytes.Buffer) {
	writer.Write(bm.Bytes)
}

func (bm *BytesLiteral) String() string {
	return fmt.Sprintf("&BytesLiteral{%x}", bm.Bytes)
}

// BytesMatcherInOrder matches a sequence of BytesMatchers that must match in
// the order given.
type BytesMatcherInOrder struct {
	Matchers []IBytesMatcher
}

func InOrder(matchers ...IBytesMatcher) *BytesMatcherInOrder {
	return &BytesMatcherInOrder{matchers}
}

func (bm *BytesMatcherInOrder) Match(b []byte) (n int, err error) {
	var consumed int

	for _, matcher := range bm.Matchers {
		remainder := b[n:]

		if consumed, err = matcher.Match(remainder); err != nil {
			return 0, err
		}

		n += consumed
	}

	return n, nil
}

func (bm *BytesMatcherInOrder) Write(writer *bytes.Buffer) {
	for _, matcher := range bm.Matchers {
		matcher.Write(writer)
	}
}

func (bm *BytesMatcherInOrder) String() string {
	parts := make([]string, len(bm.Matchers))
	for i, matcher := range bm.Matchers {
		parts[i] = matcher.String()
	}
	s := strings.Join(parts, ", ")
	return fmt.Sprintf("&BytesMatcherInOrder{%s}", s)
}

// BytesMatcherAnyOrder matches a set of BytesMatchers, but they do not have to
// match in any particular order.
type BytesMatcherAnyOrder struct {
	Matchers []IBytesMatcher
}

func AnyOrder(matchers ...IBytesMatcher) *BytesMatcherAnyOrder {
	return &BytesMatcherAnyOrder{matchers}
}

func (bm *BytesMatcherAnyOrder) Match(b []byte) (n int, err error) {
	toMatch := make([]IBytesMatcher, len(bm.Matchers))
	copy(toMatch, bm.Matchers)
	var consumed int

	for {
		if len(toMatch) == 0 {
			break
		}

		remainder := b[n:]

		foundMatch := false
		for i, matcher := range toMatch {
			if consumed, err = matcher.Match(remainder); err == nil {
				foundMatch = true
				n += consumed

				// Remove matcher from toMatch.
				if len(toMatch) > 1 {
					toMatch[i] = toMatch[len(toMatch)-1]
				}
				toMatch = toMatch[:len(toMatch)-1]

				break
			}
		}

		if !foundMatch {
			return 0, &MatchError{&BytesMatcherAnyOrder{toMatch}, remainder}
		}
	}

	return n, nil
}

func (bm *BytesMatcherAnyOrder) Write(writer *bytes.Buffer) {
	for _, matcher := range bm.Matchers {
		matcher.Write(writer)
	}
}

func (bm *BytesMatcherAnyOrder) String() string {
	parts := make([]string, len(bm.Matchers))
	for i, matcher := range bm.Matchers {
		parts[i] = matcher.String()
	}
	s := strings.Join(parts, ", ")
	return fmt.Sprintf("&BytesMatcherAnyOrder{%s}", s)
}

func Matches(matcher IBytesMatcher, b []byte) error {
	n, err := matcher.Match(b)
	if err != nil {
		return err
	}
	if n != len(b) {
		return &TrailingBytesError{matcher, b}
	}
	return nil
}
