from pathlib import Path

path = Path("/etc/nginx/sites-available/api.zhongzhuan.pro")
text = path.read_text()

block = """
    location /kiro-upstream/ {
        proxy_pass http://127.0.0.1:8000/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header CF-Connecting-IP $http_cf_connecting_ip;
        proxy_read_timeout 3600;
        proxy_send_timeout 3600;
        proxy_connect_timeout 60;
        proxy_buffering off;
        proxy_request_buffering off;
    }
"""

if "location /kiro-upstream/" in text:
    raise SystemExit(0)

marker = "    location / {\n        proxy_pass http://127.0.0.1:8080;"
if marker not in text:
    raise SystemExit("marker not found")

path.write_text(text.replace(marker, block + "\n" + marker))
