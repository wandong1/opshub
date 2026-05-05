package alert

import "time"

// AlertSubscriptionLog 订阅执行日志
type AlertSubscriptionLog struct {
	ID             uint      `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time `gorm:"index" json:"createdAt"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscriptionId"`
	EventID        uint      `gorm:"index;not null" json:"eventId"`
	RuleID         uint      `gorm:"index;not null" json:"ruleId"`
	Matched        bool      `gorm:"not null;default:false" json:"matched"`
	// 匹配结果详情，JSON格式：{"timeRange":true,"severity":true,"dataSource":true,"labelMatchers":true}
	MatchResult string `gorm:"type:text" json:"matchResult"`
	// 降噪结果详情，JSON格式：{"deduplicated":false,"grouped":false,"inhibited":false}
	DenoiseResult string `gorm:"type:text" json:"denoiseResult"`
	// 推送结果详情，JSON格式：{"channels":[{"id":1,"name":"企微","status":"success"}],"users":[1,2,3]}
	NotifyResult string `gorm:"type:text" json:"notifyResult"`
}

func (AlertSubscriptionLog) TableName() string {
	return "alert_subscription_logs"
}
