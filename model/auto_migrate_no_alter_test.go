package model

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// sqliteColumnSnapshot 抓一张表每个字段的 (name, type, nullable, default)。
// 用作「前后两次 AutoMigrate 后表结构是否相同」的比对基准。
type sqliteColumnSnapshot struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

func snapshotSQLiteTable(t *testing.T, db *gorm.DB, table string) []sqliteColumnSnapshot {
	t.Helper()
	type rawCol struct {
		Cid     int
		Name    string
		Type    string
		Notnull int
		Dflt    sql.NullString
		Pk      int
	}
	var rows []rawCol
	require.NoError(t, db.Raw(`PRAGMA table_info(`+table+`)`).Scan(&rows).Error)
	out := make([]sqliteColumnSnapshot, 0, len(rows))
	for _, r := range rows {
		out = append(out, sqliteColumnSnapshot{
			Name:     r.Name,
			Type:     strings.ToLower(r.Type),
			Nullable: r.Notnull == 0,
			Default:  r.Dflt.String,
		})
	}
	return out
}

func snapshotEqual(a, b []sqliteColumnSnapshot) bool {
	if len(a) != len(b) {
		return false
	}
	aSorted := append([]sqliteColumnSnapshot(nil), a...)
	bSorted := append([]sqliteColumnSnapshot(nil), b...)
	sort.Slice(aSorted, func(i, j int) bool { return aSorted[i].Name < aSorted[j].Name })
	sort.Slice(bSorted, func(i, j int) bool { return bSorted[i].Name < bSorted[j].Name })
	for i := range aSorted {
		if aSorted[i] != bSorted[i] {
			return false
		}
	}
	return true
}

// 10 次 AutoMigrate 后表结构应与第 1 次完全一致（type:int 已去掉 → GORM 用
// SQLite 默认的 integer，第 2 次起不再判定「类型变化」、不再触发 ALTER）。
//
// baseline（带 type:int）下：第 1 次 type=int，第 2 次 glebarez DDL parser 解
// 析 CREATE TABLE 语句时识别出 GORM 后续想要的 integer 与之不同，触发重建
// → 「invalid DDL, unbalanced brackets」或真的 DROP+CREATE。本测试钉住
// 修复后的行为：列名、类型、nullable、默认值全程一致。
func TestAutoMigrateKeepsColumnSnapshotStableAcrossTenRuns(t *testing.T) {
	dsn := fmt.Sprintf("file:stable-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	models := []interface{}{
		&User{}, &DiscountPlan{}, &DiscountRule{}, &DiscountBinding{},
		&DiscountRoutingPolicy{}, &AgentProfile{}, &CustomerCode{},
	}
	require.NoError(t, db.AutoMigrate(models...))

	tables := []string{"users", "discount_plans", "discount_rules", "discount_bindings",
		"discount_routing_policies", "agent_profiles", "customer_codes"}
	firstSnapshots := make(map[string][]sqliteColumnSnapshot, len(tables))
	for _, tbl := range tables {
		firstSnapshots[tbl] = snapshotSQLiteTable(t, db, tbl)
	}

	// 再跑 9 次——每次表结构必须和首次一致。
	for i := 0; i < 9; i++ {
		require.NoError(t, db.AutoMigrate(models...),
			"第 %d 次 AutoMigrate 仍不应报错；type:int 去掉后 GORM 不再检测类型变化", i+2)
		for _, tbl := range tables {
			snap := snapshotSQLiteTable(t, db, tbl)
			assert.True(t, snapshotEqual(snap, firstSnapshots[tbl]),
				"表 %s 第 %d 次 AutoMigrate 后列快照与首次不一致——type:int 触发重建了",
				tbl, i+2)
		}
	}
}

// 抽查：去掉 type:int 后 GORM 在 SQLite 上用的字段类型。预期是 `integer`（默认），
// 而不是会被 glebarez 解析器当成「类型变化」的 `int` 渲染。
func TestUserFieldsUseDefaultIntegerTypeOnSQLite(t *testing.T) {
	dsn := fmt.Sprintf("file:integer-type-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}))

	snap := snapshotSQLiteTable(t, db, "users")
	require.NotEmpty(t, snap)

	// 找一个原本带 type:int 的字段（Role 列），确认类型是 integer 而不是 int。
	var roleType string
	for _, c := range snap {
		if c.Name == "role" {
			roleType = c.Type
			break
		}
	}
	assert.Equal(t, "integer", roleType,
		"SQLite 上 'role' 列应是 GORM 默认的 'integer' 类型，不是会被识别为类型变更的 'int'")

	// 抽查 status 列：原本也是 type:int，期望 integer。
	var statusType string
	for _, c := range snap {
		if c.Name == "status" {
			statusType = c.Type
			break
		}
	}
	assert.Equal(t, "integer", statusType,
		"SQLite 上 'status' 列同样应是 'integer' 类型")
}