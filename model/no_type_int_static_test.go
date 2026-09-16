package model

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// task-14.1 §14.1 验收：去掉所有 `type:int` 标签，让 GORM 用数据库默认类型
// （SQLite = integer、MySQL = int、PostgreSQL = integer），三库行为一致，
// SQLite 上不再触发「检测到类型变化 → 整表重建」。
//
// 这个测试静态扫描 model/ 下所有 .go 文件（除 _test.go），断言
// `gorm:"type:int..."` 不再出现——只要有人手贱加回去，CI 立刻挂。
func TestNoTypeIntTagRemainsInModelStructs(t *testing.T) {
	fset := token.NewFileSet()
	paths, err := parser.ParseDir(fset, ".",
		func(fi fs.FileInfo) bool {
			name := fi.Name()
			return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
		},
		parser.ParseComments)
	require.NoError(t, err)

	var violations []string
	for _, pkg := range paths {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				ts, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, field := range st.Fields.List {
					if field.Tag == nil {
						continue
					}
					tag := strings.Trim(field.Tag.Value, "`")
					if !strings.Contains(tag, "gorm:") {
						continue
					}
					if hasTypeIntTag(tag) {
						fieldName := ""
						if len(field.Names) > 0 {
							fieldName = field.Names[0].Name
						} else if ident, ok := field.Type.(*ast.Ident); ok {
							fieldName = ident.Name
						}
						violations = append(violations,
							"struct "+ts.Name.Name+"."+fieldName+" tag=\""+tag+"\"")
					}
				}
				return true
			})
		}
	}

	assert.Empty(t, violations,
		"task-14.1 §14.1 验收：model/*.go 不应再有 `type:int` GORM 标签，"+
			"由 GORM 用 DB 默认类型（SQLite=integer / MySQL=int / PG=integer）渲染。\n"+
			"违规条目：\n"+strings.Join(violations, "\n"))
}

// hasTypeIntTag 检查 gorm tag 里是否带 `type:int` 子句。
// 匹配模式：`type:int` 后接 `;` 或 `"`（子句边界）；不接受 type:interval / type:integer 等。
func hasTypeIntTag(tag string) bool {
	const needle = "type:int"
	idx := strings.Index(tag, needle)
	if idx < 0 {
		return false
	}
	after := idx + len(needle)
	if after >= len(tag) {
		return true
	}
	next := tag[after]
	return next == ';' || next == '"' || next == ' '
}