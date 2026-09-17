// account_audit 是 task-17 §17.3 的对账 CLI：扫账本对照用户余额，找出漏记账的事件。
//
// 用法：
//
//	go run ./cmd/account_audit                    # 全量扫所有用户
//	go run ./cmd/account_audit -user-id=42       # 只扫指定用户
//	go run ./cmd/account_audit -json             # JSON 输出便于 CI 接入
//
// 退出码：
//
//	0 = 所有用户对账通过（差额 0）
//	1 = 发现 N 个用户对账不通过（漏记账 / 余额漂移）
//	2 = 调用方错误（如 SQL_DSN 没设）
//
// 输出（文本模式）：每个差额用户一行："user_id=X delta=Y out_status=OK"。
//
// 实现：项目自身用 ledger 之前已经积累的余额变化（充值、佣金）并未写 ledger
// （task-17 §17.3 落地后才有 ledger），所以**全新部署**一开始对账脚本会全 FAIL——
// 这是预期的。落地后的所有变化才会被 ledger 覆盖。
//
// 推荐用法：CI 每天跑一次（CI 环境用 fresh DB，ledger 与余额都从 0 开始）；
// 生产环境**只把脚本接监控**（差额 > 0 → 报警），由运营决定是否人工补一笔反向 ledger。
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

type mismatch struct {
	UserID         int   `json:"user_id"`
	Quota          int64 `json:"quota"`
	AffCommBalance int64 `json:"aff_commission_balance"`
	LedgerSum      int64 `json:"ledger_sum"`
	Delta          int64 `json:"delta"` // (Quota + AffCommBalance) - LedgerSum
}

func main() {
	var (
		userIDFlag = flag.Int("user-id", 0, "只扫指定用户；0 = 全量")
		jsonFlag   = flag.Bool("json", false, "输出 JSON 格式（便于 CI 接入）")
	)
	flag.Parse()

	common.InitEnv()
	if err := model.InitDB(); err != nil {
		fmt.Fprintf(os.Stderr, "init db failed: %v\n", err)
		os.Exit(2)
	}

	var mismatches []mismatch

	if *userIDFlag > 0 {
		m, err := auditOne(int64(*userIDFlag))
		if err != nil {
			fmt.Fprintf(os.Stderr, "audit user %d failed: %v\n", *userIDFlag, err)
			os.Exit(2)
		}
		if m != nil {
			mismatches = append(mismatches, *m)
		}
	} else {
		// 全量扫：分批 200 用户，避免一次性 SELECT IN (...) 太大。
		const batch = 200
		var lastID int64
		for {
			var users []model.User
			if err := model.DB.Select("id, quota, aff_commission_balance").
				Where("id > ?", lastID).
				Order("id ASC").
				Limit(batch).
				Find(&users).Error; err != nil {
				fmt.Fprintf(os.Stderr, "scan users failed: %v\n", err)
				os.Exit(2)
			}
			if len(users) == 0 {
				break
			}
			for _, u := range users {
				lastID = int64(u.Id)
				m, err := auditUser(u)
				if err != nil {
					logger.LogError(context.Background(), fmt.Sprintf("audit user %d failed: %s", u.Id, err.Error()))
					continue
				}
				if m != nil {
					mismatches = append(mismatches, *m)
				}
			}
		}
	}

	if *jsonFlag {
		out := struct {
			Mismatches []mismatch `json:"mismatches"`
			Total      int        `json:"total"`
		}{Mismatches: mismatches, Total: len(mismatches)}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(out)
	} else {
		fmt.Printf("=== 对账扫描完成 ===\n")
		fmt.Printf("对账不通过用户数：%d\n", len(mismatches))
		for _, m := range mismatches {
			fmt.Printf("user_id=%d quota=%d aff_comm=%d ledger_sum=%d delta=%d\n",
				m.UserID, m.Quota, m.AffCommBalance, m.LedgerSum, m.Delta)
		}
	}

	if len(mismatches) > 0 {
		os.Exit(1)
	}
}

// auditOne 扫单个用户的对账情况。
func auditOne(userID int64) (*mismatch, error) {
	var user model.User
	if err := model.DB.Select("id, quota, aff_commission_balance").
		First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return auditUser(user)
}

// auditUser 算一个用户的 ledger 累计与余额对比。
// 返回 nil = 对账通过；返回 *mismatch = 有差额。
func auditUser(user model.User) (*mismatch, error) {
	sum, err := model.SumLedgerAmount("user", user.Id)
	if err != nil {
		return nil, err
	}
	actual := int64(user.Quota) + int64(user.AffCommissionBalance)
	delta := actual - sum
	// 0 视为通过；非 0 视为漏记账 / 漂移。
	if delta == 0 {
		return nil, nil
	}
	return &mismatch{
		UserID:         user.Id,
		Quota:          int64(user.Quota),
		AffCommBalance: int64(user.AffCommissionBalance),
		LedgerSum:      sum,
		Delta:          delta,
	}, nil
}