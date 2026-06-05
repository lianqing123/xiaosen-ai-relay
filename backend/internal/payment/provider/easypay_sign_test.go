package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

func TestEasyPaySignConsistentOutput(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":          "1001",
		"type":         "alipay",
		"out_trade_no": "ORDER123",
		"name":         "Test Product",
		"money":        "10.00",
	}
	pkey := "test_secret_key"

	sign1 := easyPaySign(params, pkey)
	sign2 := easyPaySign(params, pkey)
	if sign1 != sign2 {
		t.Fatalf("easyPaySign should be deterministic: %q != %q", sign1, sign2)
	}
	if len(sign1) != 32 {
		t.Fatalf("MD5 hex should be 32 chars, got %d", len(sign1))
	}
}

func TestEasyPaySignExcludesSignAndSignType(t *testing.T) {
	t.Parallel()

	pkey := "my_key"
	base := map[string]string{
		"pid":  "1001",
		"type": "alipay",
	}
	withSign := map[string]string{
		"pid":       "1001",
		"type":      "alipay",
		"sign":      "should_be_ignored",
		"sign_type": "MD5",
	}

	signBase := easyPaySign(base, pkey)
	signWithExtra := easyPaySign(withSign, pkey)

	if signBase != signWithExtra {
		t.Fatalf("sign and sign_type should be excluded: base=%q, withExtra=%q", signBase, signWithExtra)
	}
}

func TestEasyPayRSASignContentExcludesSignAndSignType(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":       "1001",
		"type":      "alipay",
		"sign":      "should_be_ignored",
		"sign_type": "RSA",
	}

	got := easyPayRSASignContent(params)
	want := "pid=1001&type=alipay"

	if got != want {
		t.Fatalf("easyPayRSASignContent() = %q, want %q", got, want)
	}
}

func TestEasyPayCreateRedirectPaymentV2ExcludesSignTypeFromSignature(t *testing.T) {
	t.Parallel()

	privateKey, privatePEM, publicPEM := newEasyPayTestRSAKeyPair(t)
	provider := &EasyPay{
		config: map[string]string{
			"pid":         "1001",
			"apiBase":     "https://pay.example.com",
			"privateKey":  privatePEM,
			"publicKey":   publicPEM,
			"signType":    "RSA",
			"epayVersion": "v2",
		},
	}

	resp, err := provider.createRedirectPaymentV2(payment.CreatePaymentRequest{
		PaymentType: "alipay",
		OrderID:     "ORDER123",
		Subject:     "Test Product",
		Amount:      "10.00",
		NotifyURL:   "https://merchant.example.com/notify",
		ReturnURL:   "https://merchant.example.com/return",
	})
	if err != nil {
		t.Fatalf("createRedirectPaymentV2() error = %v", err)
	}

	parsed, err := url.Parse(resp.PayURL)
	if err != nil {
		t.Fatalf("parse pay url: %v", err)
	}
	values := parsed.Query()
	if values.Get("sign_type") != "RSA" {
		t.Fatalf("sign_type = %q, want RSA", values.Get("sign_type"))
	}

	params := map[string]string{}
	for key := range values {
		params[key] = values.Get(key)
	}
	if err := easyPayVerifyRSASign(params, publicPEM, values.Get("sign")); err != nil {
		t.Fatalf("signature should verify with sign_type excluded: %v; key=%v", err, privateKey.PublicKey.N.BitLen())
	}
	delete(params, "sign_type")
	if err := easyPayVerifyRSASign(params, publicPEM, values.Get("sign")); err != nil {
		t.Fatalf("signature should not depend on sign_type: %v", err)
	}
}

func TestEasyPayCreatePaymentV2UsesAPICreateEvenInPopupMode(t *testing.T) {
	t.Parallel()

	_, privatePEM, publicPEM := newEasyPayTestRSAKeyPair(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != easypayAPIPathV2 {
			t.Fatalf("path = %q, want %q", r.URL.Path, easypayAPIPathV2)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if got := r.Form.Get("sign_type"); got != signTypeRSA {
			t.Fatalf("sign_type = %q, want %q", got, signTypeRSA)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"success","trade_no":"T20260425001","pay_type":"qrcode","pay_info":"https://qr.example.com/pay"}`))
	}))
	defer server.Close()

	provider := &EasyPay{
		config: map[string]string{
			"pid":         "1001",
			"apiBase":     server.URL,
			"apiPath":     easypayAPIPathV2,
			"privateKey":  privatePEM,
			"publicKey":   publicPEM,
			"signType":    signTypeRSA,
			"epayVersion": easypayVersionV2,
			"paymentMode": paymentModePopup,
			"method":      easypayMethodWeb,
			"notifyUrl":   "https://merchant.example.com/notify",
			"returnUrl":   "https://merchant.example.com/return",
		},
		httpClient: server.Client(),
	}

	resp, err := provider.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		PaymentType: "alipay",
		OrderID:     "ORDER123",
		Subject:     "Test Product",
		Amount:      "10.00",
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("CreatePayment() error = %v", err)
	}
	if resp.PayURL != "" {
		t.Fatalf("PayURL = %q, want empty API QR response", resp.PayURL)
	}
	if resp.QRCode != "https://qr.example.com/pay" {
		t.Fatalf("QRCode = %q", resp.QRCode)
	}
}

func TestEasyPayDoesNotClaimUSDC(t *testing.T) {
	t.Parallel()

	provider := &EasyPay{
		config: map[string]string{
			"pid":     "1001",
			"apiBase": "https://pay.example.com",
		},
	}
	if easyPaySupportsType(provider.SupportedTypes(), payment.TypeUSDC) {
		t.Fatalf("SupportedTypes() = %v, must not include %q", provider.SupportedTypes(), payment.TypeUSDC)
	}
}

func easyPaySupportsType(types []payment.PaymentType, want payment.PaymentType) bool {
	for _, got := range types {
		if got == want {
			return true
		}
	}
	return false
}

func TestEasyPaySignExcludesEmptyValues(t *testing.T) {
	t.Parallel()

	pkey := "key123"
	base := map[string]string{
		"pid":  "1001",
		"type": "alipay",
	}
	withEmpty := map[string]string{
		"pid":      "1001",
		"type":     "alipay",
		"device":   "",
		"clientip": "",
	}

	signBase := easyPaySign(base, pkey)
	signWithEmpty := easyPaySign(withEmpty, pkey)

	if signBase != signWithEmpty {
		t.Fatalf("empty values should be excluded: base=%q, withEmpty=%q", signBase, signWithEmpty)
	}
}

func TestEasyPayVerifySignValid(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":          "1001",
		"type":         "alipay",
		"out_trade_no": "ORDER456",
		"money":        "25.00",
	}
	pkey := "secret"

	sign := easyPaySign(params, pkey)

	// Add sign to params (as would come in a real callback)
	params["sign"] = sign
	params["sign_type"] = "MD5"

	if !easyPayVerifySign(params, pkey, sign) {
		t.Fatal("easyPayVerifySign should return true for a valid signature")
	}
}

func TestEasyPayVerifySignTampered(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":          "1001",
		"type":         "alipay",
		"out_trade_no": "ORDER789",
		"money":        "50.00",
	}
	pkey := "secret"

	sign := easyPaySign(params, pkey)

	// Tamper with the amount
	params["money"] = "99.99"

	if easyPayVerifySign(params, pkey, sign) {
		t.Fatal("easyPayVerifySign should return false for tampered params")
	}
}

func TestEasyPayVerifySignWrongKey(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":  "1001",
		"type": "wxpay",
	}

	sign := easyPaySign(params, "correct_key")

	if easyPayVerifySign(params, "wrong_key", sign) {
		t.Fatal("easyPayVerifySign should return false with wrong key")
	}
}

func TestEasyPaySignEmptyParams(t *testing.T) {
	t.Parallel()

	sign := easyPaySign(map[string]string{}, "key123")
	if sign == "" {
		t.Fatal("easyPaySign with empty params should still produce a hash")
	}
	if len(sign) != 32 {
		t.Fatalf("MD5 hex should be 32 chars, got %d", len(sign))
	}
}

func TestEasyPaySignSortOrder(t *testing.T) {
	t.Parallel()

	pkey := "test_key"
	params1 := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}
	params2 := map[string]string{
		"c": "3",
		"a": "1",
		"b": "2",
	}

	sign1 := easyPaySign(params1, pkey)
	sign2 := easyPaySign(params2, pkey)

	if sign1 != sign2 {
		t.Fatalf("easyPaySign should be order-independent: %q != %q", sign1, sign2)
	}
}

func TestEasyPayVerifySignWrongSignValue(t *testing.T) {
	t.Parallel()

	params := map[string]string{
		"pid":  "1001",
		"type": "alipay",
	}
	pkey := "key"

	if easyPayVerifySign(params, pkey, "00000000000000000000000000000000") {
		t.Fatal("easyPayVerifySign should return false for an incorrect sign value")
	}
}

func TestEasyPayMerchantIdentityMetadata(t *testing.T) {
	t.Parallel()

	provider := &EasyPay{
		config: map[string]string{
			"pid": "1001",
		},
	}

	metadata := provider.MerchantIdentityMetadata()
	if metadata["pid"] != "1001" {
		t.Fatalf("pid = %q, want %q", metadata["pid"], "1001")
	}
}

func newEasyPayTestRSAKeyPair(t *testing.T) (*rsa.PrivateKey, string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key: %v", err)
	}
	privatePEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}))
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}))
	return key, privatePEM, publicPEM
}
