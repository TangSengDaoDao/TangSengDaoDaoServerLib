package model

type FriendResp struct {
	Remark    string //好友备注
	ToUID     string // 好友uid
	IsDeleted int    // 是否删除
	IsAlone   int    // 是否为单项好友
	ShortNo   string //短编号
}
