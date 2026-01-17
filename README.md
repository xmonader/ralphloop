# ralphloop
**The "Design-First" Autonomous Agent Loop**

RalphLoop is a Go-powered autonomous loop that orchestrates AI CLI agents (like Gemini, Claude, or custom scripts) through a rigorous **Plan -> Verify -> Execute -> Verify** lifecycle. It treats architectural design as a first-class citizen, forcing agents to stay in a "Design Box" before touching a single line of production code.


> In its purest form it's just `while :; do cat PROMPT.md | claude-code ; done` check the article on [Ralph Wiggum](https://ghuntley.com/ralph/)

## Key Concepts

### 1. The Design Box (.ralph/)
RalphLoop enforces a strict boundary. In the **Planning Phase**, the agent is restricted to the `.ralph/` directory. It must build its own "world view" here before execution:
- **`prd.json`**: Requirements, Tech Stack, and atomic User Stories.
- **`qa-plan.md`**: A matrix tracking Story IDs and verification status.
- **`guardrails.md`**: Persisted "Lessons Learned" (e.g., "avoid library X because of Y").

### 2. Autonomous Memory
RalphLoop isn't just a shell; it's a memory engine. Before every iteration, it injects the contents of `.ralph/` back into the agent's context. The agent "wakes up" knowing exactly what it decided, what it achieved, and what mistakes it needs to avoid.

### 3. Observability
Stop guessing what the agent is doing. RalphLoop maintains:
- **`activity.log`**: A chronological journal of iteration timing and phases.
- **`errors.log`**: A history of technical failures and recovery attempts.
- **`runs/`**: A complete archive of the raw output from every single iteration.

---

## Getting Started

### Prerequisites
- Go 1.21+
- An AI CLI tool (e.g., Gemini, Claude)

### Installation
```bash
git clone https://github.com/xmonader/ralphloop
cd ralphloop
make build
```

### Configuration
Set the `AGENT_CMD` environment variable. Use `{prompt}` as a placeholder for the generated context file.

**Example for Gemini:**
```bash
export AGENT_CMD='gemini -y -p "$(cat {prompt})"'
```

---

## Usage

### Phase 1: Planning & Design
Define your mission. Ralph will loop until it generates a PRD and QA plan you approve of.
```bash
./ralphloop plan "design a high-performance TCP router in Go"
```
- **Inside the loop**: Provide feedback or ask questions.
- **Finalizing**: Type `approve` when the plan in `.ralph/` looks solid.

### Phase 2: Execution
Once approved, switch to execution mode. Ralph now allows the agent to modify the entire workspace to implement the plan.
```bash
./ralphloop run
```

### Phase 3: Iteration
Want to add a feature? Re-run `plan`:
```bash
./ralphloop plan "ADD FEATURE: Add Prometheus metrics to the TCP router"
```
Ralph will load the existing `prd.json`, help the agent design the new stories, and wait for a new approval before execution.

---

## Options

- `-max <int>`: Cap the number of autonomous iterations (default: 10).
- `-v`: Enable verbose mode to see the full prompts being sent to the agent.

## Project Structure
- `cmd/ralphloop/`: Minimalist application entry point.
- `internal/loop/`: The core engine and iterative logic.
- `internal/loop/prompts/`: Embedded system prompts and personas.

## Cleanup
```bash
make clean
```
