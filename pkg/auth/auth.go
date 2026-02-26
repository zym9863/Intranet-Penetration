package auth

import (
	"encoding/json"

	"github.com/chuan/pkg/proto"
)

type authPayload struct {
	Token string `json:"token"`
}

type authRespPayload struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type Authenticator struct {
	token string
}

func NewAuthenticator(token string) *Authenticator {
	return &Authenticator{token: token}
}

func (a *Authenticator) BuildAuthMessage() *proto.Message {
	payload, _ := json.Marshal(authPayload{Token: a.token})
	return &proto.Message{
		Version: proto.ProtoVersion,
		Type:    proto.MsgAuth,
		Payload: payload,
	}
}

func (a *Authenticator) Verify(msg *proto.Message) (bool, error) {
	if msg.Type != proto.MsgAuth {
		return false, nil
	}
	var p authPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return false, err
	}
	return p.Token == a.token, nil
}

func BuildAuthRespMessage(ok bool, message string) *proto.Message {
	payload, _ := json.Marshal(authRespPayload{OK: ok, Message: message})
	return &proto.Message{
		Version: proto.ProtoVersion,
		Type:    proto.MsgAuthResp,
		Payload: payload,
	}
}

func ParseAuthResp(msg *proto.Message) (bool, string, error) {
	var p authRespPayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return false, "", err
	}
	return p.OK, p.Message, nil
}
