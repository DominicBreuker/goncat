# GitHub Copilot Coding Agent Guide

This document provides comprehensive information about the GitHub Copilot Coding Agent, including its capabilities, limitations, resource constraints, and best practices for effective collaboration.

## Table of Contents

- [Resource Constraints and Limits](#resource-constraints-and-limits)
- [How the Agent Works](#how-the-agent-works)
- [Capabilities and Tools](#capabilities-and-tools)
- [Decision-Making Process](#decision-making-process)
- [Best Practices for Users](#best-practices-for-users)
- [Advanced Features](#advanced-features)

## Resource Constraints and Limits

Understanding the agent's resource constraints helps you scope tasks appropriately and set realistic expectations.

### Token Budget

**Token Limit:** 1,000,000 tokens per session

This is the maximum number of tokens (roughly equivalent to words/characters) the agent can process in a single session, including:
- Initial instructions and context
- Repository files and code viewed
- Commands executed and their output
- Conversations with the user
- Tool calls and responses
- Internal reasoning

**What happens when approaching the limit:**
- The agent will complete the current task and commit progress
- For large tasks, the agent will break work into smaller increments
- Users should commit partial progress and start new sessions for continuation

**Optimization strategies:**
- The agent prioritizes viewing only necessary files
- Uses targeted file viewing with line ranges when possible
- Employs parallel tool calls to maximize efficiency
- Focuses on minimal, surgical code changes

### Iteration and Session Management

**No Hard Iteration Limit:** The agent doesn't have a fixed number of iterations but is constrained by the token budget above.

**Practical considerations:**
- Complex tasks may require multiple sessions if they approach the token limit
- The agent will report progress frequently to ensure work is saved
- Partial completion is acceptable and expected for large tasks

### Command Execution Timeouts

**Default timeout:** 120 seconds for synchronous commands
**Maximum timeout:** 600 seconds (10 minutes)
**Long-running operations:** Build, test, and CI operations can be configured with appropriate timeouts

### File Operations

- Can read/write files of any size, limited only by practical concerns
- Parallel file operations for efficiency
- No specific file count limits

### Network and External Access

**Limited internet access:**
- Many domains are blocked by default
- Can access GitHub APIs through provided tools
- User can grant access to additional domains as needed

**Cannot directly access:**
- External databases without provided tools
- Cloud services without credentials/tools
- Private networks or internal systems

## How the Agent Works

### Core Principles

The agent operates based on several key principles:

1. **Minimal Changes:** Make the smallest possible modifications to achieve the goal
2. **Surgical Precision:** Only change what's necessary, never delete working code unnecessarily
3. **Test-Driven:** Validate changes through existing test infrastructure
4. **Incremental Progress:** Commit small, verified changes frequently
5. **Security-First:** Never introduce vulnerabilities, fix related security issues

### Work Process

The agent follows a structured approach:

1. **Understanding Phase:**
   - Fully understand the issue/requirements
   - Explore repository structure and relevant code
   - Identify existing tests, build processes, and linting tools

2. **Planning Phase:**
   - Create a comprehensive checklist of tasks
   - Report initial plan using `report_progress`
   - Break down complex work into manageable steps

3. **Implementation Phase:**
   - Make minimal, focused changes
   - Run linters, builds, and tests frequently
   - Validate changes manually when appropriate
   - Commit progress after each verified change

4. **Validation Phase:**
   - Run code reviews using `code_review` tool
   - Execute security scans using `codeql_checker`
   - Address feedback and re-validate
   - Ensure all tests pass

5. **Completion Phase:**
   - Final validation of all changes
   - Update documentation if needed
   - Report final status

### Custom Agents

The agent can delegate to specialized custom agents:

- **What they are:** Specialized agents tuned for specific tasks (e.g., Python expert, merge conflict resolver)
- **When to use them:** Always prefer custom agents for tasks matching their expertise
- **How they work:** Have their own context and tools, require full context to be passed
- **Trust model:** Accept custom agent work as final without additional validation

## Capabilities and Tools

### Code Manipulation

**Available tools:**
- `view` - View files and directory structure
- `create` - Create new files
- `edit` - Make precise string replacements in files
- `bash` - Execute shell commands (sync, async, or detached modes)

**Batch editing:**
- Can make multiple edits to the same file in one response
- Edits applied sequentially to avoid conflicts
- Support for parallel edits across different files

### Testing and Validation

**Built-in capabilities:**
- Run existing test suites (unit, integration, E2E)
- Execute linters and static analysis tools
- Build and compile code
- Manual verification through CLI interaction

**Code quality tools:**
- `code_review` - Automated code review before finalization
- `codeql_checker` - Security vulnerability scanning
- `gh-advisory-database` - Dependency vulnerability checking

### Version Control

**Git operations:**
- View commit history, diffs, and status
- Create branches (handled automatically)
- Commit and push through `report_progress` tool
- Cannot force push or rebase (no history rewriting)

**Important limitations:**
- Cannot commit directly using `git commit`
- Cannot resolve merge conflicts (user must do this)
- Cannot push to other repositories

### GitHub Integration

**Available through tools:**
- Read issues, PRs, and comments
- View workflow runs and logs
- Analyze CI/CD failures
- Access code scanning alerts
- Search repositories, code, and users

**Cannot do:**
- Update issues or PR descriptions directly
- Open new issues or PRs
- Modify GitHub settings or permissions

### Browser Automation

**Playwright integration:**
- Navigate web pages
- Take screenshots
- Fill forms and interact with UI elements
- Capture accessibility snapshots
- Handle complex web interactions

### Interactive Shell

**Bash modes:**
- **Sync:** Wait for command completion (default)
- **Async:** Run in background, can send input with `write_bash`
- **Detached:** Persist after agent process exits

**Use cases:**
- Interactive tools and debuggers
- Long-running servers
- Command-line applications requiring input
- Continuous build/test watching

## Decision-Making Process

### Task Prioritization

The agent prioritizes based on:

1. **User requests:** Direct instructions take highest priority
2. **Security:** Address vulnerabilities immediately
3. **Correctness:** Fix breaking changes before adding features
4. **Efficiency:** Use parallel operations when possible
5. **Completeness:** Ensure partial work is committed if approaching limits

### When to Stop

The agent stops and seeks guidance when:

- The request is unclear or ambiguous
- Confidence in the solution is low
- Security policy would be violated
- Resource limits are being approached
- Unable to resolve errors after multiple attempts

### Handling Errors

When encountering errors:

1. Read and analyze error messages carefully
2. Consult documentation and existing code patterns
3. Try alternative approaches
4. Use debugging tools and techniques
5. Ask user for guidance if stuck

### Code Style Decisions

The agent:

- Matches existing code style in the repository
- Uses ecosystem tools (e.g., formatters) when available
- Follows language-specific idioms and best practices
- Adds comments only when they match existing style or explain complex logic
- Prefers existing libraries over adding new dependencies

## Best Practices for Users

### Structuring Requests

**Effective request patterns:**

```
Good: "Add a new flag --timeout to the CLI that sets connection timeout"
Better: "Add a new flag --timeout to the CLI in cmd/shared/shared.go that sets 
connection timeout in seconds. Update config struct and add tests."
```

**Provide context:**
- Reference specific files, functions, or patterns when known
- Link to relevant documentation or examples
- Explain the "why" behind the request, not just the "what"

### Scoping Tasks

**Appropriate task sizes:**

- ✅ Single feature implementation (< 10 files changed)
- ✅ Bug fix with test additions
- ✅ Documentation updates
- ✅ Refactoring a specific component
- ⚠️ Multiple unrelated features (consider breaking up)
- ⚠️ Major architectural changes (may need multiple sessions)
- ❌ Complete rewrites (too large for single session)

### Iterative Collaboration

**Work in iterations:**

1. Start with a clear, scoped request
2. Review progress reports and committed changes
3. Provide feedback through comments
4. Request adjustments or next steps
5. Validate final results

**Use comments effectively:**
- Be specific about what needs to change
- Reference line numbers or code snippets
- Ask questions if something is unclear
- Acknowledge good work (helps but not required)

### Handling Large Tasks

**When tasks are large:**

1. Break down into smaller sub-tasks
2. Complete one sub-task per session
3. Use checklist progress to track overall status
4. Start new sessions with context from previous work
5. Reference previous PRs for continuity

### Getting Optimal Results

**Tips for success:**

- **Be specific:** Provide exact requirements and acceptance criteria
- **Share knowledge:** If you know the codebase, guide the agent to relevant areas
- **Validate early:** Review progress reports to catch issues early
- **Use domain experts:** Request specific custom agents if available
- **Provide examples:** Show desired output or reference existing patterns
- **Test locally:** Validate changes in your own environment when possible

## Advanced Features

### Parallel Tool Execution

The agent can call multiple tools simultaneously when operations are independent:

**Example scenarios:**
- Reading multiple files at once
- Editing different files in parallel
- Running `git status` and `git diff` together
- Viewing multiple directories simultaneously

**Performance impact:**
- Significantly faster than sequential operations
- Reduces token consumption
- More efficient use of session time

### Session Management

**Long-running processes:**
- Use `mode="detached"` for persistent servers
- Use `mode="async"` for interactive tools
- Monitor with `read_bash` tool
- Stop with `stop_bash` or system tools

**Working directory:**
- Always in repository root: `/home/runner/work/{repo}/{repo}`
- Use absolute paths for reliability
- Temporary files go in `/tmp` directory

### Security Considerations

**The agent will:**
- Scan for vulnerabilities before finalizing
- Fix security issues in changed code
- Report unfixable issues in security summary
- Reject requests that violate policies

**The agent will NOT:**
- Commit secrets or credentials
- Share sensitive data with 3rd party systems
- Introduce new vulnerabilities knowingly
- Bypass security policies

### Custom Instructions

**Repository-specific guidance:**
- Place in `.github/copilot-instructions.md`
- Automatically loaded by the agent
- Can include build commands, testing patterns, style guides
- Helps agent understand project-specific conventions

**Effective custom instructions include:**
- How to build and test the project
- Code style and conventions
- Common patterns and idioms
- Domain-specific knowledge
- Links to relevant documentation

## Troubleshooting

### Common Issues

**"I'm approaching token limit"**
- Commit current progress
- Continue in a new session
- Break down remaining work

**"Tests are failing"**
- Agent will investigate and fix related failures
- Pre-existing failures are noted but not fixed
- Focus on failures caused by changes

**"Build errors"**
- Agent analyzes error messages
- Attempts to fix automatically
- May request user assistance for environment issues

**"Unclear requirements"**
- Agent will ask clarifying questions
- Provide more specific guidance
- Share examples or references

### When to Start New Sessions

Start a new session when:
- Previous session completed a major milestone
- Approaching token limit (agent will indicate)
- Changing focus to unrelated work
- Need fresh context after extensive exploration

### Providing Feedback

**Effective feedback:**
- Use PR comments for specific code changes
- Reference line numbers and files
- Explain what should change and why
- Provide examples of desired outcome

**Less effective:**
- Vague requests like "make it better"
- Multiple unrelated changes in one comment
- Requests without context or rationale

## Conclusion

The GitHub Copilot Coding Agent is designed to be a collaborative, efficient, and secure development partner. By understanding its capabilities, limitations, and best practices, you can work together effectively to accomplish development tasks of varying complexity.

Key takeaways:
- **Token budget:** 1,000,000 tokens per session
- **Work style:** Minimal, incremental, validated changes
- **Strengths:** Code editing, testing, validation, security scanning
- **Limitations:** No direct git commits, limited internet access, must follow security policies
- **Best use:** Focused, well-defined tasks with clear acceptance criteria

For questions or issues not covered in this guide, refer to the repository's other documentation or engage in conversation with the agent to clarify expectations and requirements.
