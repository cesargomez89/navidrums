# Navidrums Prompts

This folder contains example prompts for common software engineering tasks when working on Navidrums.

## Files

| File | Description |
|------|-------------|
| `refactor.md` | Prompts for refactoring code - extracting methods, reducing complexity, standardizing patterns |
| `bugfix.md` | Prompts for investigating and fixing bugs - stuck jobs, duplicates, database issues |
| `codereview.md` | Prompts for code review - architecture compliance, style checks, verification |
| `testing.md` | Prompts for writing tests - unit tests, integration tests, benchmarks |
| `feature.md` | Prompts for implementing new features - following the correct order |
| `tips.md` | General tips and improvements - optimization, logging, monitoring |

## How to Use

1. **Find the relevant prompt** in the appropriate file
2. **Include context files** listed at the top of the prompt (using `@filename`)
3. **Replace placeholders** like `{ServiceName}`, `{MethodName}` with actual values
4. **Customize** the context section with specific details
5. **Review** the requirements checklist before starting work
6. **Run** the verification commands at the end

## Project Context Files

Each prompt specifies which files to include:

- **@AGENTS.md** (always) - Project conventions, rules, commands, environment variables
- **@ARCHITECTURE.md** (optional) - Detailed layer structure and responsibilities
- **@DOMAIN.md** (optional) - Data models and job lifecycle
- **@API.md** (optional) - HTTP endpoints and HTMX patterns

### Token-Efficient Usage

For most tasks, **only @AGENTS.md is needed** - it contains:
- Build/test/lint commands
- Architecture flow and critical rules
- Job lifecycle
- Coding conventions

Use additional files only when you need detailed information:
- **@ARCHITECTURE.md** - When implementing across multiple layers
- **@DOMAIN.md** - When working with specific data models
- **@API.md** - When adding/modifying endpoints

## Quick Reference

See @AGENTS.md for full details on:
- Architecture flow and rules
- Job lifecycle
- Coding conventions
- Build commands

## Contributing

When you find a pattern that works well, add it to the appropriate file following the existing format:

1. Use clear, descriptive headers
2. Include context/background
3. List specific requirements
4. Provide examples where helpful
5. Include verification steps
