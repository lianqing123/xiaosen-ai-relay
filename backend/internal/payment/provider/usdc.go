package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

const (
	usdcDefaultNetwork = "USDC"
	usdcSignatureKey   = "x-usdc-signature"
)

type USDC struct {
	instanceID string
	config     map[string]string
	rate       float64
}

func NewUSDC(instanceID string, config map[string]string) (*USDC, error) {
	cfg := normalizeUSDCConfig(config)
	if strings.TrimSpace(cfg["address"]) == "" {
		return nil, fmt.Errorf("usdc config missing required key: address")
	}
	rate, err := strconv.ParseFloat(strings.TrimSpace(cfg["cnyPerUsdc"]), 64)
	if err != nil || rate <= 0 {
		return nil, fmt.Errorf("usdc config missing required key: cnyPerUsdc")
	}
	if strings.TrimSpace(cfg["network"]) == "" {
		cfg["network"] = usdcDefaultNetwork
	}
	return &USDC{instanceID: instanceID, config: cfg, rate: rate}, nil
}

func normalizeUSDCConfig(config map[string]string) map[string]string {
	cfg := make(map[string]string, len(config))
	for k, v := range config {
		cfg[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return cfg
}

func (u *USDC) Name() string        { return "USDC" }
func (u *USDC) ProviderKey() string { return payment.TypeUSDC }
func (u *USDC) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeUSDC}
}

func (u *USDC) CreatePayment(_ context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	cny, err := strconv.ParseFloat(strings.TrimSpace(req.Amount), 64)
	if err != nil || cny <= 0 {
		return nil, fmt.Errorf("usdc create payment: invalid amount")
	}
	amount := fmt.Sprintf("%.6f", cny/u.rate)
	orderID := strings.TrimSpace(req.OrderID)
	memo := orderID
	if prefix := strings.TrimSpace(u.config["memoPrefix"]); prefix != "" {
		memo = prefix + "-" + orderID
	}
	qr := u.renderQRCode(amount, memo)
	return &payment.CreatePaymentResponse{
		TradeNo: orderID,
		QRCode:  qr,
	}, nil
}

func (u *USDC) renderQRCode(amount, memo string) string {
	values := map[string]string{
		"address": strings.TrimSpace(u.config["address"]),
		"network": strings.TrimSpace(u.config["network"]),
		"amount":  amount,
		"order":   memo,
	}
	tmpl := strings.TrimSpace(u.config["qrTemplate"])
	if tmpl == "" {
		return fmt.Sprintf(
			"USDC\nNetwork: %s\nAddress: %s\nAmount: %s USDC\nOrder: %s",
			values["network"], values["address"], values["amount"], values["order"],
		)
	}
	for key, value := range values {
		tmpl = strings.ReplaceAll(tmpl, "{"+key+"}", value)
	}
	return tmpl
}

func (u *USDC) QueryOrder(_ context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	return &payment.QueryOrderResponse{
		TradeNo:  strings.TrimSpace(tradeNo),
		Status:   payment.ProviderStatusPending,
		Metadata: u.MerchantIdentityMetadata(),
	}, nil
}

func (u *USDC) VerifyNotification(_ context.Context, rawBody string, headers map[string]string) (*payment.PaymentNotification, error) {
	secret := strings.TrimSpace(u.config["webhookSecret"])
	if secret == "" {
		return nil, fmt.Errorf("usdc webhookSecret not configured")
	}
	if !verifyUSDCHMAC(rawBody, secret, headers) {
		return nil, fmt.Errorf("usdc invalid signature")
	}

	var payload struct {
		OutTradeNo string  `json:"out_trade_no"`
		OrderID    string  `json:"order_id"`
		TradeNo    string  `json:"trade_no"`
		TxHash     string  `json:"tx_hash"`
		Amount     float64 `json:"amount"`
		AmountCNY  float64 `json:"amount_cny"`
		USDCAmount float64 `json:"usdc_amount"`
		Status     string  `json:"status"`
	}
	if err := json.Unmarshal([]byte(rawBody), &payload); err != nil {
		return nil, fmt.Errorf("usdc parse notification: %w", err)
	}
	orderID := strings.TrimSpace(payload.OutTradeNo)
	if orderID == "" {
		orderID = strings.TrimSpace(payload.OrderID)
	}
	tradeNo := strings.TrimSpace(payload.TradeNo)
	if tradeNo == "" {
		tradeNo = strings.TrimSpace(payload.TxHash)
	}
	amount := payload.AmountCNY
	if amount <= 0 {
		amount = payload.Amount
	}
	if amount <= 0 && payload.USDCAmount > 0 {
		amount = payload.USDCAmount * u.rate
	}

	status := payment.ProviderStatusFailed
	switch strings.ToLower(strings.TrimSpace(payload.Status)) {
	case "success", "paid", "completed":
		status = payment.ProviderStatusSuccess
	case "pending":
		return nil, nil
	}

	metadata := u.MerchantIdentityMetadata()
	if tradeNo != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["tx_hash"] = tradeNo
	}
	return &payment.PaymentNotification{
		TradeNo:  tradeNo,
		OrderID:  orderID,
		Amount:   amount,
		Status:   status,
		RawData:  rawBody,
		Metadata: metadata,
	}, nil
}

func (u *USDC) Refund(context.Context, payment.RefundRequest) (*payment.RefundResponse, error) {
	return nil, fmt.Errorf("usdc refund is not supported")
}

func (u *USDC) MerchantIdentityMetadata() map[string]string {
	if u == nil {
		return nil
	}
	metadata := map[string]string{
		"network": strings.TrimSpace(u.config["network"]),
		"address": strings.TrimSpace(u.config["address"]),
	}
	return metadata
}

func verifyUSDCHMAC(rawBody, secret string, headers map[string]string) bool {
	signature := strings.TrimSpace(headers[usdcSignatureKey])
	if signature == "" {
		signature = strings.TrimSpace(headers["x-signature"])
	}
	signature = strings.TrimPrefix(strings.ToLower(signature), "sha256=")
	if signature == "" {
		return false
	}
	got, err := hex.DecodeString(signature)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(rawBody))
	return hmac.Equal(got, mac.Sum(nil))
}
