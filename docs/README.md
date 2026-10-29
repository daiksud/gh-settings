# Documentation

Choose the documentation for your task. Users install and operate gh-settings; contributors change its implementation, tests, documentation, or project automation.

## For users

Start with the [README quick start](../README.md#quick-start).

| Task | Guide |
| --- | --- |
| Export, validate, plan, or apply; understand targeting and errors | [Command reference](commands.md) |
| Declare repository settings, topics, and labels in YAML | [Configuration reference](settings.md) |
| Use the public Action and configure approval and credentials | [GitHub Actions guide](github-actions.md) |
| Check which published Action/CLI a copied workflow runs | [Published package and consumer pins](github-actions.md#published-package-and-consumer-pins) |
| Protect apply with Environment review and trusted required checks | [Apply protection guide](github-actions.md#protect-the-apply-path) |
| Adopt the example PR and main workflows | [Workflow examples](../examples/github-actions/README.md) |

These public guides describe the supported interface and its effects, marking source-only capabilities separately from published releases. You do not need the implementation or release procedures to use gh-settings.

## For contributors

Start with [CONTRIBUTING](../CONTRIBUTING.md), including the [documentation placement rules](../CONTRIBUTING.md#where-documentation-belongs).

| Task | Guide |
| --- | --- |
| Install development tools, build, and extend the implementation | [Development](contributing/development.md) |
| Understand the current design and code boundaries | [Architecture](contributing/architecture.md) |
| Inspect the prepared About, General and Labels UI/API boundary | [Repository UI mapping](contributing/ui-repository-mapping.md) |
| Look up declaration, observed-state, and comparison terminology | [Model glossary](contributing/glossary.md) |
| Inspect the prepared Ruleset UI/API boundary and write-only provenance | [Ruleset UI mapping](contributing/ui-ruleset-mapping.md) |
| Select checks and understand local/CI evidence | [Testing](contributing/testing.md) |
| Review PR scope, squash messages, compatibility, and implementation | [Code-review policy](../.agents/skills/code-review/SKILL.md) |
| Change the private metadata, plan-report, or Summary protocol | [Actions internals](contributing/actions-internals.md) |
| Maintain this repository's CI, review, and settings automation | [Repository automation](contributing/repository-automation.md) |
| Prepare, publish, or recover a release | [Releases](contributing/releases.md) |
| Understand the reasons for a design decision | [Architecture decision records](adr/README.md) |
| Find detailed scenarios and their Go test mappings | [Behavior specifications](features/README.md) |

The user guides stay directly under `docs/`; contributor guides live under `docs/contributing/`. ADRs and behavior specifications keep their established paths as contributor reference material.

Internal preparation: [Actions UI mappings](contributing/ui-actions-mapping.md).

Internal preparation: [Security UI mappings](contributing/ui-security-mapping.md).

Internal preparation: [Pages UI mappings](contributing/ui-pages-mapping.md).
