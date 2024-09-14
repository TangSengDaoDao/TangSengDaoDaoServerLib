package model

type Invite struct {
	InviteCode string
	Vercode    string
	Uid        string
}

type FriendResp struct {
	Remark    string // 好友备注
	ToUID     string // 好友uid
	IsDeleted int    // 是否删除
	IsAlone   int    // 是否为单项好友
	ShortNo   string // 短编号
}

type GroupMemberResp struct {
	GroupNo   string
	UID       string
	InviteUID string // 邀请人uid
	IsDeleted int    // 是否删除
	Role      int    // 成员角色 1. 创建者	 2.管理员
	CreatedAt string // 加入时间
}
