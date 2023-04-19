package testutil

import (
	"fmt"
	"strings"

	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func ProtoEqual(expected proto.Message) types.GomegaMatcher {
	return &ProtoMatcher{
		Expected: expected,
	}
}

type ProtoMatcher struct {
	Expected proto.Message
}

func (matcher *ProtoMatcher) Match(actual any) (success bool, err error) {
	if actual == nil && matcher.Expected == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <nil>.\nBe explicit and use BeNil() instead.  This is to avoid mistakes where both sides of an assertion are erroneously uninitialized.")
	}
	if _, ok := actual.(proto.Message); !ok {
		return false, fmt.Errorf("ProtoMatcher expects a proto.Message. Got:\n%s", format.Object(actual, 1))
	}
	return proto.Equal(actual.(proto.Message), matcher.Expected), nil
}

func (matcher *ProtoMatcher) FailureMessage(actual any) (message string) {
	return format.Message(actual, "to equal", matcher.Expected)
}

func (matcher *ProtoMatcher) NegatedFailureMessage(actual any) (message string) {
	return format.Message(actual, "not to equal", matcher.Expected)
}

type StatusCodeMatcher struct {
	Expected any
	matchMsg types.GomegaMatcher
}

func MatchStatusCode(expected any, matchMessage ...types.GomegaMatcher) types.GomegaMatcher {
	m := &StatusCodeMatcher{
		Expected: expected,
	}
	if len(matchMessage) > 0 {
		m.matchMsg = matchMessage[0]
	}
	return m
}

func (m *StatusCodeMatcher) code(value any) codes.Code {
	if value == nil {
		return codes.OK
	}
	switch value := value.(type) {
	case error:
		return status.Code(value)
	case *status.Status:
		return value.Code()
	case codes.Code:
		return value
	case uint32:
		return codes.Code(value)
	default:
		panic(fmt.Sprintf("MatchStatus expects a grpc status, error, or codes.Code. Got:\n%s", format.Object(value, 1)))
	}
}

func (m *StatusCodeMatcher) Match(actual any) (success bool, err error) {
	if actual == nil && m.Expected == nil {
		return false, fmt.Errorf("Refusing to compare <nil> to <nil>.\nBe explicit and use BeNil() instead.  This is to avoid mistakes where both sides of an assertion are erroneously uninitialized.")
	}

	return m.code(actual) == m.code(m.Expected), nil
}

func (matcher *StatusCodeMatcher) FailureMessage(actual any) (message string) {
	actualStatusCode := matcher.code(actual)
	expectedStatusCode := matcher.code(matcher.Expected)

	actualMsg := fmt.Sprintf("%s | %s(%d)", format.Object(actual, 1), actualStatusCode.String(), actualStatusCode)
	expectedMsg := fmt.Sprintf("%s | %s(%d)", format.Object(matcher.Expected, 1), expectedStatusCode.String(), expectedStatusCode)

	return fmt.Sprintf("Expected\n%s\nto match the status code of\n%s", actualMsg, expectedMsg)
}

func (matcher *StatusCodeMatcher) NegatedFailureMessage(actual any) (message string) {
	msg := matcher.FailureMessage(actual)
	return strings.Replace(msg, "to match", "not to match", 1)
}
