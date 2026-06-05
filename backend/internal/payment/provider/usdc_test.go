package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestNewUSDCRequiresAddressAndRate(t *testing.T) {
	t.Parallel()

	if _, err := NewUSDC("test", map[string]string{"cnyPerUsdc": "7.25"}); err == nil {
		t.Fatal("NewUSDC should require address")
	}
	if _, err := NewUSDC("test", map[string]string{"address": "0xmerchant"}); err == nil {
		t.Fatal("NewUSDC should require cnyPerUsdc")
	}
}

func TestUSDCCreatePaymentBuildsIndependentQRCode(t *testing.T) {
	t.Parallel()

	provider, err := NewUSDC("test", map[string]string{
		"address":    "0xmerchant",
		"network":    "Base",
		"cnyPerUsdc": "7.25",
	})
	if err != nil {
		t.Fatalf("NewUSDC() error = %v", err)
	}

	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "sub2_usdc_order",
		Amount:      "72.50",
		PaymentType: payment.TypeUSDC,
	})
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}
	if resp.TradeNo != "sub2_usdc_order" {
		t.Fatalf("TradeNo = %q, want order id", resp.TradeNo)
	}
	if resp.PayURL != "" {
		t.Fatalf("PayURL = %q, want empty independent QR flow", resp.PayURL)
	}
	for _, want := range []string{
		"USDC",
		"Network: Base",
		"Address: 0xmerchant",
		"Amount: 10.000000 USDC",
		"Order: sub2_usdc_order",
	} {
		if !strings.Contains(resp.QRCode, want) {
			t.Fatalf("QRCode = %q, want to contain %q", resp.QRCode, want)
		}
	}
}

func TestUSDCCreatePaymentUsesTemplateWithoutCallingGateway(t *testing.T) {
	t.Parallel()

	provider, err := NewUSDC("test", map[string]string{
		"address":    "0xmerchant",
		"network":    "Base",
		"cnyPerUsdc": "7.25",
		"qrTemplate": "usdc:{address}?amount={amount}&order={order}&network={network}",
		"memoPrefix": "zz",
	})
	if err != nil {
		t.Fatalf("NewUSDC() error = %v", err)
	}

	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID: "sub2_template_order",
		Amount:  "14.50",
	})
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}
	want := "usdc:0xmerchant?amount=2.000000&order=zz-sub2_template_order&network=Base"
	if resp.QRCode != want {
		t.Fatalf("QRCode = %q, want %q", resp.QRCode, want)
	}
}

func TestUSDCVerifySignedWebhook(t *testing.T) {
	t.Parallel()

	provider, err := NewUSDC("test", map[string]string{
		"address":       "0xmerchant",
		"network":       "Base",
		"cnyPerUsdc":    "7.25",
		"webhookSecret": "secret",
	})
	if err != nil {
		t.Fatalf("NewUSDC() error = %v", err)
	}

	body := `{"out_trade_no":"sub2_usdc_order","trade_no":"0xtx","amount":72.5,"status":"success"}`
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write([]byte(body))
	signature := hex.EncodeToString(mac.Sum(nil))

	notification, err := provider.VerifyNotification(context.Background(), body, map[string]string{
		"x-usdc-signature": signature,
	})
	if err != nil {
		t.Fatalf("VerifyNotification() error = %v", err)
	}
	if notification.OrderID != "sub2_usdc_order" || notification.TradeNo != "0xtx" {
		t.Fatalf("notification identifiers = %+v", notification)
	}
	if notification.Amount != 72.5 || notification.Status != payment.ProviderStatusSuccess {
		t.Fatalf("notification payment fields = %+v", notification)
	}
}
