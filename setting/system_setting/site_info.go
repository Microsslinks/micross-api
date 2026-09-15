package system_setting

import "github.com/QuantumNous/new-api/setting/config"

// SiteContactSettings 站点对外联系方式（页脚/关于页展示）
type SiteContactSettings struct {
	CompanyName  string `json:"company_name"`
	Phone        string `json:"phone"`
	Email        string `json:"email"`
	Address      string `json:"address"`
	ServiceHours string `json:"service_hours"`
}

// SiteSocialSettings 站点社交媒体链接（页脚/关于页展示）
type SiteSocialSettings struct {
	WechatQrcode string `json:"wechat_qrcode"`
	QQGroup      string `json:"qq_group"`
	Telegram     string `json:"telegram"`
	Discord      string `json:"discord"`
	GitHub       string `json:"github"`
	Twitter      string `json:"twitter"`
	YouTube      string `json:"youtube"`
	Bilibili     string `json:"bilibili"`
	WhatsApp     string `json:"whatsapp"`
	CustomLinks  string `json:"custom_links"`
}

// SiteContentSettings 内容页面显隐开关。
// 默认全部开启，兼容旧逻辑（内容非空即展示），管理员可手动关闭对应入口。
type SiteContentSettings struct {
	HomePageContentEnabled bool `json:"home_page_content_enabled"`
	AboutEnabled           bool `json:"about_enabled"`
	FooterEnabled          bool `json:"footer_enabled"`
	UserAgreementEnabled   bool `json:"user_agreement_enabled"`
	PrivacyPolicyEnabled   bool `json:"privacy_policy_enabled"`
}

var (
	defaultSiteContactSettings = SiteContactSettings{}
	defaultSiteSocialSettings  = SiteSocialSettings{}
	defaultSiteContentSettings = SiteContentSettings{
		HomePageContentEnabled: true,
		AboutEnabled:           true,
		FooterEnabled:          true,
		UserAgreementEnabled:   true,
		PrivacyPolicyEnabled:   true,
	}
)

func init() {
	config.GlobalConfig.Register("site_contact", &defaultSiteContactSettings)
	config.GlobalConfig.Register("site_social", &defaultSiteSocialSettings)
	config.GlobalConfig.Register("site_content", &defaultSiteContentSettings)
}

func GetSiteContactSettings() *SiteContactSettings {
	return &defaultSiteContactSettings
}

func GetSiteSocialSettings() *SiteSocialSettings {
	return &defaultSiteSocialSettings
}

func GetSiteContentSettings() *SiteContentSettings {
	return &defaultSiteContentSettings
}
