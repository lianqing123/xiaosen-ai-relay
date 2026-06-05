// Package provider contains concrete payment provider implementations.
package provider

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// EasyPay constants.
const (
	easypayVersionV1       = "v1"
	easypayVersionV2       = "v2"
	easypayCodeSuccessV1   = 1
	easypayCodeSuccessV2   = 0
	easypayStatusPaid      = 1
	easypayHTTPTimeout     = 10 * time.Second
	maxEasypayResponseSize = 1 << 20 // 1MB
	maxEasypayErrorSummary = 512
	tradeStatusSuccess     = "TRADE_SUCCESS"
	signTypeMD5            = "MD5"
	signTypeRSA            = "RSA"
	paymentModePopup       = "popup"
	paymentModeRedirect    = "redirect"
	deviceMobile           = "mobile"
	easypayMethodWeb       = "web"
	easypayAPIPathV1       = "/mapi.php"
	easypayAPIPathV2       = "/api/pay/create"
	easypaySubmitPathV1    = "/submit.php"
	easypaySubmitPathV2    = "/api/pay/submit"
)

// EasyPay implements payment.Provider for EasyPay-compatible aggregators.
// It supports both the legacy MD5 protocol and DuluPay's RSA v2 protocol.
type EasyPay struct {
	instanceID string
	config     map[string]string
	httpClient *http.Client
}

// NewEasyPay creates a new EasyPay provider.
// Supported config keys:
// - Common: pid, apiBase, notifyUrl, returnUrl, epayVersion
// - V1: pkey, cid, cidAlipay, cidWxpay
// - V2: privateKey, publicKey, apiPath, signType, method
func NewEasyPay(instanceID string, config map[string]string) (*EasyPay, error) {
	cfg := normalizeEasyPayConfig(config)
	for _, k := range []string{"pid", "apiBase", "notifyUrl", "returnUrl"} {
		if strings.TrimSpace(cfg[k]) == "" {
			return nil, fmt.Errorf("easypay config missing required key: %s", k)
		}
	}
	if cfg["epayVersion"] == easypayVersionV2 {
		for _, k := range []string{"privateKey", "publicKey"} {
			if strings.TrimSpace(cfg[k]) == "" {
				return nil, fmt.Errorf("easypay v2 config missing required key: %s", k)
			}
		}
	} else if strings.TrimSpace(cfg["pkey"]) == "" {
		return nil, fmt.Errorf("easypay config missing required key: pkey")
	}
	return &EasyPay{
		instanceID: instanceID,
		config:     cfg,
		httpClient: &http.Client{Timeout: easypayHTTPTimeout},
	}, nil
}

func normalizeEasyPayAPIBase(apiBase string) string {
	base := strings.TrimSpace(apiBase)
	if base == "" {
		return ""
	}
	if parsed, err := url.Parse(base); err == nil && parsed.Scheme != "" && parsed.Host != "" {
		parsed.RawQuery = ""
		parsed.Fragment = ""
		parsed.RawPath = ""
		parsed.Path = trimEasyPayEndpointPath(parsed.Path)
		return strings.TrimRight(parsed.String(), "/")
	}
	return strings.TrimRight(trimEasyPayEndpointPath(base), "/")
}

func trimEasyPayEndpointPath(path string) string {
	path = strings.TrimRight(strings.TrimSpace(path), "/")
	lower := strings.ToLower(path)
	for _, endpoint := range []string{"/api/pay/create", "/api/pay/submit", "/submit.php", "/mapi.php", "/api.php"} {
		if strings.HasSuffix(lower, endpoint) {
			return strings.TrimRight(path[:len(path)-len(endpoint)], "/")
		}
	}
	return path
}

func (e *EasyPay) apiBase() string {
	if e == nil {
		return ""
	}
	return normalizeEasyPayAPIBase(e.config["apiBase"])
}

func (e *EasyPay) Name() string        { return "EasyPay" }
func (e *EasyPay) ProviderKey() string { return payment.TypeEasyPay }
func (e *EasyPay) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeAlipay, payment.TypeWxpay}
}

func (e *EasyPay) MerchantIdentityMetadata() map[string]string {
	if e == nil {
		return nil
	}
	pid := strings.TrimSpace(e.config["pid"])
	if pid == "" {
		return nil
	}
	return map[string]string{"pid": pid}
}

func (e *EasyPay) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	if e.isV2() {
		return e.createAPIPaymentV2(ctx, req)
	}
	mode := strings.ToLower(strings.TrimSpace(e.config["paymentMode"]))
	if mode == paymentModePopup || mode == paymentModeRedirect {
		return e.createRedirectPaymentV1(req)
	}
	return e.createAPIPaymentV1(ctx, req)
}

func (e *EasyPay) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	if e.isV2() {
		return nil, fmt.Errorf("easypay v2 upstream query is not supported")
	}
	params := map[string]string{
		"act": "order", "pid": e.config["pid"],
		"key": e.config["pkey"], "out_trade_no": tradeNo,
	}
	body, err := e.post(ctx, buildEasyPayEndpoint(e.config["apiBase"], "/api.php"), params)
	if err != nil {
		return nil, fmt.Errorf("easypay query: %w", err)
	}
	var resp struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Status int    `json:"status"`
		Money  string `json:"money"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("easypay parse query: %w", err)
	}
	status := payment.ProviderStatusPending
	if resp.Status == easypayStatusPaid {
		status = payment.ProviderStatusPaid
	}
	amount, _ := strconv.ParseFloat(resp.Money, 64)
	return &payment.QueryOrderResponse{
		TradeNo:  tradeNo,
		Status:   status,
		Amount:   amount,
		Metadata: e.MerchantIdentityMetadata(),
	}, nil
}

func (e *EasyPay) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	values, err := url.ParseQuery(rawBody)
	if err != nil {
		return nil, fmt.Errorf("parse notify: %w", err)
	}
	params := make(map[string]string, len(values))
	for k := range values {
		params[k] = values.Get(k)
	}
	sign := strings.TrimSpace(params["sign"])
	if sign == "" {
		return nil, fmt.Errorf("missing sign")
	}

	if e.isV2() {
		if pid := strings.TrimSpace(params["pid"]); pid == "" || pid != strings.TrimSpace(e.config["pid"]) {
			return nil, fmt.Errorf("invalid merchant identity")
		}
		if err := easyPayVerifyRSASign(params, e.config["publicKey"], sign); err != nil {
			return nil, fmt.Errorf("invalid signature")
		}
	} else if !easyPayVerifySign(params, e.config["pkey"], sign) {
		return nil, fmt.Errorf("invalid signature")
	}

	status := payment.ProviderStatusFailed
	if strings.EqualFold(params["trade_status"], tradeStatusSuccess) {
		status = payment.ProviderStatusSuccess
	}
	amount, _ := strconv.ParseFloat(params["money"], 64)

	metadata := e.MerchantIdentityMetadata()
	if pid := strings.TrimSpace(params["pid"]); pid != "" {
		if metadata == nil {
			metadata = map[string]string{}
		}
		metadata["pid"] = pid
	}
	return &payment.PaymentNotification{
		TradeNo: params["trade_no"], OrderID: params["out_trade_no"],
		Amount: amount, Status: status, RawData: rawBody, Metadata: metadata,
	}, nil
}

func (e *EasyPay) Refund(ctx context.Context, req payment.RefundRequest) (*payment.RefundResponse, error) {
	if e.isV2() {
		return nil, fmt.Errorf("easypay v2 refund is not supported")
	}
	attempts := e.refundAttempts(req)
	if len(attempts) == 0 {
		return nil, fmt.Errorf("easypay refund missing order identifier")
	}
	var firstErr error
	for i, attempt := range attempts {
		body, status, err := e.postRaw(ctx, buildEasyPayEndpoint(e.config["apiBase"], "/api.php?act=refund"), attempt.params)
		if err != nil {
			return nil, fmt.Errorf("easypay refund request: %w", err)
		}
		if err := parseEasyPayRefundResponse(status, body); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			if i+1 < len(attempts) && isEasyPayRefundOrderNotFound(err) {
				continue
			}
			return nil, err
		}
		return &payment.RefundResponse{RefundID: attempt.refundID, Status: payment.ProviderStatusSuccess}, nil
	}
	return nil, firstErr
}

type easyPayRefundAttempt struct {
	params   map[string]string
	refundID string
}

func (e *EasyPay) refundAttempts(req payment.RefundRequest) []easyPayRefundAttempt {
	base := map[string]string{
		"pid": e.config["pid"], "key": e.config["pkey"], "money": req.Amount,
	}
	var attempts []easyPayRefundAttempt
	if orderID := strings.TrimSpace(req.OrderID); orderID != "" {
		params := cloneStringMap(base)
		params["out_trade_no"] = orderID
		attempts = append(attempts, easyPayRefundAttempt{params: params, refundID: orderID})
	}
	if tradeNo := strings.TrimSpace(req.TradeNo); tradeNo != "" {
		params := cloneStringMap(base)
		params["trade_no"] = tradeNo
		attempts = append(attempts, easyPayRefundAttempt{params: params, refundID: tradeNo})
	}
	return attempts
}

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func isEasyPayRefundOrderNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	return strings.Contains(msg, "订单编号不存在") ||
		strings.Contains(msg, "订单不存在") ||
		strings.Contains(lower, "order not found") ||
		strings.Contains(lower, "not exist")
}

func parseEasyPayRefundResponse(status int, body []byte) error {
	summary := summarizeEasyPayResponse(body)
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return fmt.Errorf("easypay refund HTTP %d: %s", status, summary)
	}

	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return fmt.Errorf("easypay refund empty response (HTTP %d): %s", status, summary)
	}

	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "<!doctype html") || strings.HasPrefix(lower, "<html") ||
		(strings.HasPrefix(lower, "<") && strings.Contains(lower, "html")) {
		return fmt.Errorf("easypay refund non-JSON response (HTTP %d): %s", status, summary)
	}

	var resp struct {
		Code any    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("easypay refund non-JSON response (HTTP %d): %s", status, summary)
	}
	if !easyPayResponseCodeIsSuccess(resp.Code) {
		msg := strings.TrimSpace(resp.Msg)
		if msg == "" {
			msg = summary
		}
		return fmt.Errorf("easypay refund failed (HTTP %d): %s", status, msg)
	}
	return nil
}

func easyPayResponseCodeIsSuccess(code any) bool {
	switch v := code.(type) {
	case float64:
		return int(v) == easypayCodeSuccessV1
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		return err == nil && n == easypayCodeSuccessV1
	default:
		return false
	}
}

func summarizeEasyPayResponse(body []byte) string {
	summary := strings.Join(strings.Fields(string(body)), " ")
	if summary == "" {
		return "<empty>"
	}
	if len(summary) > maxEasypayErrorSummary {
		return summary[:maxEasypayErrorSummary] + "..."
	}
	return summary
}

func (e *EasyPay) isV2() bool {
	return strings.EqualFold(strings.TrimSpace(e.config["epayVersion"]), easypayVersionV2)
}

func (e *EasyPay) resolveURLs(req payment.CreatePaymentRequest) (string, string) {
	notifyURL := req.NotifyURL
	if notifyURL == "" {
		notifyURL = e.config["notifyUrl"]
	}
	returnURL := req.ReturnURL
	if returnURL == "" {
		returnURL = e.config["returnUrl"]
	}
	return notifyURL, returnURL
}

func (e *EasyPay) resolveCID(paymentType string) string {
	if strings.HasPrefix(paymentType, "alipay") {
		if v := strings.TrimSpace(e.config["cidAlipay"]); v != "" {
			return v
		}
		return strings.TrimSpace(e.config["cid"])
	}
	if v := strings.TrimSpace(e.config["cidWxpay"]); v != "" {
		return v
	}
	return strings.TrimSpace(e.config["cid"])
}

// createRedirectPaymentV1 builds a legacy submit.php URL for browser redirect.
func (e *EasyPay) createRedirectPaymentV1(req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := e.resolveURLs(req)
	params := map[string]string{
		"pid": e.config["pid"], "type": req.PaymentType,
		"out_trade_no": req.OrderID, "notify_url": notifyURL,
		"return_url": returnURL, "name": req.Subject,
		"money": req.Amount,
	}
	if cid := e.resolveCID(req.PaymentType); cid != "" {
		params["cid"] = cid
	}
	if req.IsMobile {
		params["device"] = deviceMobile
	}
	params["sign"] = easyPaySign(params, e.config["pkey"])
	params["sign_type"] = signTypeMD5

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	payURL := buildEasyPayEndpoint(e.config["apiBase"], easypaySubmitPathV1) + "?" + q.Encode()
	return &payment.CreatePaymentResponse{PayURL: payURL}, nil
}

func (e *EasyPay) createAPIPaymentV1(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := e.resolveURLs(req)
	params := map[string]string{
		"pid": e.config["pid"], "type": req.PaymentType,
		"out_trade_no": req.OrderID, "notify_url": notifyURL,
		"return_url": returnURL, "name": req.Subject,
		"money": req.Amount, "clientip": req.ClientIP,
	}
	if cid := e.resolveCID(req.PaymentType); cid != "" {
		params["cid"] = cid
	}
	if req.IsMobile {
		params["device"] = deviceMobile
	}
	params["sign"] = easyPaySign(params, e.config["pkey"])
	params["sign_type"] = signTypeMD5

	body, err := e.post(ctx, buildEasyPayEndpoint(e.config["apiBase"], easypayAPIPathV1), params)
	if err != nil {
		return nil, fmt.Errorf("easypay create: %w", err)
	}
	var resp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		TradeNo string `json:"trade_no"`
		PayURL  string `json:"payurl"`
		PayURL2 string `json:"payurl2"`
		QRCode  string `json:"qrcode"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("easypay parse: %w", err)
	}
	if resp.Code != easypayCodeSuccessV1 {
		return nil, fmt.Errorf("easypay error: %s", resp.Msg)
	}
	payURL := resp.PayURL
	if req.IsMobile && resp.PayURL2 != "" {
		payURL = resp.PayURL2
	}
	return &payment.CreatePaymentResponse{TradeNo: resp.TradeNo, PayURL: payURL, QRCode: resp.QRCode}, nil
}

func (e *EasyPay) createRedirectPaymentV2(req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := e.resolveURLs(req)
	params := map[string]string{
		"pid": e.config["pid"], "type": req.PaymentType,
		"out_trade_no": req.OrderID, "notify_url": notifyURL,
		"return_url": returnURL, "name": req.Subject,
		"money": req.Amount, "timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}
	params["sign_type"] = e.config["signType"]
	sign, err := easyPaySignRSA(easyPayRSASignContent(params), e.config["privateKey"])
	if err != nil {
		return nil, fmt.Errorf("easypay v2 sign: %w", err)
	}
	params["sign"] = sign

	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	payURL := buildEasyPayEndpoint(e.config["apiBase"], easypaySubmitPathV2) + "?" + q.Encode()
	return &payment.CreatePaymentResponse{PayURL: payURL}, nil
}

func (e *EasyPay) createAPIPaymentV2(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	notifyURL, returnURL := e.resolveURLs(req)
	params := map[string]string{
		"pid": e.config["pid"], "method": e.config["method"], "type": req.PaymentType,
		"out_trade_no": req.OrderID, "notify_url": notifyURL,
		"return_url": returnURL, "name": req.Subject,
		"money": req.Amount, "clientip": req.ClientIP,
		"timestamp": strconv.FormatInt(time.Now().Unix(), 10),
	}
	params["sign_type"] = e.config["signType"]
	sign, err := easyPaySignRSA(easyPayRSASignContent(params), e.config["privateKey"])
	if err != nil {
		return nil, fmt.Errorf("easypay v2 sign: %w", err)
	}
	params["sign"] = sign

	body, err := e.post(ctx, buildEasyPayEndpoint(e.config["apiBase"], e.config["apiPath"]), params)
	if err != nil {
		return nil, fmt.Errorf("easypay v2 create: %w", err)
	}
	body, err = normalizeEasyPayResponseBody(body)
	if err != nil {
		return nil, fmt.Errorf("easypay v2 parse response body: %w", err)
	}

	var resp struct {
		Code    int    `json:"code"`
		Msg     string `json:"msg"`
		TradeNo string `json:"trade_no"`
		PayType string `json:"pay_type"`
		PayInfo string `json:"pay_info"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("easypay v2 parse: %w", err)
	}
	if resp.Code != easypayCodeSuccessV2 {
		return nil, fmt.Errorf("easypay v2 error: %s", resp.Msg)
	}
	result := &payment.CreatePaymentResponse{TradeNo: strings.TrimSpace(resp.TradeNo)}
	if strings.EqualFold(strings.TrimSpace(resp.PayType), "qrcode") {
		result.QRCode = strings.TrimSpace(resp.PayInfo)
	} else {
		result.PayURL = strings.TrimSpace(resp.PayInfo)
	}
	return result, nil
}

func (e *EasyPay) post(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	body, status, err := e.postRaw(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}
	if status < http.StatusOK || status >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("unexpected status: %d", status)
	}
	return body, nil
}

func (e *EasyPay) postRaw(ctx context.Context, endpoint string, params map[string]string) ([]byte, int, error) {
	form := url.Values{}
	for k, v := range params {
		if strings.TrimSpace(v) == "" {
			continue
		}
		form.Set(k, v)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Encoding", "identity")
	client := e.httpClient
	if client == nil {
		client = &http.Client{Timeout: easypayHTTPTimeout}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxEasypayResponseSize))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

func normalizeEasyPayConfig(config map[string]string) map[string]string {
	normalized := make(map[string]string, len(config)+10)
	for k, v := range config {
		normalized[k] = strings.TrimSpace(v)
	}

	alias := func(target string, keys ...string) {
		if strings.TrimSpace(normalized[target]) != "" {
			return
		}
		for _, key := range keys {
			if strings.TrimSpace(normalized[key]) != "" {
				normalized[target] = strings.TrimSpace(normalized[key])
				return
			}
		}
	}

	alias("pid", "merchant_id")
	alias("pkey", "merchant_key")
	alias("apiBase", "gateway_url", "gatewayUrl")
	alias("notifyUrl", "notify_url")
	alias("returnUrl", "return_url")
	alias("epayVersion", "epay_version")
	alias("apiPath", "api_path")
	alias("privateKey", "private_key")
	alias("publicKey", "platform_public_key", "public_key")
	alias("signType", "sign_type")
	normalized["apiBase"] = normalizeEasyPayAPIBase(normalized["apiBase"])

	version := strings.ToLower(strings.TrimSpace(normalized["epayVersion"]))
	if version == "" {
		version = easypayVersionV1
	}
	normalized["epayVersion"] = version

	if strings.TrimSpace(normalized["signType"]) == "" {
		if version == easypayVersionV2 {
			normalized["signType"] = signTypeRSA
		} else {
			normalized["signType"] = signTypeMD5
		}
	}
	if strings.TrimSpace(normalized["apiPath"]) == "" {
		if version == easypayVersionV2 {
			normalized["apiPath"] = easypayAPIPathV2
		} else {
			normalized["apiPath"] = easypayAPIPathV1
		}
	}
	if version == easypayVersionV2 && strings.TrimSpace(normalized["method"]) == "" {
		normalized["method"] = easypayMethodWeb
	}
	return normalized
}

func buildEasyPayEndpoint(baseURL, path string) string {
	base := normalizeEasyPayAPIBase(baseURL)
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return base
	}
	if strings.HasPrefix(trimmed, "/") {
		return base + trimmed
	}
	return base + "/" + trimmed
}

func normalizeEasyPayResponseBody(body []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 || trimmed[0] != '"' {
		return trimmed, nil
	}
	var inner string
	if err := json.Unmarshal(trimmed, &inner); err != nil {
		return nil, err
	}
	return bytes.TrimSpace([]byte(inner)), nil
}

func easyPaySign(params map[string]string, pkey string) string {
	content := easyPaySignContent(params) + pkey
	hash := md5.Sum([]byte(content))
	return hex.EncodeToString(hash[:])
}

func easyPayVerifySign(params map[string]string, pkey string, sign string) bool {
	return hmac.Equal([]byte(easyPaySign(params, pkey)), []byte(sign))
}

func easyPaySignContent(params map[string]string) string {
	return easyPaySignContentWithOptions(params, true)
}

func easyPayRSASignContent(params map[string]string) string {
	return easyPaySignContentWithOptions(params, true)
}

func easyPaySignContentWithOptions(params map[string]string, excludeSignType bool) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || (excludeSignType && k == "sign_type") || strings.TrimSpace(v) == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for i, k := range keys {
		if i > 0 {
			_ = buf.WriteByte('&')
		}
		_, _ = buf.WriteString(k + "=" + params[k])
	}
	return buf.String()
}

func easyPaySignRSA(content, privateKey string) (string, error) {
	key, err := parseEasyPayRSAPrivateKey(privateKey)
	if err != nil {
		return "", err
	}
	hashed := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func easyPayVerifyRSASign(params map[string]string, publicKey string, sign string) error {
	key, err := parseEasyPayRSAPublicKey(publicKey)
	if err != nil {
		return err
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sign))
	if err != nil {
		return err
	}
	hashed := sha256.Sum256([]byte(easyPayRSASignContent(params)))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, hashed[:], raw)
}

func parseEasyPayRSAPrivateKey(raw string) (*rsa.PrivateKey, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), "\\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	block, _ := pem.Decode([]byte(normalized))
	if block != nil {
		if strings.Contains(block.Type, "PRIVATE KEY") {
			if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
				if rsaKey, ok := key.(*rsa.PrivateKey); ok {
					return rsaKey, nil
				}
			}
		}
		if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
			return key, nil
		}
	}
	decoded, err := decodeEasyPayKeyBody(normalized)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(decoded); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	if key, err := x509.ParsePKCS1PrivateKey(decoded); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("invalid RSA private key")
}

func parseEasyPayRSAPublicKey(raw string) (*rsa.PublicKey, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(raw), "\\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")
	block, _ := pem.Decode([]byte(normalized))
	if block != nil {
		if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
			if rsaKey, ok := key.(*rsa.PublicKey); ok {
				return rsaKey, nil
			}
		}
		if key, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
			return key, nil
		}
	}
	decoded, err := decodeEasyPayKeyBody(normalized)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKIXPublicKey(decoded); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	if key, err := x509.ParsePKCS1PublicKey(decoded); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("invalid RSA public key")
}

func decodeEasyPayKeyBody(raw string) ([]byte, error) {
	lines := strings.Split(raw, "\n")
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "-----BEGIN ") || strings.HasPrefix(trimmed, "-----END ") {
			continue
		}
		parts = append(parts, trimmed)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty key body")
	}
	return base64.StdEncoding.DecodeString(strings.Join(parts, ""))
}
