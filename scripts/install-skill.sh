#!/usr/bin/env bash
# Installs the firememory-setup Agent Skill globally for Claude Code.
#
#   curl -fsSL https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install-skill.sh | bash
#
# After install, open any project in Claude Code and type:
#   /firememory-setup
set -euo pipefail

REPO="phmotad/firememory"
BRANCH="${BRANCH:-main}"
SKILL_NAME="firememory-setup"
SKILLS_DIR="${SKILLS_DIR:-${HOME}/.claude/skills}"
DEST="${SKILLS_DIR}/${SKILL_NAME}"

SKILL_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/.claude/skills/${SKILL_NAME}/SKILL.md"
INSTALL_URL="https://raw.githubusercontent.com/${REPO}/${BRANCH}/.claude/skills/${SKILL_NAME}/install.md"

echo "Installing ${SKILL_NAME} skill..."

mkdir -p "${DEST}"

curl -fsSL "${SKILL_URL}"   -o "${DEST}/SKILL.md"
curl -fsSL "${INSTALL_URL}" -o "${DEST}/install.md"

echo ""
echo "Skill installed to: ${DEST}"
echo ""
echo "Usage: open any project in Claude Code and type:"
echo "  /firememory-setup"
echo "  /firememory-setup cursor"
echo "  /firememory-setup claude-code"
echo "  /firememory-setup windsurf"
echo "  /firememory-setup zed"
