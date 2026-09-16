package model

// CommissionRecord 邀请佣金流水（task-10 · P4 佣金核心）。
//
// 触发点：被邀请人每次成功调用 API 之后（在 service/text_quota.go:PostTextConsumeQuota 异步 hook）。
// 计算口径：amount = gross × rate；gross = 被邀请人实付（口径 B② 平台实收）；
//          rate = 运营全局返佣率（口径 A①，存于 operation_setting.CommissionRate）。
// 不赔本校验（口径 C①）：当 amount > 平台毛利 × 0.8 时降级到 cap 并写 breach=true + 一条 audit 日志。
//
// 关键约束：
//   - (consume_log_id, inviter_id) 唯一：防重试返佣（README §七第 4 条）。
//   - Rate 是 decimal(6,6) 字符串：与项目折扣方案同口径，避免 float 精度漂移。
//   - 不动 user.Quota；只动 user.AffCommissionBalance（独立钱包，不可提现，task-11 实现）。

// CommissionRecord 邀请佣金流水（与 Log 1:1 对应——一笔记费对应一条佣金）。
//
// - Gross: 被邀请人当次实际计费金额（口径 B②：summary.Quota，1 quota = 1 / common.QuotaPerUnit 美元）
// - Rate:  运营全局返佣率快照（口径 A①），落档时快照写死，方便事后审计
// - Amount: 实际落账金额（gross × rate 后再经 AssertNoLoss 限幅）
// - Margin: 当次平台毛利快照（审计用）
// - Breach: 是否被限幅（commission > margin × 0.8 时 true），与 v0.19.6 "指定渠道亏本留痕" 同模式
type CommissionRecord struct {
	Id           int64  `json:"id" gorm:"primaryKey;autoIncrement"`
	InviterId    int    `json:"inviter_id" gorm:"index;index:idx_invitee_consume,priority:2"`
	InviteeId    int    `json:"invitee_id" gorm:"index;index:idx_invitee_consume,priority:1"`
	ConsumeLogId int64  `json:"consume_log_id" gorm:"uniqueIndex:uk_consume_inviter,priority:1"`
	Gross        int    `json:"gross" gorm:"default:0"`
	// Rate 字段类型选择 decimal(6,6) 字符串——A 类债提醒：
	// 不要写 `int` 或 `type:int`（项目早期 type:int 标签在迁移场景出过 bug，沿用
	// DiscountPlan.CommissionRatio 的 decimal 字符串做法，Float→String 转换无精度漂移）。
	Rate      string `json:"rate" gorm:"type:decimal(6,6);default:0"`
	Amount    int    `json:"amount" gorm:"default:0"`
	Margin    int    `json:"margin" gorm:"default:0"`
	Breach    bool   `json:"breach" gorm:"default:false;index"`
	Currency  string `json:"currency" gorm:"type:varchar(8);default:'USD'"`
	SettledAt int64  `json:"settled_at" gorm:"bigint;index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;index"`
}

func (cr *CommissionRecord) TableName() string { return "commission_records" }