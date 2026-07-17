package controller

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
	wechat "github.com/go-pay/gopay/wechat/v3"
	"github.com/shopspring/decimal"
)

const (
	goPayAlipayTradeSuccess  = "TRADE_SUCCESS"
	goPayAlipayTradeFinished = "TRADE_FINISHED"
)

func newGoPayAlipayClient() (*alipay.Client, error) {
	setting := operation_setting.GetPaymentSetting()
	client, err := alipay.NewClient(
		strings.TrimSpace(setting.GoPayAlipayAppID),
		strings.TrimSpace(setting.GoPayAlipayPrivateKey),
		!setting.GoPayAlipaySandbox,
	)
	if err != nil {
		return nil, err
	}
	callbackBase := strings.TrimRight(service.GetCallbackAddress(), "/")
	client.SetNotifyUrl(callbackBase + "/api/gopay/alipay/notify")
	client.SetReturnUrl(paymentReturnPath("/console/topup?show_history=true"))
	return client, nil
}

func newGoPayWeChatClient() (*wechat.ClientV3, error) {
	setting := operation_setting.GetPaymentSetting()
	client, err := wechat.NewClientV3(
		strings.TrimSpace(setting.GoPayWeChatMchID),
		strings.TrimSpace(setting.GoPayWeChatSerialNo),
		strings.TrimSpace(setting.GoPayWeChatAPIv3Key),
		strings.TrimSpace(setting.GoPayWeChatPrivateKey),
	)
	if err != nil {
		return nil, err
	}
	client.SetPlatformCert(
		[]byte(strings.TrimSpace(setting.GoPayWeChatPlatformCert)),
		strings.TrimSpace(setting.GoPayWeChatPlatformSerial),
	)
	return client, nil
}

func RequestGoPay(c *gin.Context) {
	var req EpayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.Amount < getMinTopup() {
		common.ApiErrorMsg(c, fmt.Sprintf("充值数量不能小于 %d", getMinTopup()))
		return
	}
	if req.PaymentMethod != model.PaymentMethodGoPayAlipay && req.PaymentMethod != model.PaymentMethodGoPayWeChat {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}
	if (req.PaymentMethod == model.PaymentMethodGoPayAlipay && !isGoPayAlipayEnabled()) ||
		(req.PaymentMethod == model.PaymentMethodGoPayWeChat && !isGoPayWeChatEnabled()) {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}

	userID := c.GetInt("id")
	group, err := model.GetUserGroup(userID, true)
	if err != nil {
		common.ApiErrorMsg(c, "获取用户分组失败")
		return
	}
	payMoney := getPayMoney(req.Amount, group)
	payCents := decimal.NewFromFloat(payMoney).Mul(decimal.NewFromInt(100)).Round(0).IntPart()
	if payCents < 1 {
		common.ApiErrorMsg(c, "充值金额过低")
		return
	}
	if payCents > math.MaxInt {
		common.ApiErrorMsg(c, "充值金额超出允许范围")
		return
	}

	tradeNo := fmt.Sprintf("GOPAY%d%s%d", userID, common.GetRandomString(6), time.Now().Unix())
	amount := req.Amount
	if operation_setting.GetQuotaDisplayType() == operation_setting.QuotaDisplayTypeTokens {
		amount = decimal.NewFromInt(amount).Div(decimal.NewFromFloat(common.QuotaPerUnit)).IntPart()
	}
	provider := model.PaymentProviderGoPayAlipay
	if req.PaymentMethod == model.PaymentMethodGoPayWeChat {
		provider = model.PaymentProviderGoPayWeChat
	}
	topUp := &model.TopUp{
		UserId: userID, Amount: amount, Money: float64(payCents) / 100, TradeNo: tradeNo,
		PaymentMethod: req.PaymentMethod, PaymentProvider: provider,
		CreateTime: time.Now().Unix(), Status: common.TopUpStatusPending,
	}

	if req.PaymentMethod == model.PaymentMethodGoPayAlipay {
		client, err := newGoPayAlipayClient()
		if err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝 client 初始化失败 trade_no=%s error=%q", tradeNo, err.Error()))
			common.ApiErrorMsg(c, "拉起支付失败")
			return
		}
		body := make(gopay.BodyMap)
		body.Set("subject", fmt.Sprintf("TUC%d", req.Amount)).
			Set("out_trade_no", tradeNo).
			Set("total_amount", strconv.FormatFloat(float64(payCents)/100, 'f', 2, 64)).
			Set("product_code", "FAST_INSTANT_TRADE_PAY")
		if err := topUp.Insert(); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("go-pay 创建订单失败 user_id=%d trade_no=%s error=%q", userID, tradeNo, err.Error()))
			common.ApiErrorMsg(c, "创建订单失败")
			return
		}
		payURL, err := client.TradePagePay(c.Request.Context(), body)
		if err != nil {
			_ = model.UpdatePendingTopUpStatus(tradeNo, provider, common.TopUpStatusFailed)
			logger.LogError(c.Request.Context(), fmt.Sprintf("支付宝下单失败 trade_no=%s error=%q", tradeNo, err.Error()))
			common.ApiErrorMsg(c, "拉起支付失败")
			return
		}
		common.ApiSuccess(c, gin.H{"pay_url": payURL, "trade_no": tradeNo})
		return
	}

	client, err := newGoPayWeChatClient()
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付 client 初始化失败 trade_no=%s error=%q", tradeNo, err.Error()))
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	if err := topUp.Insert(); err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("go-pay 创建订单失败 user_id=%d trade_no=%s error=%q", userID, tradeNo, err.Error()))
		common.ApiErrorMsg(c, "创建订单失败")
		return
	}
	callbackURL := strings.TrimRight(service.GetCallbackAddress(), "/") + "/api/gopay/wechat/notify"
	body := make(gopay.BodyMap)
	body.Set("appid", strings.TrimSpace(operation_setting.GetPaymentSetting().GoPayWeChatAppID)).
		Set("description", fmt.Sprintf("TUC%d", req.Amount)).
		Set("out_trade_no", tradeNo).
		Set("notify_url", callbackURL).
		SetBodyMap("amount", func(amountBody gopay.BodyMap) {
			amountBody.Set("total", int(payCents)).Set("currency", "CNY")
		})
	response, err := client.V3TransactionNative(c.Request.Context(), body)
	if err != nil || response == nil || response.Code != wechat.Success || response.Response == nil || response.Response.CodeUrl == "" {
		_ = model.UpdatePendingTopUpStatus(tradeNo, provider, common.TopUpStatusFailed)
		if err == nil && response != nil {
			err = fmt.Errorf("wechat response code=%d error=%s", response.Code, response.Error)
		}
		logger.LogError(c.Request.Context(), fmt.Sprintf("微信支付下单失败 trade_no=%s error=%q", tradeNo, fmt.Sprint(err)))
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	common.ApiSuccess(c, gin.H{"code_url": response.Response.CodeUrl, "trade_no": tradeNo})
}

func GoPayAlipayNotify(c *gin.Context) {
	if !isGoPayAlipayEnabled() {
		c.String(http.StatusBadRequest, "failure")
		return
	}
	body, err := alipay.ParseNotifyToBodyMap(c.Request)
	if err != nil {
		logger.LogWarn(c.Request.Context(), "支付宝回调解析失败: "+err.Error())
		c.String(http.StatusBadRequest, "failure")
		return
	}
	ok, err := alipay.VerifySign(strings.TrimSpace(operation_setting.GetPaymentSetting().GoPayAlipayPublicKey), body)
	if err != nil || !ok {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝回调验签失败 error=%q", fmt.Sprint(err)))
		c.String(http.StatusBadRequest, "failure")
		return
	}
	if body.GetString("app_id") != strings.TrimSpace(operation_setting.GetPaymentSetting().GoPayAlipayAppID) {
		logger.LogWarn(c.Request.Context(), "支付宝回调 App ID 与配置不匹配")
		c.String(http.StatusBadRequest, "failure")
		return
	}
	status := body.GetString("trade_status")
	if status != goPayAlipayTradeSuccess && status != goPayAlipayTradeFinished {
		c.String(http.StatusOK, "success")
		return
	}
	paidCents, err := decimal.NewFromString(body.GetString("total_amount"))
	if err != nil {
		c.String(http.StatusBadRequest, "failure")
		return
	}
	cents := paidCents.Mul(decimal.NewFromInt(100)).Round(0).IntPart()
	if err := model.CompleteGoPayTopUp(body.GetString("out_trade_no"), model.PaymentProviderGoPayAlipay, model.PaymentMethodGoPayAlipay, cents, c.ClientIP()); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("支付宝回调结算失败 trade_no=%s error=%q", body.GetString("out_trade_no"), err.Error()))
		c.String(http.StatusBadRequest, "failure")
		return
	}
	c.String(http.StatusOK, "success")
}

func GoPayWeChatNotify(c *gin.Context) {
	if !isGoPayWeChatEnabled() {
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "payment disabled"})
		return
	}
	notify, err := wechat.V3ParseNotify(c.Request)
	if err != nil {
		logger.LogWarn(c.Request.Context(), "微信支付回调解析失败: "+err.Error())
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "invalid notification"})
		return
	}
	client, err := newGoPayWeChatClient()
	if err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付 client 初始化失败 error=%q", err.Error()))
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "payment unavailable"})
		return
	}
	if notify.SignInfo == nil || notify.SignInfo.HeaderSerial != strings.TrimSpace(operation_setting.GetPaymentSetting().GoPayWeChatPlatformSerial) {
		logger.LogWarn(c.Request.Context(), "微信支付回调平台证书序列号与配置不匹配")
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "invalid certificate serial"})
		return
	}
	if err := notify.VerifySignByPK(client.WxPublicKey()); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付回调验签失败 error=%q", err.Error()))
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "invalid signature"})
		return
	}
	result, err := notify.DecryptCipherText(strings.TrimSpace(operation_setting.GetPaymentSetting().GoPayWeChatAPIv3Key))
	if err != nil || result == nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付回调解密失败 error=%q", fmt.Sprint(err)))
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "invalid resource"})
		return
	}
	if result.TradeState != wechat.TradeStateSuccess {
		c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: gopay.SUCCESS, Message: "成功"})
		return
	}
	if result.Amount == nil || result.Mchid != strings.TrimSpace(operation_setting.GetPaymentSetting().GoPayWeChatMchID) {
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "merchant or amount mismatch"})
		return
	}
	if err := model.CompleteGoPayTopUp(result.OutTradeNo, model.PaymentProviderGoPayWeChat, model.PaymentMethodGoPayWeChat, int64(result.Amount.Total), c.ClientIP()); err != nil {
		logger.LogWarn(c.Request.Context(), fmt.Sprintf("微信支付回调结算失败 trade_no=%s error=%q", result.OutTradeNo, err.Error()))
		c.JSON(http.StatusBadRequest, &wechat.V3NotifyRsp{Code: "FAIL", Message: "settlement failed"})
		return
	}
	c.JSON(http.StatusOK, &wechat.V3NotifyRsp{Code: gopay.SUCCESS, Message: "成功"})
}
