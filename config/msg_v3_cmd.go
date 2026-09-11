package config

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	"strings"

	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
)

const v3CMDBindingBatchSize = 1000

// ensureV3CMDDiscovery commits targeted discovery before a persistent one-shot SEND.
// Group discovery belongs to membership changes and the pre-cutover backfill.
func (c *Context) ensureV3CMDDiscovery(req *MsgSendReq) error {
	if !c.cfg.WuKongIM.V3ExplicitCMDBindings || req.Header.SyncOnce == 0 || req.Header.NoPersist != 0 {
		return nil
	}
	if len(req.Subscribers) > 0 {
		// Do not split or reorder: the exact recipient snapshot determines the SEND channel.
		if len(req.Subscribers) > v3CMDBindingBatchSize {
			return fmt.Errorf("v3 scoped CMD supports at most %d recipients per send", v3CMDBindingBatchSize)
		}
		return c.postV3CMDBinding("bind", map[string]interface{}{"subscribers": req.Subscribers})
	}
	if req.ChannelType != 1 {
		return nil
	}
	sender := strings.TrimSpace(req.FromUID)
	if sender == "" {
		sender = c.cfg.WuKongIM.V3SystemUID
	}
	peer := strings.TrimSpace(req.ChannelID)
	if sender == "" || peer == "" {
		return fmt.Errorf("v3 person CMD requires a sender and recipient")
	}
	if strings.Contains(peer, "@") {
		parts := strings.Split(peer, "@")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid v3 person CMD channel")
		}
		if parts[0] == sender {
			peer = parts[1]
		} else if parts[1] == sender {
			peer = parts[0]
		} else {
			return fmt.Errorf("v3 CMD sender does not belong to person channel")
		}
	}
	left, right := sender, peer
	lh, rh := crc32.ChecksumIEEE([]byte(left)), crc32.ChecksumIEEE([]byte(right))
	if lh < rh || (lh == rh && left < right) {
		left, right = right, left
	}
	// Bind both real peers so sender's other devices keep the existing CMD behavior.
	return c.updateV3SourceBindings([]string{sender, peer}, left+"@"+right, 1, false)
}

// updateV3SourceBindings batches membership lifecycle writes, never group message sends.
// A failed batch can have a committed prefix; propagate it so the caller retries.
func (c *Context) updateV3SourceBindings(uids []string, channelID string, channelType uint8, remove bool) error {
	if !c.cfg.WuKongIM.V3ExplicitCMDBindings {
		return nil
	}
	action := "bind"
	if remove {
		action = "unbind"
	}
	for start := 0; start < len(uids); start += v3CMDBindingBatchSize {
		end := start + v3CMDBindingBatchSize
		if end > len(uids) {
			end = len(uids)
		}
		if err := c.postV3CMDBinding(action, map[string]interface{}{"uids": uids[start:end], "channel_id": channelID, "channel_type": channelType}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Context) postV3CMDBinding(action string, request interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if len(body) > 256*1024 {
		return fmt.Errorf("v3 CMD binding request exceeds 256 KiB")
	}
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/message/cmd/"+action, body, nil)
	if err != nil {
		return err
	}
	return c.handlerIMError(resp)
}

// cmdSyncAPIURL keeps the latest process-local sync generation with its matching ack.
// An unavailable pinned process must fail; another process cannot acknowledge its records.
func (c *Context) cmdSyncAPIURL() string {
	if endpoint := strings.TrimSpace(c.cfg.WuKongIM.CMDSyncAPIURL); endpoint != "" {
		return strings.TrimRight(endpoint, "/")
	}
	return c.cfg.WuKongIM.APIURL
}
