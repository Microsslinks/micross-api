/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package ratio_setting

import (
	"encoding/json"
	"testing"
)

// P5 task-12.3 倍率退役（master-plan §四 / task-12/task-list.md Phase 3.1）：
// GetGroupRatio 在所有读取处强制返回 1，业务方已迁移到 discount_bindings
// （task-12 phase 1）。配置保留便于回滚：本测试钉死"vip/svip 等分组都返 1"，
// rollback 验证 line 88 改 `_ = name; return 1` 回 `_ = name; return ratio`
// 时本测试会 fail，提示 reviewer 确认回滚决策。

func TestGetGroupRatioForcedOne(t *testing.T) {
	cases := []string{
		"default",
		"vip",
		"svip",
		"any-future-group-name",
		"",
	}
	for _, name := range cases {
		got := GetGroupRatio(name)
		if got != 1 {
			t.Errorf("GetGroupRatio(%q) = %v, want 1 (task-12.3 倍率退役：强制 1)", name, got)
		}
	}
}

// 配置保留校验：UpdateGroupRatioByJSONString 写入后，ContainsGroupRatio
// 仍能读到（虽然 GetGroupRatio 返回 1），证明 rollback 路径仍可恢复。

func TestGetGroupRatioPreservesConfiguration(t *testing.T) {
	// 用临时配置覆盖
	defer func() {
		_ = UpdateGroupRatioByJSONString(`{"default":1}`)
	}()

	// 写入含非 1 值的配置
	input := `{"vip":0.7,"svip":0.8,"default":1}`
	if err := UpdateGroupRatioByJSONString(input); err != nil {
		t.Fatalf("UpdateGroupRatioByJSONString 失败: %v", err)
	}

	// ContainsGroupRatio 应能查到（配置存在）
	if !ContainsGroupRatio("vip") {
		t.Errorf("ContainsGroupRatio(\"vip\") = false, 期望 true（配置应保留以备 rollback）")
	}

	// GetGroupRatio 必须仍返回 1（强制退役，不读配置）
	for _, name := range []string{"vip", "svip", "default"} {
		if got := GetGroupRatio(name); got != 1 {
			t.Errorf("GetGroupRatio(%q) = %v, want 1（即使配置是 %v）", name, got, name)
		}
	}

	// GetGroupRatioCopy 应能完整读出原始配置
	ratioMap := GetGroupRatioCopy()
	if ratioMap["vip"] != 0.7 {
		t.Errorf("GetGroupRatioCopy[\"vip\"] = %v, want 0.7（配置保留）", ratioMap["vip"])
	}
}

// UpdateGroupRatioByJSONString 应能正确解析合法 JSON，验证校验路径不受退役影响。

func TestUpdateGroupRatioByJSONStringValid(t *testing.T) {
	input := `{"default":1,"vip":0.8}`
	if err := UpdateGroupRatioByJSONString(input); err != nil {
		t.Fatalf("合法 JSON 写入失败: %v", err)
	}
}

// CheckGroupRatio 边界值校验：负数应被拒。

func TestCheckGroupRatioNegative(t *testing.T) {
	if err := CheckGroupRatio(`{"default":-0.1}`); err == nil {
		t.Errorf("CheckGroupRatio 接受负数, 期望失败")
	}
}

// CheckGroupRatio 合法值应通过。

func TestCheckGroupRatioValid(t *testing.T) {
	cases := []string{
		`{"default":1}`,
		`{"default":0.5,"vip":0.7}`,
		`{}`,
	}
	for _, jsonStr := range cases {
		if err := CheckGroupRatio(jsonStr); err != nil {
			t.Errorf("CheckGroupRatio(%s) 失败: %v", jsonStr, err)
		}
	}
}

// GetGroupGroupRatio 走 matrix 配置读取：业务侧使用 user × using group 二维矩阵
// 时仍生效；不在 12.3 退役范围（保留兼容）。

func TestGetGroupGroupRatioMatrixLookup(t *testing.T) {
	// 写入一个矩阵项
	input := `{"vip":{"default":0.9}}`
	if err := UpdateGroupGroupRatioByJSONString(input); err != nil {
		t.Fatalf("UpdateGroupGroupRatioByJSONString 失败: %v", err)
	}

	got, ok := GetGroupGroupRatio("vip", "default")
	if !ok {
		t.Errorf("GetGroupGroupRatio(\"vip\",\"default\") ok = false, 期望 true（matrix 配置保留）")
	}
	if got != 0.9 {
		t.Errorf("GetGroupGroupRatio(\"vip\",\"default\") = %v, want 0.9", got)
	}

	// 不存在组合应返回 ok=false
	if _, ok := GetGroupGroupRatio("vip", "nonexistent"); ok {
		t.Errorf("GetGroupGroupRatio(\"vip\",\"nonexistent\") ok = true, 期望 false")
	}
}

// Marshal → Unmarshal 往返测试：GroupRatio2JSONString 应能正确序列化。

func TestGroupRatio2JSONStringRoundTrip(t *testing.T) {
	// 写入
	if err := UpdateGroupRatioByJSONString(`{"default":1,"vip":0.7}`); err != nil {
		t.Fatalf("写入失败: %v", err)
	}

	jsonStr := GroupRatio2JSONString()

	// 解析回 map
	var got map[string]float64
	if err := json.Unmarshal([]byte(jsonStr), &got); err != nil {
		t.Fatalf("JSON 解析失败: %v", err)
	}

	if got["vip"] != 0.7 {
		t.Errorf("JSON round-trip: vip = %v, want 0.7", got["vip"])
	}
}