# Installs the firememory-setup Agent Skill globally for Claude Code (Windows).
#
#   irm https://raw.githubusercontent.com/phmotad/firememory/main/scripts/install-skill.ps1 | iex
#
# After install, open any project in Claude Code and type:
#   /firememory-setup
param(
    [string]$Branch = "main",
    [string]$SkillsDir = "$env:USERPROFILE\.claude\skills"
)

$ErrorActionPreference = "Stop"

$Repo      = "phmotad/firememory"
$SkillName = "firememory-setup"
$Dest      = Join-Path $SkillsDir $SkillName
$Base      = "https://raw.githubusercontent.com/$Repo/$Branch/.claude/skills/$SkillName"

Write-Host "Installing $SkillName skill..."

New-Item -ItemType Directory -Force -Path $Dest | Out-Null

Invoke-WebRequest "$Base/SKILL.md"   -OutFile (Join-Path $Dest "SKILL.md")
Invoke-WebRequest "$Base/install.md" -OutFile (Join-Path $Dest "install.md")

Write-Host ""
Write-Host "Skill installed to: $Dest"
Write-Host ""
Write-Host "Usage: open any project in Claude Code and type:"
Write-Host "  /firememory-setup"
Write-Host "  /firememory-setup cursor"
Write-Host "  /firememory-setup claude-code"
Write-Host "  /firememory-setup windsurf"
Write-Host "  /firememory-setup zed"
