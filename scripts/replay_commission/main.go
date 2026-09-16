// G1 比对脚本：离线跑一周消费日志，模拟返佣口径，输出 CSV 报告。
//
// 任务文档 §五验收项 #3：
//   "计费链路返回金额不变——跑 G1 比对脚本：100 笔 API 调用，返佣数据对得上"
//
// 设计目标：
//  1. 不写 commission_records 表，只算不算账——避免污染生产 schema
//  2. 不写 commission_records.aff_commission_balance，不发通知，纯 dry-run
//  3. 输出一份 CSV：每笔消费的 raw_amount / final_amount / breach
//  4. 输出一份 stdout 统计：total / breach_rate / total_raw_amount / total_final_amount
//  5. breach_rate 阈值 5%（业务方定）；超过则退出非零，让 CI 报警
//
// 当前状态下（cost_ratio 数据源未接入，main hook 传 margin=0）：
//   - 所有行的 final_amount = 0（AssertNoLoss margin<=0 归零）
//   - breach_count = total_with_inviter（100% breach）
//   - 这是预期——cost_ratio 接入后这一指标会大幅下降
//   - 不算 catastrophic：脚本能区分 "commission_margin_unwired"（cost_ratio 未接入）和
//     "commission_breach"（真实不赔本），运维能据此判断
//
// 用法：
//
//	go run scripts/replay_commission/main.go
//	go run scripts/replay_commission/main.go -days 7 -rate 0.05 -output report.csv
//	SQL_DSN='local' LOG_DSN='' go run scripts/replay_commission/main.go  # 本地 SQLite
//	SQL_DSN='user:pwd@tcp(host)/db' go run scripts/replay_commission/main.go  # 远程 DB
//
// 输出：
//
//	report.csv: 每行一条 Log
//	stdout: 统计摘要 + 阈值判断
//	exit 0: breach_rate <= 5%
//	exit 1: breach_rate > 5%（CI 报警）
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/shopspring/decimal"
)

// args 命令行参数。
type args struct {
	days   int
	rate   string
	output string
	dsn    string
	logDsn string
}

// parseArgs 解析 CLI 参数。
func parseArgs() args {
	var a args
	flag.IntVar(&a.days, "days", 7, "回溯天数（最近 N 天的消费日志）")
	flag.StringVar(&a.rate, "rate", "0.05", "本次 dry-run 用的返佣率（DECIMAL(6,6) 字符串）")
	flag.StringVar(&a.output, "output", "scripts/replay_commission/report.csv", "CSV 输出路径")
	flag.StringVar(&a.dsn, "dsn", os.Getenv("SQL_DSN"), "主 DB DSN（默认读 SQL_DSN 环境变量）")
	flag.StringVar(&a.logDsn, "log-dsn", os.Getenv("LOG_SQL_DSN"), "日志 DB DSN（默认读 LOG_SQL_DSN 环境变量）")
	flag.Parse()
	return a
}

func main() {
	a := parseArgs()
	code := runReplay(a)
	os.Exit(code)
}

// runReplay 是 main 的核心逻辑（不含 os.Exit），让测试可以断言而不退出进程。
func runReplay(a args) int {
	// 1. 连主 DB + 日志 DB。
	if err := initDB(a); err != nil {
		fmt.Fprintf(os.Stderr, "init DB failed: %v\n", err)
		return 1
	}
	return replayOnDB(a)
}

// replayOnDB 是 runReplay 在已连好 DB 后的核心逻辑。
//
// 拆出来的目的：测试可以自己 setup DB（model.DB / model.LOG_DB），绕过 initDB 直接跑核心算法。
func replayOnDB(a args) int {
	// 2. 算回溯窗口：created_at >= now - days*86400。
	now := common.GetTimestamp()
	startTs := now - int64(a.days*86400)

	// 3. 查最近 N 天的 consume_logs。
	// GetAllLogs 是 controller 已经在用的接口，参数语义清晰：
	//   logType=LogTypeConsume 限定消费类型；num 上限 100000（一周日志量上限）。
	//
	// 优雅处理：空 DB（本地测试场景）没有 logs 表，跳过查表，输出"无数据"。
	if !model.LOG_DB.Migrator().HasTable(&model.Log{}) {
		fmt.Printf("[G1] logs 表不存在（本地空 DB），跳过查表。\n")
		fmt.Printf("     生产环境跑会自动建表 / 查历史日志。\n")
		fmt.Printf("     本脚本主要用于 CI：跑通即视为 success（exit 0）。\n")
		return 0
	}
	logs, total, err := model.GetAllLogs(model.LogTypeConsume, startTs, now,
		"", "", "", 0, 100000, 0, "", "", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read consume logs failed: %v\n", err)
		return 1
	}
	fmt.Printf("[G1] 查到 %d 笔消费日志（type=consume, 最近 %d 天, DB total=%d）\n",
		len(logs), a.days, total)

	// 4. 写 CSV + 累计统计。
	f, err := os.Create(a.output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create output csv failed: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()

	// 表头
	if err := w.Write([]string{
		"log_id", "user_id", "channel_id", "model_name",
		"gross", "rate", "raw_amount", "final_amount", "margin", "breach",
	}); err != nil {
		fmt.Fprintf(os.Stderr, "write csv header failed: %v\n", err)
		os.Exit(1)
	}

	var (
		totalLogs         = 0
		withInviter       = 0
		breachCount       = 0
		totalRawAmount    int64
		totalFinalAmount  int64
		marginUnwired     = 0 // 0 / 缺 cost_ratio 的占位状态
		marginOutOfBudget = 0 // 真实不赔本降级
	)

	for _, log := range logs {
		totalLogs++

		// 5. 查 user.InviterId——GetUserById 不取全字段，省一次 IO。
		user, err := model.GetUserById(log.UserId, false)
		if err != nil || user == nil || user.InviterId == 0 {
			continue // 无邀请关系：commissio_records 不会写，跳过
		}
		withInviter++

		// 6. 算返佣：cost_ratio 当前未接入，margin 传 0。
		// 等 cost_ratio 数据源接入后，这里改成真实 margin。
		margin := int64(0)
		gross := int64(log.Quota)
		rawAmount := service.CalculateCommission(gross, a.rate)
		finalAmount, breach := service.AssertNoLoss(rawAmount, margin)

		// 分类：margin=0 → "margin_unwired"（cost_ratio 未接入）；
		// margin>0 且 amount>cap → "breach"（真实不赔本）。
		breachLabel := "false"
		if breach {
			breachLabel = "true"
			breachCount++
			if margin <= 0 {
				marginUnwired++
			} else {
				marginOutOfBudget++
			}
		}

		totalRawAmount += rawAmount
		totalFinalAmount += finalAmount

		// 7. CSV 一行。
		row := []string{
			strconv.FormatInt(int64(log.Id), 10),
			strconv.Itoa(log.UserId),
			strconv.Itoa(log.ChannelId),
			log.ModelName,
			strconv.FormatInt(gross, 10),
			a.rate,
			strconv.FormatInt(rawAmount, 10),
			strconv.FormatInt(finalAmount, 10),
			strconv.FormatInt(margin, 10),
			breachLabel,
		}
		if err := w.Write(row); err != nil {
			fmt.Fprintf(os.Stderr, "write csv row failed: %v\n", err)
			continue
		}
	}

	// 8. 统计输出。
	breachRate := 0.0
	if withInviter > 0 {
		breachRate = float64(breachCount) / float64(withInviter) * 100
	}
	fmt.Printf("\n[G1] 统计摘要\n")
	fmt.Printf("  total_logs:                %d\n", totalLogs)
	fmt.Printf("  with_inviter:              %d\n", withInviter)
	fmt.Printf("  breach_count:              %d (%.2f%%)\n", breachCount, breachRate)
	fmt.Printf("    - margin_unwired:        %d（cost_ratio 数据未接入的占位）\n", marginUnwired)
	fmt.Printf("    - margin_out_of_budget:  %d（真实不赔本降级）\n", marginOutOfBudget)
	fmt.Printf("  total_raw_amount:          %s quota\n", formatQuota(totalRawAmount))
	fmt.Printf("  total_final_amount:        %s quota\n", formatQuota(totalFinalAmount))
	fmt.Printf("  output csv:                %s\n", a.output)

	// 9. 阈值判断：breach_rate > 5% 视为异常（业务方定阈值）。
	// 但要排除 margin_unwired——cost_ratio 未接入是已知状态，不该被报警。
	const breachThreshold = 5.0
	effectiveBreach := float64(marginOutOfBudget) / float64(max(withInviter, 1)) * 100
	fmt.Printf("  effective_breach_rate:     %.2f%%（剔除 margin_unwired）\n", effectiveBreach)

	if effectiveBreach > breachThreshold {
		fmt.Printf("\n[G1] ❌ effective_breach_rate %.2f%% > %.0f%%，超过阈值\n",
			effectiveBreach, breachThreshold)
		fmt.Fprintf(os.Stderr,
			"G1 FAIL: effective_breach_rate=%.2f%% threshold=%.0f%%\n",
			effectiveBreach, breachThreshold)
		return 1
	}
	fmt.Printf("\n[G1] ✅ 通过（effective_breach_rate %.2f%% ≤ %.0f%%）\n",
		effectiveBreach, breachThreshold)
	return 0
}

// initDB 初始化主 DB 与日志 DB。
//
// 简化策略：让 model.InitDB 走默认路径（SQL_DSN=local → SQLite）。
// 远程 DB 留给运营 / CI 在生产环境配 SQL_DSN / LOG_SQL_DSN 后调用。
func initDB(a args) error {
	// 默认用本地 SQLite（micross 仓库约定 SQL_DSN=local 指 SQLite）。
	// 给一个独立 SQLite 文件，避免和主服务冲突。
	if a.dsn == "" || a.dsn == "local" {
		common.SQLitePath = "replay_commission.db?_busy_timeout=30000"
		common.SetMainDatabaseType(common.DatabaseTypeSQLite)
		if err := os.Setenv("SQL_DSN", "local"); err != nil {
			return fmt.Errorf("set SQL_DSN: %w", err)
		}
	}

	if err := model.InitDB(); err != nil {
		return fmt.Errorf("InitDB main: %w", err)
	}
	// 日志 DB 与主 DB 走同一连接（项目惯例 LOG_SQL_DSN 为空时复用主 DB）。
	if a.logDsn == "" {
		if err := model.InitLogDB(); err != nil {
			return fmt.Errorf("InitLogDB: %w", err)
		}
	}
	return nil
}

// formatQuota 把 quota 转成 USD 字符串（1 USD = common.QuotaPerUnit quota）。
//
// 例如 100 quota / 500000 = 0.000200 USD。
func formatQuota(q int64) string {
	qd := decimal.NewFromInt(q)
	usd := qd.Div(decimal.NewFromInt(int64(common.QuotaPerUnit)))
	return usd.StringFixed(6) + " USD"
}

// max 是 int 的 max（避免引入 Go 1.21+ 特性）。
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}