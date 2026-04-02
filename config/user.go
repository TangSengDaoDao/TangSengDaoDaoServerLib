package config

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/network"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/util"
)

// AddSystemUidsResp 添加系统账号返回数据
type AddSystemUidsResp struct {
	Status int `json:"status"` // 状态
}

// AddSystemUids 添加系统账号
func (c *Context) AddSystemUids(uids []string) (*AddSystemUidsResp, error) {
	resp, err := network.Post(c.cfg.WuKongIM.APIURL+"/user/systemuids_add", []byte(util.ToJson(map[string]interface{}{
		"uids": uids,
	})), nil)
	if err != nil {
		return nil, err
	}
	err = c.handlerIMError(resp)
	if err != nil {
		return nil, err
	}
	var result *AddSystemUidsResp
	if err := util.ReadJsonByByte([]byte(resp.Body), &result); err != nil {
		return nil, err
	}
	return result, nil
}
