package config

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendMessageWithResultPreservesDirectAndWrappedIdentity(t *testing.T) {
	const fields = `"message_id":2098329132244295680,"message_seq":7,"client_msg_no":"fixture-v3"`
	for _, tc := range []struct{ name, body string }{
		{"direct v3", "{" + fields + ",\"reason\":1}"},
		{"wrapped legacy", "{\"data\":{" + fields + "}}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/message/send" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			cfg := New()
			cfg.WuKongIM.APIURL = server.URL
			c := &Context{cfg: cfg}
			got, err := c.SendMessageWithResult(&MsgSendReq{FromUID: "sender", ChannelID: "receiver", ChannelType: 1, Payload: []byte(`{"type":1,"content":"test"}`)})
			if err != nil {
				t.Fatal(err)
			}
			if got.MessageID != 2098329132244295680 || got.MessageSeq != 7 || got.ClientMsgNo != "fixture-v3" {
				t.Fatalf("message identity lost: %+v", got)
			}
		})
	}
}
