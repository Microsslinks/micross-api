package model

import (
	"errors"
	"sort"
	"strings"
)

// 经销商的自助台账：他自己的钱包还剩多少，以及他发出去的每个 Key 用了多少、花了多少。
//
// 「花了多少」是平台从他钱包里扣掉的额度，也就是他按拿货价向平台买下的那部分，
// 不是他卖给客户收了多少钱——平台不知道也不关心后者，台账里也不出现那个数。
//
// 归并口径是 logs.token_id：每一笔消费都挂在发出去的那个 Key 上。Key 行是软删除
// （tokens.deleted_at），删掉的 Key 连同它的历史一起留在台账里，这样每个 Key 的
// 花费加起来才对得上他的累计花费。

// AgentKeyUsage 台账里的一行：他发出去的一个 Key 的用量与花费。
//
// 金额与计数用 int64 而不是 int：这里是若干行相加的结果，SQLite/MySQL/PostgreSQL
// 对 SUM 返回的类型各不相同，收在宽类型里再交给前端格式化，不必在中间做饱和处理。
type AgentKeyUsage struct {
	TokenId      int    `json:"token_id"`
	Name         string `json:"name"`
	Status       int    `json:"status"`
	Removed      bool   `json:"removed"`
	UsedQuota    int64  `json:"used_quota"`
	RequestCount int64  `json:"request_count"`
	TotalTokens  int64  `json:"total_tokens"`
	LastUsedAt   int64  `json:"last_used_at"`
}

// AgentLedger 一位经销商的自助台账。
//
// Quota / UsedQuota / RequestCount 是他账户上的总数（与钱包页同一组数），
// Keys 是这些消费按 Key 拆开的样子。MarkupRatio 是平台给他定的那档毛利，
// 让他看得见自己的拿货价是按什么算出来的。
type AgentLedger struct {
	Quota        int              `json:"quota"`
	UsedQuota    int              `json:"used_quota"`
	RequestCount int              `json:"request_count"`
	MarkupRatio  string           `json:"markup_ratio"`
	Keys         []*AgentKeyUsage `json:"keys"`
}

// GetAgentLedger 取这位经销商的自助台账。
// 不是经销商时返回 ErrAgentProfileNotFound（该用户不是经销商）。
//
// 身份先看 users.subject_type，跟拿货价那条链同源（见 agent_wholesale.go）：
// 档案理论上跟身份一起写，真丢了也不该让经销商看不到自己的账，所以缺档案按缺省档继续。
func GetAgentLedger(userId int) (*AgentLedger, error) {
	if userId <= 0 {
		return nil, ErrAgentProfileNotFound
	}
	var user User
	if err := DB.First(&user, userId).Error; err != nil {
		return nil, err
	}
	if user.SubjectType != SubjectTypeAgent {
		return nil, ErrAgentProfileNotFound
	}
	profile, err := GetAgentProfileByUserId(userId)
	if err != nil && !errors.Is(err, ErrAgentProfileNotFound) {
		return nil, err
	}

	usageByToken, err := sumAgentKeyUsage(userId)
	if err != nil {
		return nil, err
	}
	keys, err := listAgentKeyUsage(userId, usageByToken)
	if err != nil {
		return nil, err
	}

	return &AgentLedger{
		Quota:        user.Quota,
		UsedQuota:    user.UsedQuota,
		RequestCount: user.RequestCount,
		MarkupRatio:  profile.EffectiveMarkupRatio(),
		Keys:         keys,
	}, nil
}

// agentKeyUsageRow 是聚合查询的一行：某个 Key 的消费合计。
type agentKeyUsageRow struct {
	TokenId      int
	UsedQuota    int64
	RequestCount int64
	TotalTokens  int64
	LastUsedAt   int64
}

// sumAgentKeyUsage 把这个用户所有 Key 的消费按 token_id 归并。
//
// token_id = 0 的行不进结果：它们不是从任何一个 Key 发出来的请求（例如平台侧
// 的内部调用），挂到某个 Key 头上是错的。type 只取消费，充值、管理这类流水
// 与「这个 Key 花了多少」无关。
func sumAgentKeyUsage(userId int) (map[int]*agentKeyUsageRow, error) {
	rows := make([]*agentKeyUsageRow, 0)
	err := LOG_DB.Table("logs").
		Select("token_id AS token_id, "+
			"COALESCE(sum(quota), 0) AS used_quota, "+
			"count(*) AS request_count, "+
			"COALESCE(sum(prompt_tokens), 0) + COALESCE(sum(completion_tokens), 0) AS total_tokens, "+
			"COALESCE(max(created_at), 0) AS last_used_at").
		Where("user_id = ?", userId).
		Where("type = ?", LogTypeConsume).
		Where("token_id > 0").
		Group("token_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	byToken := make(map[int]*agentKeyUsageRow, len(rows))
	for _, row := range rows {
		if row == nil {
			continue
		}
		byToken[row.TokenId] = row
	}
	return byToken, nil
}

// listAgentKeyUsage 列出这位经销商的每个 Key，并贴上它自己的消费合计。
//
// 用 Unscoped() 连软删除的 Key 一起列：客户那个 Key 用废了被删掉，那笔账还在，
// 从台账里抹掉会让「各 Key 之和」对不上他的累计花费。Removed 就是给这类行留的标记。
//
// 没有对应 Key 行的消费不进列表：Key 行不会被真正删除，出现对不上的行说明数据被
// 手工清理过，这种情况下不能凭空造一个名字出来，宁可少列一行。
// 排序按花费从多到少——经销商打开台账首先想知道哪个 Key 花得最多。
func listAgentKeyUsage(userId int, usageByToken map[int]*agentKeyUsageRow) ([]*AgentKeyUsage, error) {
	var tokens []*Token
	if err := DB.Unscoped().Where("user_id = ?", userId).Find(&tokens).Error; err != nil {
		return nil, err
	}

	keys := make([]*AgentKeyUsage, 0, len(tokens))
	for _, token := range tokens {
		usage := &AgentKeyUsage{
			TokenId: token.Id,
			Name:    strings.TrimSpace(token.Name),
			Status:  token.Status,
			Removed: token.DeletedAt.Valid,
		}
		if row, ok := usageByToken[token.Id]; ok {
			usage.UsedQuota = row.UsedQuota
			usage.RequestCount = row.RequestCount
			usage.TotalTokens = row.TotalTokens
			usage.LastUsedAt = row.LastUsedAt
		}
		keys = append(keys, usage)
	}
	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].UsedQuota != keys[j].UsedQuota {
			return keys[i].UsedQuota > keys[j].UsedQuota
		}
		return keys[i].TokenId < keys[j].TokenId
	})
	return keys, nil
}
