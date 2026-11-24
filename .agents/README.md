# Repository agent skills

This directory contains the instructions and metadata for skills specific to gh-settings.
The [code-review skill](skills/code-review/SKILL.md) defines how to review a change against
the CLI and Action contracts and how to assess the proposed squash message.

Edit the skill when review criteria change. Its [interface metadata](skills/code-review/agents/openai.yaml)
sets the display name, short description and default prompt. The [repository instructions](../AGENTS.md)
define when to use the skill.
