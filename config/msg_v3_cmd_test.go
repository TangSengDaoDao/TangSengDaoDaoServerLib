package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

type cmdRequest struct {
	Path string
	Body map[string]interface{}
}

func cmdTestContext(t *testing.T, failPath string) (*Context, *[]cmdRequest) {
	t.Helper()
	requests := []cmdRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode: %v", err)
		}
		requests = append(requests, cmdRequest{r.URL.Path, body})
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == failPath {
			w.WriteHeader(503)
			fmt.Fprint(w, `{"status":503}`)
			return
		}
		fmt.Fprint(w, `{"status":200,"message_id":123,"message_seq":4}`)
	}))
	t.Cleanup(server.Close)
	cfg := New()
	cfg.WuKongIM.APIURL = server.URL
	cfg.WuKongIM.V3ExplicitCMDBindings = true
	return &Context{cfg: cfg}, &requests
}

func TestV3ScopedCMDBindsExactSnapshotBeforeSend(t *testing.T) {
	c, requests := cmdTestContext(t, "")
	req := &MsgSendReq{Header: MsgHeader{SyncOnce: 1}, Subscribers: []string{" u2 ", "u1", "u2"}, Payload: []byte("cmd")}
	if _, err := c.SendMessageWithResult(req); err != nil {
		t.Fatal(err)
	}
	got := *requests
	if len(got) != 2 || got[0].Path != "/message/cmd/bind" || got[1].Path != "/message/send" {
		t.Fatalf("wrong ordering: %v", got)
	}
	want := []interface{}{" u2 ", "u1", "u2"}
	if len(got[0].Body) != 1 || !reflect.DeepEqual(got[0].Body["subscribers"], want) || !reflect.DeepEqual(got[1].Body["subscribers"], want) {
		t.Fatalf("scope changed: %v", got)
	}
}

func TestV3CMDBindingFailurePreventsSend(t *testing.T) {
	c, requests := cmdTestContext(t, "/message/cmd/bind")
	if _, err := c.SendMessageWithResult(&MsgSendReq{Header: MsgHeader{SyncOnce: 1}, Subscribers: []string{"u1"}}); err == nil {
		t.Fatal("accepted failed binding")
	}
	if len(*requests) != 1 {
		t.Fatal("sent after binding failed")
	}
}

func TestV3PersonCMDUsesIMSystemIdentity(t *testing.T) {
	c, requests := cmdTestContext(t, "")
	c.cfg.WuKongIM.V3SystemUID = "native-system"
	c.cfg.Account.SystemUID = "business-system"
	if err := c.SendFriendApply(&MsgFriendApplyReq{ApplyUID: "u1", ToUID: "u2"}); err != nil {
		t.Fatal(err)
	}
	got := (*requests)[0].Body
	if got["channel_id"] != "native-system@u2" {
		t.Fatalf("wrong source %v", got)
	}
	if strings.Contains(got["channel_id"].(string), "business-system") {
		t.Fatal("wrong system identity")
	}
}

func TestV3BindingSkipsTransientOrdinaryAndLifecycleBoundGroups(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		header  MsgHeader
		kind    uint8
	}{
		{"v2", false, MsgHeader{SyncOnce: 1}, 1},
		{"ordinary", true, MsgHeader{}, 1},
		{"typing", true, MsgHeader{SyncOnce: 1, NoPersist: 1}, 1},
		{"group", true, MsgHeader{SyncOnce: 1}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, requests := cmdTestContext(t, "")
			c.cfg.WuKongIM.V3ExplicitCMDBindings = tc.enabled
			if _, err := c.SendMessageWithResult(&MsgSendReq{Header: tc.header, ChannelID: "target", ChannelType: tc.kind}); err != nil {
				t.Fatal(err)
			}
			if len(*requests) != 1 || (*requests)[0].Path != "/message/send" {
				t.Fatal("unexpected directory write")
			}
		})
	}
}

func TestV3MembershipLifecycleBatchesAfterMutation(t *testing.T) {
	for _, op := range []string{"create", "add", "remove"} {
		t.Run(op, func(t *testing.T) {
			c, requests := cmdTestContext(t, "")
			uids := make([]string, 1001)
			for i := range uids {
				uids[i] = fmt.Sprintf("u%d", i)
			}
			var err error
			path := "/channel"
			action := "bind"
			switch op {
			case "create":
				err = c.IMCreateOrUpdateChannel(&ChannelCreateReq{ChannelID: "g", ChannelType: 2, Subscribers: uids})
			case "add":
				path = "/channel/subscriber_add"
				err = c.IMAddSubscriber(&SubscriberAddReq{ChannelID: "g", ChannelType: 2, Subscribers: uids})
			case "remove":
				path = "/channel/subscriber_remove"
				action = "unbind"
				err = c.IMRemoveSubscriber(&SubscriberRemoveReq{ChannelID: "g", ChannelType: 2, Subscribers: uids})
			}
			if err != nil {
				t.Fatal(err)
			}
			got := *requests
			if len(got) != 3 || got[0].Path != path || got[1].Path != "/message/cmd/"+action || len(got[1].Body["uids"].([]interface{})) != 1000 || len(got[2].Body["uids"].([]interface{})) != 1 {
				t.Fatalf("wrong batches: %d", len(got))
			}
		})
	}
}

func TestV3MembershipFailureAndResetFailClosed(t *testing.T) {
	c, requests := cmdTestContext(t, "/channel/subscriber_add")
	req := &SubscriberAddReq{ChannelID: "g", ChannelType: 2, Subscribers: []string{"u1"}}
	if err := c.IMAddSubscriber(req); err == nil || len(*requests) != 1 {
		t.Fatal("failed mutation bound recipients")
	}
	req.Reset = 1
	if err := c.IMAddSubscriber(req); err == nil || len(*requests) != 1 {
		t.Fatal("reset mutated before explicit reconciliation")
	}
	c2, r2 := cmdTestContext(t, "/message/cmd/bind")
	req.Reset = 0
	if err := c2.IMAddSubscriber(req); err == nil || len(*r2) != 2 {
		t.Fatal("binding failure hidden from lifecycle caller")
	}
}

func TestV3ScopedCMDCannotSilentlySplitScope(t *testing.T) {
	c, requests := cmdTestContext(t, "")
	if err := c.SendMessage(&MsgSendReq{Header: MsgHeader{SyncOnce: 1}, Subscribers: make([]string, 1001)}); err == nil || len(*requests) != 0 {
		t.Fatal("oversized scope reached IM")
	}
}

func TestV3CMDSyncAndAckUseSamePinnedProcess(t *testing.T) {
	c, requests := cmdTestContext(t, "")
	paths := []string{}
	pinned := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		io.Copy(io.Discard, r.Body)
		if r.URL.Path == "/message/sync" {
			fmt.Fprint(w, `[]`)
		} else {
			fmt.Fprint(w, `{"status":200}`)
		}
	}))
	defer pinned.Close()
	c.cfg.WuKongIM.CMDSyncAPIURL = pinned.URL + "/"
	if _, err := c.IMSyncMessage(&MsgSyncReq{}); err != nil {
		t.Fatal(err)
	}
	if err := c.IMSyncMessageAck(&SyncackReq{}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{"/message/sync", "/message/syncack"}) || len(*requests) != 0 {
		t.Fatal("CMD generation crossed processes")
	}
}
