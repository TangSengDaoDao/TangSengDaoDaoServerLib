package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSetting(t *testing.T) {
	setting := SettingFromUint8(160)
	fmt.Println(setting.Signal)
}

func TestMsgSendReqClientMsgNoJSON(t *testing.T) {
	req := MsgSendReq{
		ClientMsgNo: "hall_reply_001",
		FromUID:     "robot",
		ChannelID:   "group",
		ChannelType: 2,
		Payload:     []byte(`{"type":1,"content":"ok"}`),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal MsgSendReq: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal MsgSendReq JSON: %v", err)
	}
	if got := payload["client_msg_no"]; got != req.ClientMsgNo {
		t.Fatalf("client_msg_no = %v, want %q", got, req.ClientMsgNo)
	}
}

func TestMsgSendReqClientMsgNoOmittedWhenEmpty(t *testing.T) {
	data, err := json.Marshal(MsgSendReq{})
	if err != nil {
		t.Fatalf("marshal MsgSendReq: %v", err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal MsgSendReq JSON: %v", err)
	}
	if _, exists := payload["client_msg_no"]; exists {
		t.Fatalf("client_msg_no should be omitted when empty: %s", data)
	}
}

func TestSendMessageWithResultForwardsClientMsgNo(t *testing.T) {
	const clientMsgNo = "hall_reply_retry_stable"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer request.Body.Close()
		var body map[string]interface{}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		if body["client_msg_no"] != clientMsgNo {
			t.Errorf("client_msg_no = %v, want %q", body["client_msg_no"], clientMsgNo)
		}
		writer.Header().Set("content-type", "application/json")
		_, _ = writer.Write([]byte(`{"data":{"message_id":101,"message_seq":7,"client_msg_no":"hall_reply_retry_stable"}}`))
	}))
	defer server.Close()

	cfg := New()
	cfg.WuKongIM.APIURL = server.URL
	result, err := (&Context{cfg: cfg}).SendMessageWithResult(&MsgSendReq{
		ClientMsgNo: clientMsgNo, FromUID: "robot", ChannelID: "group", ChannelType: 2,
		Payload: []byte(`{"type":1,"content":"ok"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.MessageID != 101 || result.MessageSeq != 7 || result.ClientMsgNo != clientMsgNo {
		t.Fatalf("unexpected send result: %+v", result)
	}
}
