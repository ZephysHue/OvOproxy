#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

PASS=0; FAIL=0
pass() { echo "  ✅ $1"; ((PASS++)) || true; }
fail() { echo "  ❌ $1"; ((FAIL++)) || true; }

echo "========================================="
echo "  HostOVO 推送前自测"
echo "========================================="

# --- 1. Go 交叉编译 ---
echo "[1/5] Go 交叉编译 (windows/amd64)"
mkdir -p frontend/dist
echo '<!DOCTYPE html><html><body></body></html>' > frontend/dist/index.html
if CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /dev/null . 2>&1; then
  pass "Go 编译通过"
else
  fail "Go 编译失败"
fi

# --- 2. 前端 TypeScript 检查 (如果有 node_modules) ---
echo "[2/5] 前端 TypeScript 检查"
if [ -d "frontend/node_modules" ]; then
  if cd frontend && npx vue-tsc --noEmit 2>&1; then
    pass "TypeScript 类型检查通过"
  else
    fail "TypeScript 类型检查失败"
  fi
  cd ..
else
  echo "  ⏭️  跳过 (node_modules 未安装)"
fi

# --- 3. 嵌入资源路径检查 ---
echo "[3/5] 嵌入资源路径检查"
EMBED_OK=true
for embed_target in $(grep -r "go:embed" --include="*.go" . | sed 's/.*go:embed //' | sed 's/all://'); do
  target=$(echo "$embed_target" | tr -d '[:space:]')
  if [ -e "$target" ] || [ -d "$target" ]; then
    pass "embed: $target"
  else
    fail "embed: $target 缺失!"
    EMBED_OK=false
  fi
done

# --- 4. 关键文件完整性 ---
echo "[4/5] 关键文件完整性"
for f in appicon.png assets/tray.ico frontend/src/assets/images/logo-universal.png wails.json build.bat; do
  if [ -f "$f" ]; then
    pass "$f ($(wc -c < "$f") bytes)"
  else
    fail "$f 缺失!"
  fi
done

# Icon 格式验证
if file appicon.png | grep -q "PNG"; then pass "appicon.png 是 PNG"; else fail "appicon.png 不是 PNG"; fi
if file assets/tray.ico | grep -q "MS Windows icon"; then pass "tray.ico 是 ICO"; else fail "tray.ico 不是 ICO"; fi

# --- 5. 旧名称残留检查 ---
echo "[5/5] 旧名称残留检查"
check_residue() {
  local pattern="$1" label="$2"
  local hits
  hits=$(grep -rn "$pattern" --include="*.go" --include="*.json" --include="*.ts" --include="*.vue" --include="*.bat" . 2>/dev/null | grep -v CHANGELOG.md | grep -v README.md | grep -v verify.sh | grep -v '.cursor/archives/' || true)
  if [ -z "$hits" ]; then
    pass "无 '$label' 残留"
  else
    fail "'$label' 残留:"
    echo "$hits" | while read -r line; do echo "       $line"; done
  fi
}
check_residue "ZephyHosts" "ZephyHosts"
check_residue "MultiHostProxy" "MultiHostProxy"

# wails.json 中的名称
if grep -q '"outputfilename": "HostOVO"' wails.json; then
  pass "wails.json outputfilename = HostOVO"
else
  fail "wails.json outputfilename 不正确!"
fi

# build.bat 中的名称
if grep -q "HostOVO.exe" build.bat; then
  pass "build.bat 引用 HostOVO.exe"
else
  fail "build.bat 未引用 HostOVO.exe!"
fi

echo "========================================="
echo "  结果: $PASS 通过 / $((PASS+FAIL)) 项"
if [ "$FAIL" -gt 0 ]; then
  echo "  ❌ 有 $FAIL 项失败，请修复后再推送"
  exit 1
else
  echo "  ✅ 全部通过"
fi
