package alert

import (
	"time"

	"gorm.io/gorm"
)

// AlertSubscription 告警订阅任务
type AlertSubscription struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Name         string         `gorm:"size:100;not null" json:"name"`
	AssetGroupID uint           `gorm:"index" json:"assetGroupId"`
	Description  string         `gorm:"size:500" json:"description"`
	Enabled      bool           `gorm:"default:true" json:"enabled"`
}

func (AlertSubscription) TableName() string {
	return "alert_subscriptions"
}

// AlertSubscriptionRule 订阅与规则的关联（含细粒度生效时间、级别过滤、通道、用户）
type AlertSubscriptionRule struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscriptionId"`
	RuleID         uint      `gorm:"index;not null" json:"ruleId"`
	// 多组生效时间段，JSON格式：[{"weekdays":[1,2,3,4,5],"start":"08:00","end":"18:00"}]
	// 空值表示全天全周生效
	TimeRanges string `gorm:"type:text" json:"timeRanges"`
	// 级别过滤，JSON格式：["critical","major"]，空=全部级别
	Severities string `gorm:"type:text" json:"severities"`
	// 该时间段关联的通知通道 ID 列表，JSON格式：[1,2,3]，空=使用订阅全局通道
	ChannelIDs string `gorm:"type:text" json:"channelIds"`
	// 该时间段关联的接收用户 ID 列表，JSON格式：[1,2,3]，0=@all
	UserIDs string `gorm:"type:text" json:"userIds"`
}

func (AlertSubscriptionRule) TableName() string {
	return "alert_subscription_rules"
}

// AlertSubscriptionChannel 订阅与通道的关联
type AlertSubscriptionChannel struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscriptionId"`
	ChannelID      uint      `gorm:"index;not null" json:"channelId"`
}

func (AlertSubscriptionChannel) TableName() string {
	return "alert_subscription_channels"
}

// AlertSubscriptionUser 订阅与用户的关联
type AlertSubscriptionUser struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `json:"createdAt"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscriptionId"`
	UserID         uint      `gorm:"index;not null" json:"userId"` // 复用 sys_user.id
}

func (AlertSubscriptionUser) TableName() string {
	return "alert_subscription_users"
}
