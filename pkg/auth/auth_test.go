package auth

import (
	"bytes"
	"testing"

	"github.com/chuan/pkg/proto"
)

func TestAuthenticateSuccess(t *testing.T) {
	a := NewAuthenticator("my-secret")
	msg := a.BuildAuthMessage()

	buf := &bytes.Buffer{}
	msg.Encode(buf)

	decoded, _ := proto.Decode(buf)
	ok, err := a.Verify(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected auth success")
	}
}

func TestAuthenticateFailure(t *testing.T) {
	server := NewAuthenticator("correct-token")
	client := NewAuthenticator("wrong-token")

	msg := client.BuildAuthMessage()
	buf := &bytes.Buffer{}
	msg.Encode(buf)

	decoded, _ := proto.Decode(buf)
	ok, err := server.Verify(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected auth failure")
	}
}

func TestBuildAuthRespMessage(t *testing.T) {
	resp := BuildAuthRespMessage(true, "ok")
	if resp.Type != proto.MsgAuthResp {
		t.Fatal("wrong type")
	}

	resp2 := BuildAuthRespMessage(false, "bad token")
	if resp2.Type != proto.MsgAuthResp {
		t.Fatal("wrong type")
	}
}
