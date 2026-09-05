# Vasooli — Slash Commands (workaround for the Agent-tool bug)

## Why this exists
Claude Code's Agent tool currently has a confirmed, unresolved bug where project-level custom agents in `.claude/agents/` aren't registered as valid `subagent_type` values — you'll see `Agent type 'x' not found. Available agents: claude, claude-code-guide, Explore, general-purpose, Plan, statusline-setup` even with correctly formatted files. There's no fix on Anthropic's side yet.

Slash commands (`.claude/commands/`) are a separate, more stable mechanism. Typing `/data-schema-agent` directly injects that file's content as your prompt to the main session — no subagent dispatch involved, so the bug doesn't apply.

## Setup
1. Keep the original 4 files in `.claude/agents/` — harmless, and worth keeping in case a future Claude Code update fixes the Agent-tool bug.
2. Add these 4 new files to `.claude/commands/` (create the folder if it doesn't exist).
3. Restart Claude Code (or start a fresh session) so it picks up the new commands directory.

## Usage
Just type the command name directly — no path pasting needed:
- `/data-schema-agent`
- `/pipeline-logic-agent`
- `/dashboard-agent`
- `/qa-demo-agent`

Each one already ends with an instruction telling it to start proposing its first step — so running the command alone is enough to kick off the milestone.

## If this also doesn't show up
Run `claude update` first to make sure you're on the latest version — command discovery bugs have also been reported in some versions and are usually fixed quickly. If `/data-schema-agent` still doesn't appear in the command list or `/help`, tell me exactly what you see and we'll troubleshoot further.
