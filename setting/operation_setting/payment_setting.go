package operation_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type PaymentSetting struct {
	AmountOptions  []int           `json:"amount_options"`
	AmountDiscount map[int]float64 `json:"amount_discount"` // 充值金额对应的折扣，例如 100 元 0.9 表示 100 元充值享受 9 折优惠

	CorporatePaymentEnabled       bool     `json:"corporate_payment_enabled"`
	CorporatePaymentName          string   `json:"corporate_payment_name"`
	CorporatePaymentBank          string   `json:"corporate_payment_bank"`
	CorporatePaymentAccount       string   `json:"corporate_payment_account"`
	CorporatePaymentInstructions  string   `json:"corporate_payment_instructions"`
	CorporatePaymentMinTopUp      int      `json:"corporate_payment_min_topup"`
	CorporatePaymentContactPhone  string   `json:"corporate_payment_contact_phone"`
	CorporatePaymentContactWeChat string   `json:"corporate_payment_contact_wechat"`
	CorporatePaymentAllowedGroups []string `json:"corporate_payment_allowed_groups"`

	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
}

const CurrentComplianceTermsVersion = "v1"

// 默认配置
var paymentSetting = PaymentSetting{
	AmountOptions:                []int{10, 20, 50, 100, 200, 500},
	AmountDiscount:               map[int]float64{},
	CorporatePaymentMinTopUp:     1,
	CorporatePaymentInstructions: "Please complete the bank transfer first, then upload a clear transfer receipt. The balance will be credited after manual review.",
}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("payment_setting", &paymentSetting)
}

func GetPaymentSetting() *PaymentSetting {
	return &paymentSetting
}

func IsPaymentComplianceConfirmed() bool {
	return paymentSetting.ComplianceConfirmed &&
		paymentSetting.ComplianceTermsVersion == CurrentComplianceTermsVersion
}

// IsCorporatePaymentGroupAllowed treats an empty allowlist as visible to all groups.
func IsCorporatePaymentGroupAllowed(group string) bool {
	allowedGroups := paymentSetting.CorporatePaymentAllowedGroups
	if len(allowedGroups) == 0 {
		return true
	}
	for _, allowedGroup := range allowedGroups {
		if strings.TrimSpace(allowedGroup) == group {
			return true
		}
	}
	return false
}
