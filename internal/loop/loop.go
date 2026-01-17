package loop

import (
    "bufio"
    "embed"
    "flag"
    "fmt"
    "io"
    "log"
    "os"
    "os/exec"
    "strings"
    "time"
)

//go:embed prompts/*.md
var promptsFS embed.FS

const (
    defaultMaxIterations = 10
    ralphDir             = ".ralph"
    promptFile           = ralphDir + "/PROMPT.md"
    runsDir              = ralphDir + "/runs"
    progressFile         = ralphDir + "/progress.md"
    guardrailsFile       = ralphDir + "/guardrails.md"
    activityFile         = ralphDir + "/activity.log"
    errorsFile           = ralphDir + "/errors.log"
)

func runAgent(agentCmd, promptPath string, verbose bool) (string, error) {
    cmdStr := strings.ReplaceAll(agentCmd, "{prompt}", promptPath)

    if verbose {
        log.Printf("[DEBUG] Agent Command: %s", cmdStr)
    } else {
        log.Println("--- Consulting Agent (Streaming) ---")
    }

    cmd := exec.Command("bash", "-c", cmdStr)
    var outBuf strings.Builder
    multi := io.MultiWriter(os.Stdout, &outBuf)
    cmd.Stdout = multi
    cmd.Stderr = multi

    if err := cmd.Start(); err != nil {
        return "", fmt.Errorf("failed to start agent: %v", err)
    }

    err := cmd.Wait()
    if err != nil {
        return outBuf.String(), fmt.Errorf("agent process finished with error: %v", err)
    }
    return outBuf.String(), nil
}

func printUsage() {
    log.Println("Usage: ralphloop <command> [options] [goal]")
    log.Println("Commands:")
    log.Println("  plan <goal>   Start or resume the planning phase.")
    log.Println("  run [goal]    Execute an approved plan.")
    log.Println("Global Options:")
    log.Println("  -max int      Maximum number of iterations (default 10)")
    log.Println("  -v            Verbose mode")
    log.Println("Examples:")
    log.Println("  # Gemini (Google OSS CLI)")
    log.Println("  AGENT_CMD='gemini -y -s -p \"$(cat {prompt})\"' ralphloop plan \"design a vault manager\"")
    log.Println("  # Claude Code (Anthropic)")
    log.Println("  AGENT_CMD='claude -p \"$(cat {prompt})\" --dangerously-skip-permissions' ralphloop run")
    log.Println("  # OpenCode")
    log.Println("  AGENT_CMD='opencode run \"$(cat {prompt})\"' ralphloop plan \"...\"")
    log.Println("  # Droid")
    log.Println("  AGENT_CMD='droid exec --skip-permissions-unsafe -f {prompt}' ralphloop run")
    log.Println("  # GitHub Copilot")
    log.Println("  AGENT_CMD='copilot --allow-all-tools -p \"$(cat {prompt})\"' ralphloop run")
    log.Println("  ralphloop run")
}

func Run() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    subcommand := os.Args[1]
    var maxIter *int
    var verbose *bool
    var goal string

    planCmd := flag.NewFlagSet("plan", flag.ExitOnError)
    maxIterP := planCmd.Int("max", defaultMaxIterations, "Maximum number of iterations")
    verboseP := planCmd.Bool("v", false, "Verbose mode")

    runCmd := flag.NewFlagSet("run", flag.ExitOnError)
    maxIterR := runCmd.Int("max", defaultMaxIterations, "Maximum number of iterations")
    verboseR := runCmd.Bool("v", false, "Verbose mode")

    switch subcommand {
    case "plan":
        if err := planCmd.Parse(os.Args[2:]); err != nil {
            log.Fatalf("Failed to parse plan flags: %v", err)
        }
        maxIter = maxIterP
        verbose = verboseP
        goal = strings.Join(planCmd.Args(), " ")
        if goal == "" {
            log.Println("Error: 'plan' requires a goal.")
            planCmd.Usage()
            os.Exit(1)
        }
    case "run":
        if err := runCmd.Parse(os.Args[2:]); err != nil {
            log.Fatalf("Failed to parse run flags: %v", err)
        }
        maxIter = maxIterR
        verbose = verboseR
        goal = strings.Join(runCmd.Args(), " ")
    default:
        log.Printf("Unknown subcommand: %s", subcommand)
        printUsage()
        os.Exit(1)
    }

    agentCmd := os.Getenv("AGENT_CMD")
    if agentCmd == "" {
        agentCmd = "gemini -y -s -p \"$(cat {prompt})\""
        log.Println("Warning: AGENT_CMD not set. Using default: gemini (streaming enabled)")
    }

    log.Printf("Subcommand: %s", subcommand)
    if goal != "" {
        log.Printf("Goal: %s", goal)
    }

    if err := os.MkdirAll(ralphDir, 0755); err != nil {
        log.Fatalf("Failed to create %s directory: %v", ralphDir, err)
    }
    if err := os.MkdirAll(runsDir, 0755); err != nil {
        log.Fatalf("Failed to create %s directory: %v", runsDir, err)
    }

    // Initialize files if they don't exist
    initFiles := []string{progressFile, guardrailsFile, activityFile, errorsFile}
    for _, f := range initFiles {
        if _, err := os.Stat(f); os.IsNotExist(err) {
            if err := os.WriteFile(f, []byte(""), 0644); err != nil {
                log.Printf("Warning: Failed to create %s: %v", f, err)
            }
        }
    }

    scanner := bufio.NewScanner(os.Stdin)
    var userFeedback string
    planningComplete := false

    if _, err := os.Stat(ralphDir + "/PLAN_APPROVED"); err == nil {
        planningComplete = true
    }

    if subcommand == "plan" && planningComplete {
        log.Println("--- Existing plan found. Re-entering Planning Phase to add features/refine... ---")
        if err := os.Remove(ralphDir + "/PLAN_APPROVED"); err != nil {
            log.Printf("Warning: Failed to remove approval file: %v", err)
        }
        planningComplete = false
    }

    if subcommand == "run" && !planningComplete {
        log.Println("Error: Plan not approved yet. Run 'ralphloop plan' first and type 'approve'.")
        os.Exit(1)
    }

    for i := 1; i <= *maxIter; i++ {
        log.Printf("==== Iteration %d/%d ====", i, *maxIter)

        phase := "PLANNING PHASE: You are in DESIGN mode. ONLY modify .ralph/ files. DO NOT write code or install dependencies. Once the plan is ready, STOP and wait for approval."
        if subcommand == "run" {
            phase = "EXECUTION PHASE: Implement the approved plan story by story. Run verification scripts."
        }

        systemPrompt, err := promptsFS.ReadFile("prompts/system.md")
        if err != nil {
            log.Fatalf("Failed to read embedded system prompt: %v", err)
        }

        fullPrompt := fmt.Sprintf("# SYSTEM PROMPT\n%s\n\n# CURRENT PHASE\n%s\n\n# USER GOAL\n%s\n\n", string(systemPrompt), phase, goal)
        if userFeedback != "" {
            fullPrompt += fmt.Sprintf("# USER FEEDBACK / ANSWERS\n%s\n\n", userFeedback)
            userFeedback = ""
        }

        fullPrompt += "# CURRENT WORKSPACE\n"
        files, _ := os.ReadDir(".")
        var fileList []string
        for _, f := range files {
            name := f.Name()
            if strings.HasPrefix(name, ".") && name != ".ralph" {
                continue
            }
            if f.IsDir() {
                fileList = append(fileList, name+"/")
            } else {
                fileList = append(fileList, name)
            }
        }
        fullPrompt += "Files: " + strings.Join(fileList, ", ") + "\n\n"
        fullPrompt += "# PROJECT PROGRESS\n"

        if prdData, err := os.ReadFile(ralphDir + "/prd.json"); err == nil {
            fullPrompt += "## Current PRD (JSON):\n" + string(prdData) + "\n\n"
        } else if prdData, err := os.ReadFile(ralphDir + "/prd.md"); err == nil {
            fullPrompt += "## Current PRD (Markdown):\n" + string(prdData) + "\n\n"
        } else {
            fullPrompt += "No PRD generated yet.\n\n"
        }

        if qaData, err := os.ReadFile(ralphDir + "/qa-plan.md"); err == nil {
            fullPrompt += "## Current QA Plan:\n" + string(qaData) + "\n\n"
        } else {
            fullPrompt += "No QA Plan generated yet.\n\n"
        }

        if progData, err := os.ReadFile(progressFile); err == nil && len(progData) > 0 {
            fullPrompt += "## Progress Log:\n" + string(progData) + "\n\n"
        }

        if guardData, err := os.ReadFile(guardrailsFile); err == nil && len(guardData) > 0 {
            fullPrompt += "## Guardrails (Lessons Learned):\n" + string(guardData) + "\n\n"
        }

        if errData, err := os.ReadFile(errorsFile); err == nil && len(errData) > 0 {
            fullPrompt += "## Error Notes:\n" + string(errData) + "\n\n"
        }

        if *verbose {
            log.Printf("[DEBUG] Full Prompt written to %s:\n%s", promptFile, fullPrompt)
        }

        if err := os.WriteFile(promptFile, []byte(fullPrompt), 0644); err != nil {
            log.Printf("Error: Failed to write prompt file: %v", err)
        }

        startTime := time.Now()
        output, err := runAgent(agentCmd, promptFile, *verbose)
        duration := time.Since(startTime)

        if err != nil {
            log.Printf("Agent Error: %v", err)
        }

        // Activity and Raw Logs
        timestamp := time.Now().Format("20060102-150405")
        activityMsg := fmt.Sprintf("[%s] Iteration %d | Phase: %s | Time: %v\n", time.Now().Format("15:04:05"), i, subcommand, duration)
        af, err := os.OpenFile(activityFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
        if err == nil {
            if _, err := af.WriteString(activityMsg); err != nil {
                log.Printf("Warning: Failed to write to activity file: %v", err)
            }
            if err := af.Close(); err != nil {
                log.Printf("Warning: Failed to close activity file: %v", err)
            }
        }

        runLog := fmt.Sprintf("%s/run-%s-iter-%02d.log", runsDir, timestamp, i)
        if err := os.WriteFile(runLog, []byte(output), 0644); err != nil {
            log.Printf("Warning: Failed to write run log: %v", err)
        }

        if strings.Contains(strings.ToUpper(output), "DONE") {
            log.Println("[✔] Agent signaled completion (DONE).")
            break
        }

        if i < *maxIter {
            if subcommand == "plan" && !planningComplete {
                fmt.Print("\n[?] Plan looks good? Type 'approve' to save and exit, or provide feedback: ")
            } else {
                fmt.Print("\n[?] Provide feedback/answers (or press Enter to continue): ")
            }

            if scanner.Scan() {
                input := scanner.Text()
                if subcommand == "plan" && !planningComplete && strings.ToLower(strings.TrimSpace(input)) == "approve" {
                    if err := os.WriteFile(ralphDir+"/PLAN_APPROVED", []byte("approved"), 0644); err != nil {
                        log.Fatalf("Failed to write approval file: %v", err)
                    }
                    log.Println("--- Plan Approved! You can now run 'ralphloop run' to start execution. ---")
                    break
                } else {
                    userFeedback = input
                }
            }
        }

        if i == *maxIter {
            log.Println("[!] Reached maximum iterations.")
        }
    }
    if err := os.Remove(promptFile); err != nil {
        log.Printf("Warning: Failed to remove prompt file: %v", err)
    }
    log.Println("Finished.")
}
