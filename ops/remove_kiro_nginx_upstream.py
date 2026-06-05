from pathlib import Path

path = Path("/etc/nginx/sites-available/api.zhongzhuan.pro")
text = path.read_text()
start = text.find("\n    location /kiro-upstream/ {")
if start == -1:
    raise SystemExit(0)

end = text.find("\n    location / {", start)
if end == -1:
    raise SystemExit("next location not found")

path.write_text(text[:start] + "\n" + text[end:])
