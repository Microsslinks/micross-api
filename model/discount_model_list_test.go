package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 模型清单（可复用的模型名单）的用例。
// 这张表不依赖用户 / 方案，所以自己起一个只有它的内存库（与 agent_code_test.go 同一套做法）。

func setupDiscountModelListTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:discount-model-list-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&DiscountModelList{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// 存进去再读出来，名单必须一模一样：顺序照输入（运营对着客户单子核对）、
// 空行与重复项已经清掉。
func TestDiscountModelListInsertAndReadRoundTrip(t *testing.T) {
	setupDiscountModelListTest(t)

	names, err := NormalizeDiscountModelListNames([]string{
		" gpt-4o ", "", "claude-3-5-sonnet", "gpt-4o", "Gemini 2.0 Flash",
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"gpt-4o", "claude-3-5-sonnet", "Gemini 2.0 Flash"}, names, "去空白、去重、保序；带空格的模型名要原样留着")

	list := &DiscountModelList{Name: "企业VIP标准包", Remark: "20 个模型"}
	require.NoError(t, list.SetModelNames(names))
	require.NoError(t, list.Insert())
	require.NotZero(t, list.Id)

	stored, err := GetDiscountModelListById(list.Id)
	require.NoError(t, err)
	assert.Equal(t, "企业VIP标准包", stored.Name)
	assert.Equal(t, "20 个模型", stored.Remark)
	assert.Equal(t, names, stored.ModelNames)
	assert.NotZero(t, stored.CreatedAt)
	assert.NotZero(t, stored.UpdatedAt)
}

// 前端靠 models 这个字段填格子，库里那串 JSON 文本不能漏出去。
func TestDiscountModelListJSONExposesModelNames(t *testing.T) {
	setupDiscountModelListTest(t)

	list := &DiscountModelList{Id: 7, Name: "一档"}
	require.NoError(t, list.SetModelNames([]string{"gpt-4o"}))
	raw, err := common.Marshal(list)
	require.NoError(t, err)

	var decoded map[string]interface{}
	require.NoError(t, common.Unmarshal(raw, &decoded))
	assert.Equal(t, []interface{}{"gpt-4o"}, decoded["models"])
	assert.NotContains(t, decoded, "Models")
}

// 同一份清单改名、改短名单都要存得进去：清单是给人用的，改起来必须随改随生效。
func TestDiscountModelListUpdateShrinksAndRenames(t *testing.T) {
	setupDiscountModelListTest(t)

	list := &DiscountModelList{Name: "旧名", Remark: "旧备注"}
	require.NoError(t, list.SetModelNames([]string{"a", "b", "c"}))
	require.NoError(t, list.Insert())
	// 更新时间戳可能与创建时间同秒，这里只要求有值，不比较大小。
	before := list.UpdatedAt

	list.Name = "新名"
	list.Remark = ""
	require.NoError(t, list.SetModelNames([]string{"a"}))
	require.NoError(t, list.Update())

	stored, err := GetDiscountModelListById(list.Id)
	require.NoError(t, err)
	assert.Equal(t, "新名", stored.Name)
	assert.Empty(t, stored.Remark)
	assert.Equal(t, []string{"a"}, stored.ModelNames, "名单改短必须存进去，不能被零值规则跳过")
	assert.GreaterOrEqual(t, stored.UpdatedAt, before)
}

// 重名要能查出来，且排除自己——不然编辑一份清单、什么都没改也会被自己挡住。
func TestDiscountModelListNameDuplication(t *testing.T) {
	setupDiscountModelListTest(t)

	first := &DiscountModelList{Name: "企业VIP标准包"}
	require.NoError(t, first.SetModelNames([]string{"gpt-4o"}))
	require.NoError(t, first.Insert())

	dup, err := IsDiscountModelListNameDuplicated(0, "企业VIP标准包")
	require.NoError(t, err)
	assert.True(t, dup)

	dup, err = IsDiscountModelListNameDuplicated(first.Id, "企业VIP标准包")
	require.NoError(t, err)
	assert.False(t, dup, "改自己时不该判成重名")

	dup, err = IsDiscountModelListNameDuplicated(0, "另一个名字")
	require.NoError(t, err)
	assert.False(t, dup)
}

// 清单名与模型名都是人填的，边界要当场说清，不能静默截断——
// 静默截断会让人以为清单里有的模型其实没存进去。
func TestNormalizeDiscountModelListNamesRejectsBadInput(t *testing.T) {
	_, err := NormalizeDiscountModelListNames([]string{"", "   "})
	assert.EqualError(t, err, "模型清单不能为空")

	tooMany := make([]string, DiscountModelListMaxModels+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("model-%d", i)
	}
	_, err = NormalizeDiscountModelListNames(tooMany)
	assert.EqualError(t, err, fmt.Sprintf("一份清单最多 %d 个模型", DiscountModelListMaxModels))

	_, err = NormalizeDiscountModelListNames([]string{strings.Repeat("x", DiscountModelListModelNameMaxLength+1)})
	assert.Error(t, err, "单个模型名超长要报错，不能让一条脏名字进库")
}

// 库里的 JSON 文本被手工改坏时按空清单处理：一条脏数据不该把整页清单拖挂。
func TestDiscountModelListDecodesBrokenJSONAsEmpty(t *testing.T) {
	setupDiscountModelListTest(t)

	list := &DiscountModelList{Name: "手工改坏的"}
	require.NoError(t, list.SetModelNames([]string{"gpt-4o"}))
	require.NoError(t, list.Insert())
	require.NoError(t, DB.Model(&DiscountModelList{}).Where("id = ?", list.Id).
		Update("models", "{not json").Error)

	stored, err := GetDiscountModelListById(list.Id)
	require.NoError(t, err)
	assert.Empty(t, stored.ModelNames)
}

// 删掉清单：查不到，而且同名的清单能重新建（不是软删留了个占位的）。
func TestDeleteDiscountModelList(t *testing.T) {
	setupDiscountModelListTest(t)

	list := &DiscountModelList{Name: "临时清单"}
	require.NoError(t, list.SetModelNames([]string{"gpt-4o"}))
	require.NoError(t, list.Insert())

	require.NoError(t, DeleteDiscountModelList(list.Id))
	_, err := GetDiscountModelListById(list.Id)
	require.Error(t, err)

	dup, err := IsDiscountModelListNameDuplicated(0, "临时清单")
	require.NoError(t, err)
	assert.False(t, dup, "删干净之后同名清单还能再建")

	again := &DiscountModelList{Name: "临时清单"}
	require.NoError(t, again.SetModelNames([]string{"gpt-4o"}))
	require.NoError(t, again.Insert(), "唯一索引不该被删掉的行占着")
}
