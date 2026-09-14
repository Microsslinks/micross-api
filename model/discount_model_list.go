package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	// DiscountModelListNameMaxLength 清单名长度上界，与列宽 varchar(64) 对齐。
	DiscountModelListNameMaxLength = 64
	// DiscountModelListModelNameMaxLength 单个模型名长度上界，与报价核算 / 查价的入参上界一致。
	DiscountModelListModelNameMaxLength = 128
	// DiscountModelListMaxModels 一份清单最多存多少个模型。
	// 与 service.CustomerAuditMaxModels 取同一个数（100）：这份清单的用处就是填
	// 「客户报价核算 / 客户查价」那个模型清单格子，存得比那边一次能算的还多没有意义。
	DiscountModelListMaxModels = 100
)

// DiscountModelList 是一份起好名字、能反复使用的模型清单（比如「企业VIP标准包」＝20 个模型）。
//
// 这一版清单只解决「别再每次手打、重贴一串模型名」：它存一份名单，供管理员在
// 报价核算 / 查价 / 规则表单里一键带入，不参与任何计价——清单里有什么、
// 有没有被谁引用，都不影响某个客户按几折算钱。
//
// 让「折扣规则引用清单」是另一种做法（规则的作用范围多一种取值），它会动扣费裁决，
// 因此这一版不做：清单被改一下、被删一下，价格就可能跟着变，
// 得先定清楚它跟模型级规则同不同档、以及有规则在引用时还能不能删。
type DiscountModelList struct {
	Id     int    `json:"id"`
	Name   string `json:"name" gorm:"type:varchar(64);not null;uniqueIndex:uk_model_list_name"`
	Remark string `json:"remark" gorm:"type:varchar(255);not null;default:''"`
	// Models 是模型名单本身，库里存 JSON 数组文本（模型名可以带空格、大小写，
	// 逗号分隔会跟名字里的字符打架）。接口返回的是下面的 ModelNames。
	Models     string   `json:"-" gorm:"type:text;not null"`
	ModelNames []string `json:"models" gorm:"-"`
	CreatedAt  int64    `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64    `json:"updated_at" gorm:"bigint"`
}

func (DiscountModelList) TableName() string {
	return "discount_model_lists"
}

func (l *DiscountModelList) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	l.CreatedAt = now
	l.UpdatedAt = now
	return nil
}

func (l *DiscountModelList) BeforeUpdate(tx *gorm.DB) error {
	l.UpdatedAt = common.GetTimestamp()
	return nil
}

// SetModelNames 把模型名单写进库里的 JSON 文本，同时回填给接口输出用的字段。
// 走 common.Marshal（业务代码不直接用 encoding/json）。
func (l *DiscountModelList) SetModelNames(names []string) error {
	data, err := common.Marshal(names)
	if err != nil {
		return err
	}
	l.Models = string(data)
	l.ModelNames = names
	return nil
}

// DecodeModelNames 把库里的 JSON 文本解出来填进 ModelNames。
// 文本坏了（有人手工改过库）不报错也不编：按空清单处理并记一笔日志——
// 一条脏数据不该把整页清单拖挂，但这个「清单怎么空了」得留下线索。
func (l *DiscountModelList) DecodeModelNames() {
	names := make([]string, 0)
	if strings.TrimSpace(l.Models) != "" {
		if err := common.UnmarshalJsonStr(l.Models, &names); err != nil {
			common.SysError(fmt.Sprintf("模型清单 %s 的模型名单解析失败，按空清单处理：%v", l.Name, err))
			names = make([]string, 0)
		}
	}
	l.ModelNames = names
}

// NormalizeDiscountModelListNames 修剪并校验一份模型名单：去空白、去重、保留输入顺序。
// 空白行直接跳过（粘贴时常带一串空行），其余情况一律报错说清原因，
// 不做静默截断——静默截断会让人以为清单里存着的模型其实没存进去。
func NormalizeDiscountModelListNames(names []string) ([]string, error) {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}
		if len([]rune(name)) > DiscountModelListModelNameMaxLength {
			return nil, fmt.Errorf("模型名不能超过 %d 个字符", DiscountModelListModelNameMaxLength)
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	if len(result) == 0 {
		return nil, errors.New("模型清单不能为空")
	}
	if len(result) > DiscountModelListMaxModels {
		return nil, fmt.Errorf("一份清单最多 %d 个模型", DiscountModelListMaxModels)
	}
	return result, nil
}

// GetDiscountModelLists 分页查询清单；keyword 按名称与备注模糊匹配。
func GetDiscountModelLists(keyword string, offset int, limit int) ([]*DiscountModelList, int64, error) {
	db := DB.Model(&DiscountModelList{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR remark LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	lists := make([]*DiscountModelList, 0)
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&lists).Error; err != nil {
		return nil, 0, err
	}
	for _, list := range lists {
		list.DecodeModelNames()
	}
	return lists, total, nil
}

func GetDiscountModelListById(id int) (*DiscountModelList, error) {
	var list DiscountModelList
	if err := DB.First(&list, id).Error; err != nil {
		return nil, err
	}
	list.DecodeModelNames()
	return &list, nil
}

// IsDiscountModelListNameDuplicated 清单名全表唯一（对应 uk_model_list_name）。
func IsDiscountModelListNameDuplicated(excludeId int, name string) (bool, error) {
	var count int64
	err := DB.Model(&DiscountModelList{}).
		Where("name = ? AND id <> ?", name, excludeId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (l *DiscountModelList) Insert() error {
	return DB.Create(l).Error
}

// Update 整体更新清单。这一版接口不做局部更新，名称与名单必须一起给全，
// 所以 Models 也在可更新列里（不然「把清单改短」会存不进去）。
func (l *DiscountModelList) Update() error {
	return DB.Model(&DiscountModelList{}).Where("id = ?", l.Id).
		Select("name", "remark", "models", "updated_at").
		Updates(l).Error
}

// DeleteDiscountModelList 删除一份清单。
//
// 这一版清单不参与计价，删掉只影响「以后还能不能一键带入」——已经填进格子里
// 那些模型名不受影响，也没有任何客户的价格会因此变化，所以不需要引用保护。
// 将来若让规则引用清单，删除前必须先查有没有规则在用它（否则会静默换价）。
func DeleteDiscountModelList(id int) error {
	return DB.Delete(&DiscountModelList{}, id).Error
}
