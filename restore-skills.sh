#!/usr/bin/env bash
# HostOVO 会话恢复脚本：技能 + 记忆 + 规则 一键加载
# 用法: bash restore-skills.sh (从项目根目录执行)

BASE="$(cd "$(dirname "$0")" && pwd)"

echo "🧠 恢复 HostOVO 会话... (路径: $BASE)"

# 1. 技能 (从 .codebuddy/skills/ 复制到全局 skills 目录)
if [ -d "$BASE/.codebuddy/skills" ]; then
  mkdir -p /root/.codebuddy/skills
  cp -r "$BASE/.codebuddy/skills/"* /root/.codebuddy/skills/ 2>/dev/null
  echo "  ✅ 技能 ($(ls "$BASE/.codebuddy/skills/" 2>/dev/null | wc -l) 个)"
else
  echo "  ⏭️ 技能目录不存在 (.codebuddy/skills/)"
fi

# 2. 永久规则
if [ -f "$BASE/AGENTS.md" ]; then
  cp "$BASE/AGENTS.md" /root/AGENTS.md 2>/dev/null || true
  echo "  ✅ AGENTS.md"
else
  echo "  ⏭️ AGENTS.md 不存在"
fi

# 3. 自进化记忆
if [ -d "$BASE/self-improving" ]; then
  cp -r "$BASE/self-improving" ~/ 2>/dev/null
  echo "  ✅ self-improving 记忆"
else
  echo "  ⏭️ self-improving 不存在"
fi

# 4. 结构化学习日志
if [ -d "$BASE/.learnings" ]; then
  cp -r "$BASE/.learnings" ~/ 2>/dev/null
  echo "  ✅ .learnings ($(ls "$BASE/.learnings/"*.md 2>/dev/null | wc -l) 文件)"
else
  echo "  ⏭️ .learnings 不存在"
fi

# 汇总
echo ""
echo "恢复内容:"
echo "  类别        数量      位置"
echo "  技能        $(ls /root/.codebuddy/skills/ 2>/dev/null | wc -l) 个     /root/.codebuddy/skills/"
[ -f /root/AGENTS.md ] && echo "  永久规则    12 条    /root/AGENTS.md" || echo "  永久规则    --       未加载"
[ -d ~/.learnings ] && echo "  自进化记忆   $(ls ~/.learnings/*.md 2>/dev/null | wc -l) 条    ~/.learnings/" || echo "  自进化记忆   --       未加载"
echo ""
echo "🎯 就绪。可开始 HostOVO 开发。"
