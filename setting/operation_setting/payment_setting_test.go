package operation_setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsCorporatePaymentGroupAllowed(t *testing.T) {
	original := append([]string(nil), paymentSetting.CorporatePaymentAllowedGroups...)
	t.Cleanup(func() {
		paymentSetting.CorporatePaymentAllowedGroups = original
	})

	paymentSetting.CorporatePaymentAllowedGroups = nil
	assert.True(t, IsCorporatePaymentGroupAllowed("default"))

	paymentSetting.CorporatePaymentAllowedGroups = []string{"vip", " business "}
	assert.True(t, IsCorporatePaymentGroupAllowed("vip"))
	assert.True(t, IsCorporatePaymentGroupAllowed("business"))
	assert.False(t, IsCorporatePaymentGroupAllowed("default"))
}
