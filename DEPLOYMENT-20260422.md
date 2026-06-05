## Sub2API Deployment Record

Date: 2026-04-22

### Local Reference

- Repository: `F:\codex项目\中转站项目\sub2api`

### Server Layout

- Server: `root@23.169.169.107`
- App directory: `/opt/sub2api`
- Deploy directory: `/opt/sub2api/deploy`
- Runtime compose file: `/opt/sub2api/deploy/docker-compose.hostnet.yml`

### Routing

- Frontend root site remains on `https://zhongzhuan.pro`
- Sub2API is exposed on `https://api.zhongzhuan.pro`
- Nginx proxies `api.zhongzhuan.pro` to `127.0.0.1:8080`

### Runtime Notes

- Containers: `sub2api`, `sub2api-postgres`, `sub2api-redis`
- Deployment uses Docker host networking with loopback binds to avoid the broken Docker bridge/iptables chain on this server
- Nginx `http` block includes `underscores_in_headers on;` for Codex CLI sticky-session headers

### Payment Notes

- Runtime image for `sub2api` has been replaced with `sub2api:custom`
- Custom backend includes EasyPay/DuluPay v2 RSA support for `easypay`
- Imported payment channels from the cardshop payment account:
  - `DuluPay Alipay` -> supported type `alipay`
  - `DuluPay WeChat Pay` -> supported type `wxpay`
- Admin payment config now enables `alipay` and `wxpay`
- Checkout info exposes both methods successfully
- Actual order creation is currently blocked by upstream DuluPay domain whitelisting:
  - error: `api.zhongzhuan.pro is not approved in the DuluPay domain whitelist`
