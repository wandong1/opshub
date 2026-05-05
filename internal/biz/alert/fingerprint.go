package alert

import "time"

// AlertFingerprint 告警指纹记录
type AlertFingerprint struct {
	ID              uint       `gorm:"primarykey" json:"id"`
	SubscriptionID  uint       `gorm:"index;not null" json:"subscriptionId"`
	Fingerprint     string     `gorm:"size:64;not null" json:"fingerprint"`
	RuleName        string     `gorm:"size:200;not null" json:"ruleName"`
	Severity        string     `gorm:"size:20;not null" json:"severity"`
	Labels          string     `gorm:"type:text" json:"labels"`
	FirstSeenAt     time.Time  `gorm:"not null" json:"firstSeenAt"`
	LastSeenAt      time.Time  `gorm:"not null" json:"lastSeenAt"`
	OccurrenceCount int        `gorm:"default:1" json:"occurrenceCount"`
	LastSentAt      *time.Time `json:"lastSentAt"`
}

func (AlertFingerprint) TableName() string {
	return "alert_fingerprints"
}
