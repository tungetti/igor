# IGOR PROJECT DOCUMENTATION

> **Single Source of Truth for Project Management and Session Persistence**

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Development Pipeline](#2-development-pipeline)
3. [Phase Breakdown](#3-phase-breakdown)
4. [Sprint Details](#4-sprint-details)
5. [Activity Log](#5-activity-log)
6. [Git Workflow](#6-git-workflow)
7. [Appendices](#7-appendices)
8. [Glossary](#8-glossary)
9. [Continuation Prompt](#9-continuation-prompt)

---

## 1. Project Overview

### 1.1 Executive Summary

**Igor** is a Go-based Terminal User Interface (TUI) application designed to simplify the installation of NVIDIA drivers and CUDA toolkit on Linux systems. Built with the Bubble Tea framework, it replaces the existing Ubuntu-only bash script with a cross-platform solution supporting all major Linux distribution families.

### 1.2 Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.21+ |
| TUI Framework | github.com/charmbracelet/bubbletea v0.25.0 |
| UI Components | github.com/charmbracelet/bubbles v0.18.0 |
| Styling | github.com/charmbracelet/lipgloss v0.9.1 |
| Logging | github.com/charmbracelet/log v0.3.1 |
| Testing | github.com/stretchr/testify v1.8.4 |
| Config | gopkg.in/yaml.v3 v3.0.1 |

### 1.3 Supported Distributions

| Family | Distributions | Package Manager |
|--------|---------------|-----------------|
| **Debian** | Ubuntu, Debian, Linux Mint, Pop!_OS | APT |
| **RHEL** | Fedora, CentOS, RHEL, Rocky, AlmaLinux | DNF/YUM |
| **Arch** | Arch Linux, Manjaro, EndeavourOS | Pacman |
| **SUSE** | openSUSE Leap, openSUSE Tumbleweed | Zypper |

### 1.4 Project Structure (Current State after Phase 2)

```
igor/
├── cmd/igor/                    # Main entry point [IMPLEMENTED]
│   ├── main.go                  # Entry point with version
│   ├── cli.go                   # CLI integration
│   └── version.go               # Build version variables
├── internal/
│   ├── app/                     # Application lifecycle [P1-MS9]
│   │   ├── app.go               # Application struct
│   │   ├── container.go         # Dependency injection
│   │   └── lifecycle.go         # Signal handling, shutdown
│   ├── cli/                     # CLI parsing [P1-MS6]
│   │   ├── parser.go            # Argument parser
│   │   ├── commands.go          # Subcommands
│   │   └── flags.go             # Flag definitions
│   ├── config/                  # Configuration [P1-MS5]
│   │   ├── config.go            # Config struct
│   │   ├── loader.go            # YAML/env loader
│   │   ├── validator.go         # Validation
│   │   └── defaults.go          # Default values
│   ├── constants/               # Constants [P1-MS3]
│   │   └── constants.go         # DistroFamily, exit codes
│   ├── distro/                  # Distribution detection [P2-MS2]
│   │   ├── detector.go          # Main detector
│   │   ├── osrelease.go         # /etc/os-release parser
│   │   └── types.go             # Distribution struct
│   ├── errors/                  # Custom errors [P1-MS3]
│   │   └── errors.go            # Error types with codes
│   ├── exec/                    # Command execution [P1-MS8]
│   │   ├── executor.go          # Executor interface
│   │   ├── mock.go              # MockExecutor for tests
│   │   └── output.go            # Result struct
│   ├── logging/                 # Logging [P1-MS4]
│   │   ├── logger.go            # Logger interface
│   │   └── levels.go            # Log levels
│   ├── pkg/                     # Package managers [P2-MS1-MS9]
│   │   ├── manager.go           # Manager interface
│   │   ├── types.go             # Package, Repository structs
│   │   ├── errors.go            # Package-specific errors
│   │   ├── factory/             # Factory pattern [P2-MS8]
│   │   │   └── factory.go
│   │   ├── apt/                 # APT [P2-MS3]
│   │   │   ├── apt.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── dnf/                 # DNF [P2-MS4]
│   │   │   ├── dnf.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── yum/                 # YUM [P2-MS5]
│   │   │   ├── yum.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── pacman/              # Pacman [P2-MS6]
│   │   │   ├── pacman.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── zypper/              # Zypper [P2-MS7]
│   │   │   ├── zypper.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   └── nvidia/              # NVIDIA packages [P2-MS9]
│   │       ├── packages.go
│   │       └── repository.go
│   ├── privilege/               # Root privileges [P1-MS7]
│   │   ├── privilege.go         # Manager interface
│   │   ├── sudo.go              # sudo support
│   │   └── pkexec.go            # pkexec support
│   ├── gpu/                     # GPU detection [Phase 3 - PENDING]
│   ├── install/                 # Installation [Phase 5 - PENDING]
│   ├── recovery/                # Recovery [Phase 6 - PENDING]
│   ├── testing/                 # Test utilities [Phase 7 - PENDING]
│   ├── ui/                      # TUI [Phase 4 - PENDING]
│   └── uninstall/               # Uninstall [Phase 6 - PENDING]
├── pkg/                         # Public packages (empty)
├── scripts/
│   ├── build.sh
│   └── test.sh
├── go.mod                       # Go module
├── go.sum
├── Makefile
├── VERSION                      # Current: 2.9.0
├── CHANGELOG.md
├── README.md
├── LICENSE                      # MIT
├── IGOR_PROJECT.md              # This file (Single Source of Truth)
└── install_nvidia.sh            # Original bash script (reference)
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   └── nvidia/
│   │       ├── packages.go
│   │       ├── debian.go
│   │       ├── rhel.go
│   │       ├── arch.go
│   │       └── suse.go
│   ├── privilege/               # Root privilege handling
│   │   ├── privilege.go
```

### 1.5 Current Project Metrics (as of v6.5.0)

| Metric | Value |
|--------|-------|
| **Version** | 6.5.0 |
| **Total Go Files** | ~130 (source + test) |
| **Lines of Code** | ~65,000 |
| **Test Coverage** | 90%+ average |
| **Commits** | 54 |
| **Tags** | 52 (v1.1.0 - v6.5.0) |
| **Phases Complete** | 5 of 7 |
| **Sprints Complete** | 52 of 62 (84%) |

### 1.6 Package Test Coverage

| Package | Coverage |
|---------|----------|
| internal/constants | 100% |
| internal/errors | 100% |
| internal/pkg (interface) | 100% |
| internal/logging | 98.9% |
| internal/exec | 98.4% |
| internal/cli | 97% |
| internal/pkg/factory | 97% |
| internal/pkg/nvidia | 97.9% |
| internal/distro | 96% |
| internal/pkg/apt | 94% |
| internal/pkg/dnf | 93.8% |
| internal/app | 93% |
| internal/privilege | 91% |
| internal/pkg/zypper | 90.7% |
| internal/pkg/pacman | 90.6% |
| internal/config | 89.4% |
| internal/pkg/yum | 87% |

### 1.5 Key Features

1. **Dynamic Package Discovery** - Queries distribution repositories for available NVIDIA driver versions
2. **Multi-GPU Support** - Detects and handles multiple NVIDIA GPUs
3. **Intelligent Recommendations** - Suggests optimal driver based on GPU architecture
4. **Rollback on Failure** - Automatically undoes partial installations
5. **Dry-Run Mode** - Preview changes without executing
6. **Recovery Mode** - Works from TTY when X.org fails to start
7. **Single Binary** - Static Go compilation for easy deployment

### 1.6 TUI Flow

```
Welcome Screen
    │
    ▼
System Detection (auto)
    │
    ▼
GPU Info Display
    │
    ▼
Driver Version Selection (list)
    │
    ▼
CUDA Toolkit Selection (optional)
    │
    ▼
Installation Confirmation
    │
    ▼
Installation Progress
    │
    ▼
Completion / Reboot Prompt
```

---

## 2. Development Pipeline

### 2.1 Pipeline Overview

The development follows a strict phase-by-phase, sprint-by-sprint approach with mandatory code review cycles.

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         DEVELOPMENT PIPELINE                             │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│   ┌──────────────┐                                                       │
│   │ START PHASE  │                                                       │
│   └──────┬───────┘                                                       │
│          ▼                                                               │
│   ┌──────────────────┐                                                   │
│   │  SELECT SPRINT   │◄─────────────────────────────────────┐            │
│   └────────┬─────────┘                                      │            │
│            ▼                                                │            │
│   ┌──────────────────────────┐                              │            │
│   │  CODE IMPLEMENTATION     │                              │            │
│   │  (code-implementator)    │                              │            │
│   └────────┬─────────────────┘                              │            │
│            ▼                                                │            │
│   ┌──────────────────────────┐                              │            │
│   │  CODE REVIEW             │                              │            │
│   │  (code-reviewer)         │                              │            │
│   └────────┬─────────────────┘                              │            │
│            ▼                                                │            │
│   ┌──────────────────────────┐    ┌───────────────┐         │            │
│   │  REVIEW PASSED?          │─NO─►│ INCREMENT z   │─────────┘            │
│   └────────┬─────────────────┘    │ FIX ISSUES    │                      │
│            │ YES                   └───────────────┘                      │
│            ▼                                                              │
│   ┌──────────────────────────┐                                           │
│   │  RUN TESTS               │                                           │
│   │  go test ./...           │                                           │
│   │  go build ./...          │                                           │
│   └────────┬─────────────────┘                                           │
│            ▼                                                              │
│   ┌──────────────────────────┐    ┌───────────────┐                      │
│   │  TESTS PASSED?           │─NO─►│ INCREMENT z   │──────────┘           │
│   └────────┬─────────────────┘    │ FIX ISSUES    │                      │
│            │ YES                   └───────────────┘                      │
│            ▼                                                              │
│   ┌──────────────────────────┐                                           │
│   │  HUMAN VALIDATION        │                                           │
│   │  (WAIT FOR APPROVAL)     │                                           │
│   └────────┬─────────────────┘                                           │
│            │ APPROVED                                                     │
│            ▼                                                              │
│   ┌──────────────────────────┐                                           │
│   │  UPDATE DOCUMENTATION    │                                           │
│   │  (documentation-manager) │                                           │
│   └────────┬─────────────────┘                                           │
│            ▼                                                              │
│   ┌──────────────────────────┐                                           │
│   │  UPDATE VERSION          │                                           │
│   │  VERSION file + tag      │                                           │
│   └────────┬─────────────────┘                                           │
│            ▼                                                              │
│   ┌──────────────────────────┐                                           │
│   │  GIT COMMIT & PUSH       │                                           │
│   │  to GitHub remote        │                                           │
│   └────────┬─────────────────┘                                           │
│            ▼                                                              │
│   ┌──────────────────────────┐    ┌───────────────────┐                  │
│   │  MORE SPRINTS IN PHASE?  │─YES─►│ NEXT SPRINT      │──────┐           │
│   └────────┬─────────────────┘    │ y++, z=0          │      │           │
│            │ NO                    └───────────────────┘      │           │
│            ▼                                                   │           │
│   ┌──────────────────────────┐                                │           │
│   │  MORE PHASES?            │─YES─► x++, y=1, z=0 ───────────┘           │
│   └────────┬─────────────────┘                                            │
│            │ NO                                                           │
│            ▼                                                              │
│   ┌──────────────────────────┐                                           │
│   │  PROJECT COMPLETE        │                                           │
│   │  Final release v7.6.x    │                                           │
│   └──────────────────────────┘                                           │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Version Management

**Version Format: `x.y.z`**

| Component | Meaning | Increment Rule |
|-----------|---------|----------------|
| **x** | Phase number | Increments when starting a new phase |
| **y** | Sprint number within phase | Increments when moving to next sprint |
| **z** | Iteration number | Increments each code-review cycle within a sprint |

**Version Examples:**

| Version | Meaning |
|---------|---------|
| `1.1.0` | Phase 1, Sprint 1, initial implementation |
| `1.1.1` | Phase 1, Sprint 1, first revision after code review |
| `1.1.2` | Phase 1, Sprint 1, second revision after code review |
| `1.2.0` | Phase 1, Sprint 2, initial implementation (after 1.1.x approved) |
| `2.1.0` | Phase 2, Sprint 1, initial implementation (after Phase 1 complete) |
| `7.6.0` | Phase 7, Sprint 6, initial implementation (final phase) |

### 2.3 Sprint Execution Protocol

For **EACH** sprint, follow these steps **IN ORDER**:

#### Step 1: PRE-SPRINT PREPARATION

```
1. Read IGOR_PROJECT.md to understand context
2. Read the specific sprint requirements
3. Check sprint dependencies are completed
4. Update sprint status to `IN_PROGRESS` in Activity Log
```

#### Step 2: CODE IMPLEMENTATION

```
1. Delegate to `code-implementator` agent with sprint specs
2. Implement ALL files listed in sprint
3. Follow ALL acceptance criteria
4. Create unit tests for new code
5. Initial version: x.y.0
```

#### Step 3: CODE REVIEW

```
1. Delegate to `code-reviewer` agent
2. Review criteria:
   - Code quality and Go best practices
   - Error handling completeness
   - Test coverage (target >80%)
   - Documentation (godoc comments)
   - Security considerations
3. If issues found:
   - Increment z (e.g., 1.1.0 → 1.1.1)
   - Fix issues
   - Return to Step 3
```

#### Step 4: TESTING

```
1. Run: go test ./...
2. Run: go build ./...
3. Run: go vet ./...
4. Check for race conditions: go test -race ./...
5. If failures:
   - Increment z
   - Fix issues
   - Return to Step 3
```

#### Step 5: HUMAN VALIDATION

```
1. Present completed work summary to user
2. Show test results and code changes
3. WAIT for explicit approval
4. Address any feedback
5. If changes requested:
   - Increment z
   - Implement changes
   - Return to Step 3
```

#### Step 6: POST-SPRINT FINALIZATION

```
1. Update sprint status to `COMPLETED` in Activity Log
2. Update VERSION file with final version
3. Update CHANGELOG.md with changes
4. Git operations:
   - git add -A
   - git commit -m "[Phase X Sprint Y] Description"
   - git tag vX.Y.Z
   - git push origin main --tags
5. Proceed to next sprint
```

### 2.4 Agent Responsibilities

| Agent | When Used | Responsibility |
|-------|-----------|----------------|
| `code-implementator` | Step 2 | Write code according to sprint specifications |
| `code-reviewer` | Step 3 | Review code quality, suggest improvements |
| `documentation-manager` | Step 6 | Update documentation after sprint completion |
| `security-optimizer` | Steps 2-3 | Review security-sensitive code (privilege, exec) |

### 2.5 Quality Gates

Each sprint must pass these gates before approval:

| Gate | Criteria | Required |
|------|----------|----------|
| **Build** | `go build ./...` succeeds | YES |
| **Tests** | `go test ./...` passes | YES |
| **Coverage** | >80% for new code | YES |
| **Vet** | `go vet ./...` clean | YES |
| **Review** | Code reviewer approves | YES |
| **Human** | User validation received | YES |

---

## 3. Phase Breakdown

### Phase Summary

| Phase | Name | Sprints | Status | Version Range |
|-------|------|---------|--------|---------------|
| 1 | Project Foundation & Architecture | 9 | `NOT_STARTED` | 1.1.0 - 1.9.x |
| 2 | Distribution Detection & Package Manager Abstraction | 9 | `NOT_STARTED` | 2.1.0 - 2.9.x |
| 3 | GPU Detection & System Analysis | 7 | `NOT_STARTED` | 3.1.0 - 3.7.x |
| 4 | TUI Framework & Core Views | 11 | `NOT_STARTED` | 4.1.0 - 4.11.x |
| 5 | Installation Workflow Engine | 11 | `NOT_STARTED` | 5.1.0 - 5.11.x |
| 6 | Uninstallation & Recovery | 9 | `NOT_STARTED` | 6.1.0 - 6.9.x |
| 7 | Testing, Validation & Documentation | 6 | `NOT_STARTED` | 7.1.0 - 7.6.x |

**Total: 62 Sprints**

### Phase 1: Project Foundation & Architecture

**Status:** `COMPLETED`
**Objective:** Establish the Go project structure, core interfaces, and build infrastructure.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P1-MS1 | Initialize Go Module and Project Skeleton | `COMPLETED` | 1.1.0 | None | Small |
| P1-MS2 | Add Core Dependencies | `COMPLETED` | 1.2.0 | P1-MS1 | Small |
| P1-MS3 | Create Error Types and Constants | `COMPLETED` | 1.3.0 | P1-MS1 | Small |
| P1-MS4 | Implement Logging Infrastructure | `COMPLETED` | 1.4.0 | P1-MS2, P1-MS3 | Medium |
| P1-MS5 | Create Configuration Management System | `COMPLETED` | 1.5.0 | P1-MS3, P1-MS4 | Medium |
| P1-MS6 | Implement CLI Argument Parser | `COMPLETED` | 1.6.0 | P1-MS5 | Medium |
| P1-MS7 | Create Root Privilege Handler | `COMPLETED` | 1.7.0 | P1-MS3, P1-MS4 | Medium |
| P1-MS8 | Implement Command Executor Interface | `COMPLETED` | 1.8.0 | P1-MS7 | Medium |
| P1-MS9 | Create Application Bootstrap and Lifecycle | `COMPLETED` | 1.9.0 | P1-MS4, P1-MS5, P1-MS6, P1-MS8 | Medium |

### Phase 2: Distribution Detection & Package Manager Abstraction

**Status:** `COMPLETED`
**Objective:** Create robust distribution detection and package manager abstraction for all target distributions.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P2-MS1 | Define Package Manager Interface | `COMPLETED` | 2.1.0 | P1-MS8 | Small |
| P2-MS2 | Implement Distribution Detection Core | `COMPLETED` | 2.2.0 | P1-MS8, P1-MS3 | Medium |
| P2-MS3 | Implement APT Package Manager (Debian/Ubuntu) | `COMPLETED` | 2.3.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS4 | Implement DNF Package Manager (Fedora/RHEL) | `COMPLETED` | 2.4.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS5 | Implement YUM Package Manager (CentOS 7/RHEL 7) | `COMPLETED` | 2.5.0 | P2-MS1, P2-MS2, P1-MS7 | Medium |
| P2-MS6 | Implement Pacman Package Manager (Arch) | `COMPLETED` | 2.6.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS7 | Implement Zypper Package Manager (openSUSE) | `COMPLETED` | 2.7.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS8 | Create Package Manager Factory | `COMPLETED` | 2.8.0 | P2-MS2 through P2-MS7 | Small |
| P2-MS9 | Create Distribution-Specific NVIDIA Package Mappings | `COMPLETED` | 2.9.0 | P2-MS8 | Medium |

### Phase 3: GPU Detection & System Analysis

**Status:** `COMPLETED`
**Objective:** Implement comprehensive GPU detection, driver status checking, and system compatibility analysis.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P3-MS1 | Implement PCI Device Scanner | `COMPLETED` | 3.1.0 | P1-MS8 | Medium |
| P3-MS2 | Create NVIDIA GPU Database | `COMPLETED` | 3.2.0 | P3-MS1 | Medium |
| P3-MS3 | Implement nvidia-smi Parser | `COMPLETED` | 3.3.0 | P1-MS8, P3-MS2 | Medium |
| P3-MS4 | Create Nouveau Driver Detector | `COMPLETED` | 3.4.0 | P3-MS1, P1-MS8 | Small |
| P3-MS5 | Implement Kernel Version and Module Detection | `COMPLETED` | 3.5.0 | P1-MS8, P2-MS8 | Medium |
| P3-MS6 | Create System Requirements Validator | `COMPLETED` | 3.6.0 | P3-MS5, P2-MS8 | Medium |
| P3-MS7 | Build GPU Detection Orchestrator | `COMPLETED` | 3.7.0 | P3-MS1 through P3-MS6 | Medium |

### Phase 4: TUI Framework & Core Views

**Status:** `NOT_STARTED`
**Objective:** Build the Bubble Tea-based TUI application with all core views and navigation.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P4-MS1 | Create Base TUI Application Structure | `COMPLETED` | 4.1.0 | P1-MS2, P1-MS9 | Medium |
| P4-MS2 | Implement Theme and Styling System | `COMPLETED` | 4.2.0 | P1-MS2 | Medium |
| P4-MS3 | Create Common UI Components | `COMPLETED` | 4.3.0 | P4-MS2 | Medium |
| P4-MS4 | Implement Welcome Screen View | `COMPLETED` | 4.4.0 | P4-MS1, P4-MS2, P4-MS3 | Small |
| P4-MS5 | Implement System Detection View | `COMPLETED` | 4.5.0 | P4-MS1, P4-MS3, P3-MS7 | Medium |
| P4-MS6 | Implement Driver Selection View | `COMPLETED` | 4.6.0 | P4-MS1, P4-MS3, P3-MS7, P2-MS9 | Large |
| P4-MS7 | Implement Installation Confirmation View | `COMPLETED` | 4.7.0 | P4-MS1, P4-MS3, P4-MS6 | Medium |
| P4-MS8 | Implement Installation Progress View | `COMPLETED` | 4.8.0 | P4-MS1, P4-MS3 | Large |
| P4-MS9 | Implement Completion/Result View | `COMPLETED` | 4.9.0 | P4-MS1, P4-MS3 | Small |
| P4-MS10 | Implement Error/Help View | `COMPLETED` | 4.10.0 | P4-MS1, P4-MS3 | Small |
| P4-MS11 | Implement View Navigation and State Machine | `COMPLETED` | 4.11.0 | P4-MS4 through P4-MS10 | Medium |

### Phase 5: Installation Workflow Engine

**Status:** `COMPLETED`
**Objective:** Implement core installation orchestration with steps, rollback, and progress tracking.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P5-MS1 | Define Installation Workflow Interface | `COMPLETED` | 5.1.0 | P2-MS8, P3-MS7 | Medium |
| P5-MS2 | Implement Pre-Installation Validation Step | `COMPLETED` | 5.2.0 | P5-MS1, P3-MS6 | Small |
| P5-MS3 | Implement Repository Configuration Step | `COMPLETED` | 5.3.0 | P5-MS1, P2-MS3 through P2-MS7 | Medium |
| P5-MS4 | Implement Nouveau Blacklist Step | `COMPLETED` | 5.4.0 | P5-MS1, P3-MS4 | Small |
| P5-MS5 | Implement Package Installation Step | `COMPLETED` | 5.5.0 | P5-MS1, P2-MS8, P2-MS9 | Large |
| P5-MS6 | Implement DKMS Module Build Step | `COMPLETED` | 5.6.0 | P5-MS1, P3-MS5 | Medium |
| P5-MS7 | Implement Module Loading Step | `COMPLETED` | 5.7.0 | P5-MS1, P3-MS5 | Small |
| P5-MS8 | Implement X.org Configuration Step | `COMPLETED` | 5.8.0 | P5-MS1 | Medium |
| P5-MS9 | Implement Post-Installation Verification Step | `COMPLETED` | 5.9.0 | P5-MS1, P3-MS3 | Small |
| P5-MS10 | Implement Workflow Orchestrator | `COMPLETED` | 5.10.0 | P5-MS2 through P5-MS9 | Large |
| P5-MS11 | Create Distribution-Specific Workflow Builders | `COMPLETED` | 5.11.0 | P5-MS10, P2-MS2 | Medium |

### Phase 6: Uninstallation & Recovery

**Status:** `IN_PROGRESS`
**Objective:** Implement safe uninstallation and recovery mechanisms.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P6-MS1 | Implement Uninstallation Workflow Interface | `COMPLETED` | 6.1.0 | P5-MS1 | Small |
| P6-MS2 | Implement Installed Package Discovery | `COMPLETED` | 6.2.0 | P6-MS1, P2-MS8 | Medium |
| P6-MS3 | Implement Package Removal Step | `COMPLETED` | 6.3.0 | P6-MS1, P6-MS2 | Medium |
| P6-MS4 | Implement Kernel Module Cleanup Step | `COMPLETED` | 6.4.0 | P6-MS1, P3-MS5 | Small |
| P6-MS5 | Implement Configuration Cleanup Step | `COMPLETED` | 6.5.0 | P6-MS1 | Small |
| P6-MS6 | Implement Fallback Driver Restoration | `NOT_STARTED` | 6.6.0 | P6-MS4, P6-MS5 | Small |
| P6-MS7 | Implement Uninstall TUI Views | `NOT_STARTED` | 6.7.0 | P6-MS1, P4-MS7, P4-MS8, P4-MS9 | Medium |
| P6-MS8 | Create Uninstall Orchestrator | `NOT_STARTED` | 6.8.0 | P6-MS2 through P6-MS6 | Medium |
| P6-MS9 | Implement System Recovery Mode | `NOT_STARTED` | 6.9.0 | P6-MS8, P1-MS6 | Medium |

### Phase 7: Testing, Validation & Documentation

**Status:** `NOT_STARTED`
**Objective:** Comprehensive testing and documentation.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P7-MS1 | Create Unit Test Infrastructure | `NOT_STARTED` | 7.1.0 | P1-MS2 | Medium |
| P7-MS2 | Implement Distribution Detection Tests | `NOT_STARTED` | 7.2.0 | P7-MS1, P2-MS2 | Medium |
| P7-MS3 | Implement Package Manager Tests | `NOT_STARTED` | 7.3.0 | P7-MS1, P2-MS3 through P2-MS7 | Large |
| P7-MS4 | Implement GPU Detection Tests | `NOT_STARTED` | 7.4.0 | P7-MS1, P3-MS1, P3-MS3, P3-MS7 | Medium |
| P7-MS5 | Implement TUI Component Tests | `NOT_STARTED` | 7.5.0 | P7-MS1, P4-MS1 through P4-MS6 | Medium |
| P7-MS6 | Implement Installation Workflow Tests | `NOT_STARTED` | 7.6.0 | P7-MS1, P5-MS10 | Large |

---

## 4. Sprint Details

### Sprint Template

Each sprint entry follows this template:

```markdown
#### P{X}-MS{Y}: {Title}

**Status:** `NOT_STARTED` | `IN_PROGRESS` | `COMPLETED` | `BLOCKED`
**Version:** X.Y.Z
**Effort:** Small | Medium | Large
**Dependencies:** List of prerequisite sprints

**Description:**
Brief description of what this sprint accomplishes.

**Files to Create:**
- List of files to create

**Files to Modify:**
- List of existing files to modify

**Acceptance Criteria:**
- [ ] Criterion 1
- [ ] Criterion 2
- [ ] ...

**Implementation Notes:**
Technical guidance and code examples.

**Iteration History:**
| Version | Date | Changes | Reviewer Notes |
|---------|------|---------|----------------|
| x.y.0 | - | Initial | - |
```

---

### Phase 1 Sprint Details

#### P1-MS1: Initialize Go Module and Project Skeleton

**Status:** `COMPLETED`
**Version:** 1.1.0
**Effort:** Small
**Dependencies:** None

**Description:**
Initialize the Go module, create the complete directory structure, and set up the basic project skeleton with placeholder files.

**Files to Create:**
```
igor/
├── go.mod
├── .gitignore
├── README.md
├── LICENSE (MIT)
├── Makefile
├── VERSION (contains "1.1.0")
├── CHANGELOG.md
├── cmd/igor/main.go
├── internal/app/.gitkeep
├── internal/cli/.gitkeep
├── internal/config/.gitkeep
├── internal/constants/.gitkeep
├── internal/distro/.gitkeep
├── internal/errors/.gitkeep
├── internal/exec/.gitkeep
├── internal/gpu/.gitkeep
├── internal/install/.gitkeep
├── internal/logging/.gitkeep
├── internal/pkg/.gitkeep
├── internal/privilege/.gitkeep
├── internal/recovery/.gitkeep
├── internal/testing/.gitkeep
├── internal/ui/.gitkeep
├── internal/uninstall/.gitkeep
├── pkg/.gitkeep
├── scripts/build.sh
└── scripts/test.sh
```

**Acceptance Criteria:**
- [ ] `go mod init github.com/tommasomariaungetti/igor` succeeds
- [ ] All directories exist as specified
- [ ] `go build ./cmd/igor` produces binary
- [ ] `make build` works
- [ ] `make test` works (even if no tests yet)
- [ ] `.gitignore` excludes: `*.exe`, `igor`, `/vendor/`, `*.log`
- [ ] VERSION file contains `1.1.0`
- [ ] Running `./igor` prints version info

**Implementation Notes:**

```go
// cmd/igor/main.go
package main

import (
    "fmt"
    "os"
)

var (
    Version   = "1.1.0"
    BuildTime = "unknown"
    GitCommit = "unknown"
)

func main() {
    fmt.Printf("Igor - NVIDIA TUI Installer v%s\n", Version)
    fmt.Printf("Build: %s (%s)\n", BuildTime, GitCommit)
    os.Exit(0)
}
```

```makefile
# Makefile
.PHONY: build test clean

VERSION := $(shell cat VERSION)
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)

build:
	go build -ldflags "$(LDFLAGS)" -o igor ./cmd/igor

test:
	go test -v ./...

clean:
	rm -f igor
```

---

#### P1-MS2: Add Core Dependencies

**Status:** `NOT_STARTED`
**Version:** 1.2.0
**Effort:** Small
**Dependencies:** P1-MS1

**Description:**
Add all required external dependencies to go.mod.

**Files to Modify:**
- `go.mod`

**Files Created:**
- `go.sum` (auto-generated)

**Acceptance Criteria:**
- [ ] All dependencies added to go.mod
- [ ] `go mod tidy` produces no changes
- [ ] `go mod verify` passes
- [ ] Can import all packages without errors

**Implementation Notes:**

```bash
go get github.com/charmbracelet/bubbletea@v0.25.0
go get github.com/charmbracelet/bubbles@v0.18.0
go get github.com/charmbracelet/lipgloss@v0.9.1
go get github.com/charmbracelet/log@v0.3.1
go get github.com/stretchr/testify@v1.8.4
go get gopkg.in/yaml.v3@v3.0.1
go mod tidy
```

---

#### P1-MS3: Create Error Types and Constants

**Status:** `NOT_STARTED`
**Version:** 1.3.0
**Effort:** Small
**Dependencies:** P1-MS1

**Description:**
Define custom error types and application constants.

**Files to Create:**
```
internal/errors/errors.go
internal/errors/errors_test.go
internal/constants/constants.go
```

**Acceptance Criteria:**
- [ ] Error types implement `error` interface
- [ ] Errors support `Unwrap()` for chaining
- [ ] `errors.Is()` and `errors.As()` work
- [ ] Constants are typed (not untyped)
- [ ] Unit tests pass

**Implementation Notes:**

```go
// internal/errors/errors.go
package errors

import "fmt"

type Code int

const (
    Unknown Code = iota
    DistroDetection
    GPUDetection
    PackageManager
    Installation
    Permission
    Network
    Configuration
    Validation
)

type Error struct {
    Code    Code
    Message string
    Cause   error
}

func New(code Code, message string) *Error {
    return &Error{Code: code, Message: message}
}

func Wrap(code Code, message string, cause error) *Error {
    return &Error{Code: code, Message: message, Cause: cause}
}

func (e *Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %v", e.Message, e.Cause)
    }
    return e.Message
}

func (e *Error) Unwrap() error {
    return e.Cause
}
```

---

#### P1-MS4: Implement Logging Infrastructure

**Status:** `NOT_STARTED`
**Version:** 1.4.0
**Effort:** Medium
**Dependencies:** P1-MS2, P1-MS3

**Description:**
Create centralized logging with configurable levels and file output.

**Files to Create:**
```
internal/logging/logger.go
internal/logging/levels.go
internal/logging/logger_test.go
```

**Acceptance Criteria:**
- [ ] Supports Debug, Info, Warn, Error levels
- [ ] Can log to file without affecting TUI
- [ ] Structured logging with key-value pairs
- [ ] Logger interface for mocking
- [ ] Thread-safe logging

---

#### P1-MS5: Create Configuration Management System

**Status:** `NOT_STARTED`
**Version:** 1.5.0
**Effort:** Medium
**Dependencies:** P1-MS3, P1-MS4

**Description:**
Configuration loading from files and environment variables.

**Files to Create:**
```
internal/config/config.go
internal/config/loader.go
internal/config/validator.go
internal/config/defaults.go
internal/config/config_test.go
```

**Acceptance Criteria:**
- [ ] Loads YAML configuration files
- [ ] Environment variables override file settings
- [ ] Validates configuration
- [ ] Applies defaults for missing values
- [ ] XDG Base Directory compliance

---

#### P1-MS6: Implement CLI Argument Parser

**Status:** `NOT_STARTED`
**Version:** 1.6.0
**Effort:** Medium
**Dependencies:** P1-MS5

**Description:**
Command-line argument parsing with subcommands.

**Files to Create:**
```
internal/cli/parser.go
internal/cli/commands.go
internal/cli/flags.go
internal/cli/parser_test.go
cmd/igor/cli.go
cmd/igor/version.go
```

**Acceptance Criteria:**
- [ ] Subcommands: install, uninstall, detect, list, version
- [ ] Global flags: --verbose, --quiet, --config, --dry-run
- [ ] Help text for all commands
- [ ] Version displays build info

---

#### P1-MS7: Create Root Privilege Handler

**Status:** `NOT_STARTED`
**Version:** 1.7.0
**Effort:** Medium
**Dependencies:** P1-MS3, P1-MS4

**Description:**
Privilege elevation with sudo/pkexec.

**Files to Create:**
```
internal/privilege/privilege.go
internal/privilege/sudo.go
internal/privilege/pkexec.go
internal/privilege/privilege_test.go
```

**Acceptance Criteria:**
- [ ] Detects root user
- [ ] Executes commands with sudo
- [ ] Falls back to pkexec
- [ ] Sanitizes environment

---

#### P1-MS8: Implement Command Executor Interface

**Status:** `NOT_STARTED`
**Version:** 1.8.0
**Effort:** Medium
**Dependencies:** P1-MS7

**Description:**
Testable command execution abstraction.

**Files to Create:**
```
internal/exec/executor.go
internal/exec/mock.go
internal/exec/output.go
internal/exec/executor_test.go
```

**Acceptance Criteria:**
- [ ] Output capture (stdout, stderr)
- [ ] Timeout handling
- [ ] Exit code capture
- [ ] Mockable for testing
- [ ] Streaming output support

---

#### P1-MS9: Create Application Bootstrap and Lifecycle

**Status:** `NOT_STARTED`
**Version:** 1.9.0
**Effort:** Medium
**Dependencies:** P1-MS4, P1-MS5, P1-MS6, P1-MS8

**Description:**
Application initialization and shutdown.

**Files to Create:**
```
internal/app/app.go
internal/app/container.go
internal/app/lifecycle.go
internal/app/app_test.go
```

**Files to Modify:**
```
cmd/igor/main.go
```

**Acceptance Criteria:**
- [ ] Initializes all components
- [ ] Signal handling (SIGINT, SIGTERM)
- [ ] Graceful shutdown
- [ ] Panic recovery
- [ ] Dependency injection

---

### Phase 2 Sprint Specifications

#### P2-MS1: Define Package Manager Interface

**Status:** `COMPLETED`
**Version:** 2.1.0
**Effort:** Small
**Dependencies:** P1-MS8

**Description:**
Define the core package manager interface that all distribution-specific implementations will conform to. This interface abstracts package operations across APT, DNF, YUM, Pacman, and Zypper.

**Files to Create:**
```
internal/pkg/manager.go       # Package manager interface
internal/pkg/types.go         # Package-related types (Package, Repository, etc.)
internal/pkg/errors.go        # Package manager specific errors
internal/pkg/manager_test.go  # Interface tests
```

**Interface Design:**
```go
// Package represents a software package
type Package struct {
    Name        string
    Version     string
    Installed   bool
    Repository  string
    Description string
}

// Repository represents a package repository
type Repository struct {
    Name    string
    URL     string
    Enabled bool
    GPGKey  string
}

// Manager defines the package manager interface
type Manager interface {
    // Package operations
    Install(ctx context.Context, packages ...string) error
    Remove(ctx context.Context, packages ...string) error
    Update(ctx context.Context) error
    Upgrade(ctx context.Context, packages ...string) error
    
    // Query operations
    IsInstalled(ctx context.Context, pkg string) (bool, error)
    Search(ctx context.Context, query string) ([]Package, error)
    Info(ctx context.Context, pkg string) (*Package, error)
    ListInstalled(ctx context.Context) ([]Package, error)
    
    // Repository operations
    AddRepository(ctx context.Context, repo Repository) error
    RemoveRepository(ctx context.Context, name string) error
    ListRepositories(ctx context.Context) ([]Repository, error)
    
    // Utility
    Clean(ctx context.Context) error
    Name() string
}
```

**Acceptance Criteria:**
- [ ] Manager interface with all CRUD operations for packages
- [ ] Package struct with name, version, installed status
- [ ] Repository struct for repo management
- [ ] Custom error types (ErrPackageNotFound, ErrRepositoryExists, etc.)
- [ ] Context support for cancellation/timeout
- [ ] >90% test coverage on types and error handling

---

#### P2-MS2: Implement Distribution Detection Core

**Status:** `COMPLETED`
**Version:** 2.2.0
**Effort:** Medium
**Dependencies:** P1-MS8, P1-MS3

**Description:**
Detect the Linux distribution by parsing /etc/os-release, lsb_release, and fallback methods. Determine the distribution family (Debian, RHEL, Arch, SUSE).

**Files to Create:**
```
internal/distro/detector.go      # Distribution detection logic
internal/distro/types.go         # Distribution types and families
internal/distro/osrelease.go     # /etc/os-release parser
internal/distro/detector_test.go # Tests with mock filesystems
```

**Distribution Families:**
```go
type Family string

const (
    FamilyDebian Family = "debian"   // Ubuntu, Debian, Mint, Pop!_OS
    FamilyRHEL   Family = "rhel"     // Fedora, RHEL, CentOS, Rocky, Alma
    FamilyArch   Family = "arch"     // Arch, Manjaro, EndeavourOS
    FamilySUSE   Family = "suse"     // openSUSE Leap, Tumbleweed
    FamilyUnknown Family = "unknown"
)

type Distribution struct {
    ID          string  // e.g., "ubuntu", "fedora"
    Name        string  // e.g., "Ubuntu 24.04 LTS"
    Version     string  // e.g., "24.04"
    VersionID   string  // e.g., "24.04"
    Family      Family
    PrettyName  string
    CodeName    string  // e.g., "noble"
}
```

**Acceptance Criteria:**
- [ ] Parse /etc/os-release correctly
- [ ] Fallback to lsb_release if needed
- [ ] Correctly identify distribution family
- [ ] Support all target distributions (Ubuntu, Debian, Fedora, RHEL, CentOS, Arch, Manjaro, openSUSE)
- [ ] Mock filesystem for testing
- [ ] >90% test coverage

---

#### P2-MS3: Implement APT Package Manager (Debian/Ubuntu)

**Status:** `COMPLETED`
**Version:** 2.3.0
**Effort:** Large
**Dependencies:** P2-MS1, P2-MS2, P1-MS7

**Description:**
Implement the Manager interface for APT-based distributions (Debian, Ubuntu, Mint, Pop!_OS).

**Files to Create:**
```
internal/pkg/apt/apt.go          # APT implementation
internal/pkg/apt/apt_test.go     # APT tests
internal/pkg/apt/repository.go   # APT repository handling
```

**Acceptance Criteria:**
- [ ] Implement all Manager interface methods
- [ ] apt-get update, install, remove, upgrade
- [ ] dpkg-query for package status
- [ ] add-apt-repository for PPA support
- [ ] GPG key handling for repositories
- [ ] DEBIAN_FRONTEND=noninteractive for unattended operation
- [ ] >85% test coverage with mocked executor

---

#### P2-MS4: Implement DNF Package Manager (Fedora/RHEL 8+)

**Status:** `COMPLETED`
**Version:** 2.4.0
**Effort:** Large
**Dependencies:** P2-MS1, P2-MS2, P1-MS7

**Description:**
Implement the Manager interface for DNF-based distributions (Fedora, RHEL 8+, Rocky 8+, Alma 8+).

**Files to Create:**
```
internal/pkg/dnf/dnf.go          # DNF implementation
internal/pkg/dnf/dnf_test.go     # DNF tests
internal/pkg/dnf/repository.go   # DNF repository handling (RPM Fusion)
```

**Acceptance Criteria:**
- [ ] Implement all Manager interface methods
- [ ] dnf check-update, install, remove, upgrade
- [ ] rpm -q for package status
- [ ] dnf config-manager for repository management
- [ ] RPM Fusion repository support
- [ ] -y flag for non-interactive operation
- [ ] >85% test coverage

---

#### P2-MS5: Implement YUM Package Manager (CentOS 7/RHEL 7)

**Status:** `COMPLETED`
**Version:** 2.5.0
**Effort:** Medium
**Dependencies:** P2-MS1, P2-MS2, P1-MS7

**Description:**
Implement the Manager interface for YUM-based distributions (CentOS 7, RHEL 7).

**Files to Create:**
```
internal/pkg/yum/yum.go          # YUM implementation
internal/pkg/yum/yum_test.go     # YUM tests
internal/pkg/yum/repository.go   # YUM repository handling (EPEL)
```

**Acceptance Criteria:**
- [ ] Implement all Manager interface methods
- [ ] yum check-update, install, remove, upgrade
- [ ] rpm -q for package status
- [ ] yum-config-manager for repositories
- [ ] EPEL repository support
- [ ] -y flag for non-interactive
- [ ] >85% test coverage

---

#### P2-MS6: Implement Pacman Package Manager (Arch)

**Status:** `COMPLETED`
**Version:** 2.6.0
**Effort:** Large
**Dependencies:** P2-MS1, P2-MS2, P1-MS7

**Description:**
Implement the Manager interface for Pacman-based distributions (Arch, Manjaro, EndeavourOS).

**Files to Create:**
```
internal/pkg/pacman/pacman.go       # Pacman implementation
internal/pkg/pacman/pacman_test.go  # Pacman tests
internal/pkg/pacman/repository.go   # Pacman repository handling
```

**Acceptance Criteria:**
- [ ] Implement all Manager interface methods
- [ ] pacman -Syu, -S, -R, -Ss, -Q
- [ ] --noconfirm for non-interactive
- [ ] Repository configuration via pacman.conf
- [ ] Handle Arch-specific NVIDIA packages (nvidia, nvidia-dkms)
- [ ] >85% test coverage

---

#### P2-MS7: Implement Zypper Package Manager (openSUSE)

**Status:** `COMPLETED`
**Version:** 2.7.0
**Effort:** Large
**Dependencies:** P2-MS1, P2-MS2, P1-MS7

**Description:**
Implement the Manager interface for Zypper-based distributions (openSUSE Leap, Tumbleweed).

**Files to Create:**
```
internal/pkg/zypper/zypper.go       # Zypper implementation
internal/pkg/zypper/zypper_test.go  # Zypper tests
internal/pkg/zypper/repository.go   # Zypper repository handling
```

**Acceptance Criteria:**
- [ ] Implement all Manager interface methods
- [ ] zypper refresh, install, remove, update
- [ ] zypper search, info
- [ ] zypper addrepo, removerepo
- [ ] --non-interactive flag
- [ ] Handle openSUSE NVIDIA repository
- [ ] >85% test coverage

---

#### P2-MS8: Create Package Manager Factory

**Status:** `NOT_STARTED`
**Version:** 2.8.0
**Effort:** Small
**Dependencies:** P2-MS2 through P2-MS7

**Description:**
Create a factory that returns the correct package manager implementation based on detected distribution.

**Files to Create:**
```
internal/pkg/factory.go        # Package manager factory
internal/pkg/factory_test.go   # Factory tests
```

**Acceptance Criteria:**
- [ ] Return correct Manager for each distribution family
- [ ] Handle unknown distributions with meaningful error
- [ ] Support dependency injection for testing
- [ ] Lazy initialization of package managers
- [ ] >90% test coverage

---

#### P2-MS9: Create Distribution-Specific NVIDIA Package Mappings

**Status:** `NOT_STARTED`
**Version:** 2.9.0
**Effort:** Medium
**Dependencies:** P2-MS8

**Description:**
Create mappings between NVIDIA components and distribution-specific package names.

**Files to Create:**
```
internal/pkg/nvidia/packages.go       # NVIDIA package mappings
internal/pkg/nvidia/packages_test.go  # Package mapping tests
internal/pkg/nvidia/repository.go     # NVIDIA repository definitions
```

**Package Mappings:**
| Component | Ubuntu/Debian | Fedora/RHEL | Arch | openSUSE |
|-----------|--------------|-------------|------|----------|
| Driver | nvidia-driver-XXX | akmod-nvidia | nvidia | nvidia-driver-G06 |
| CUDA Toolkit | cuda-toolkit-XX-X | cuda | cuda | cuda |
| cuDNN | libcudnn8 | cudnn | cudnn | - |
| NVCC | nvidia-cuda-toolkit | cuda-compiler | cuda | cuda |

**Acceptance Criteria:**
- [ ] Package name mapping for all target distributions
- [ ] Version-specific package mappings (driver versions)
- [ ] Repository URLs for NVIDIA repos per distribution
- [ ] GPG key URLs for each repository
- [ ] Helper to get all required packages for a component
- [ ] >90% test coverage

---

### Phase 3 Sprint Specifications

#### P3-MS1: Implement PCI Device Scanner

**Status:** `COMPLETED`
**Version:** 3.1.0
**Effort:** Medium
**Dependencies:** P1-MS8

**Description:**
Implement a PCI device scanner that reads from `/sys/bus/pci/devices` to detect all PCI devices, with a focus on identifying NVIDIA GPUs. This is the foundation for GPU detection.

**Files to Create:**
```
internal/gpu/pci/scanner.go       # PCI device scanning logic
internal/gpu/pci/device.go        # PCI device struct and parsing
internal/gpu/pci/scanner_test.go  # Scanner tests with mock sysfs
```

**PCI Device Structure:**
```go
// PCIDevice represents a PCI device
type PCIDevice struct {
    Address     string // e.g., "0000:01:00.0"
    VendorID    string // e.g., "10de" for NVIDIA
    DeviceID    string // e.g., "2684" for RTX 4090
    Class       string // e.g., "0300" for VGA controller
    SubVendorID string
    SubDeviceID string
    Driver      string // Currently loaded driver (nvidia, nouveau, vfio-pci)
    Revision    string
}

// Scanner interface for PCI device discovery
type Scanner interface {
    // ScanAll returns all PCI devices
    ScanAll(ctx context.Context) ([]PCIDevice, error)
    
    // ScanByVendor returns devices matching vendor ID
    ScanByVendor(ctx context.Context, vendorID string) ([]PCIDevice, error)
    
    // ScanByClass returns devices matching class code
    ScanByClass(ctx context.Context, classCode string) ([]PCIDevice, error)
    
    // ScanNVIDIA returns all NVIDIA GPU devices
    ScanNVIDIA(ctx context.Context) ([]PCIDevice, error)
}
```

**Implementation Notes:**
- Read from `/sys/bus/pci/devices/*/` for device info
- Parse `vendor`, `device`, `class`, `subsystem_vendor`, `subsystem_device`
- Check `driver` symlink for currently bound driver
- NVIDIA vendor ID: `0x10de`
- VGA class: `0x0300`, 3D controller class: `0x0302`

**Acceptance Criteria:**
- [ ] Scanner interface defined with all methods
- [ ] PCIDevice struct with all relevant fields
- [ ] Reads from `/sys/bus/pci/devices` directory
- [ ] Parses vendor, device, class from sysfs files
- [ ] Detects currently bound driver via symlink
- [ ] ScanNVIDIA filters by vendor ID 0x10de
- [ ] Mock sysfs for unit testing
- [ ] >90% test coverage

---

#### P3-MS2: Create NVIDIA GPU Database

**Status:** `COMPLETED`
**Version:** 3.2.0
**Effort:** Medium
**Dependencies:** P3-MS1

**Description:**
Create a database mapping NVIDIA PCI device IDs to GPU models, architectures, and driver requirements. This enables matching detected hardware to appropriate driver versions.

**Files to Create:**
```
internal/gpu/nvidia/database.go      # GPU database and lookup
internal/gpu/nvidia/models.go        # GPU model definitions
internal/gpu/nvidia/database_test.go # Database tests
```

**GPU Model Structure:**
```go
// Architecture represents NVIDIA GPU architecture
type Architecture string

const (
    ArchTuring   Architecture = "turing"    // RTX 20xx, GTX 16xx
    ArchAmpere   Architecture = "ampere"    // RTX 30xx
    ArchAdaLovelace Architecture = "ada"    // RTX 40xx
    ArchHopper   Architecture = "hopper"    // H100
    ArchBlackwell Architecture = "blackwell" // B100, B200
    ArchPascal   Architecture = "pascal"    // GTX 10xx
    ArchMaxwell  Architecture = "maxwell"   // GTX 9xx
    ArchKepler   Architecture = "kepler"    // GTX 6xx, 7xx
    ArchUnknown  Architecture = "unknown"
)

// GPUModel represents an NVIDIA GPU model
type GPUModel struct {
    DeviceID       string       // PCI device ID
    Name           string       // e.g., "GeForce RTX 4090"
    Architecture   Architecture
    MinDriverVersion string     // Minimum supported driver version
    ComputeCapability string    // e.g., "8.9" for Ada
    MemorySize     string       // e.g., "24GB"
    IsDataCenter   bool         // H100, A100, etc.
}

// Database provides GPU model lookup
type Database interface {
    Lookup(deviceID string) (*GPUModel, bool)
    LookupByName(name string) (*GPUModel, bool)
    ListByArchitecture(arch Architecture) []GPUModel
    GetMinDriverVersion(deviceID string) (string, error)
}
```

**Acceptance Criteria:**
- [ ] Database interface for GPU lookup
- [ ] GPU model struct with all relevant fields
- [ ] Database of common NVIDIA GPUs (RTX 20xx through 50xx, GTX 10xx/16xx)
- [ ] Data center GPU support (A100, H100, H200)
- [ ] Architecture detection from device ID
- [ ] Minimum driver version requirements per architecture
- [ ] Lookup by device ID and by name
- [ ] >90% test coverage

---

#### P3-MS3: Implement nvidia-smi Parser

**Status:** `COMPLETED`
**Version:** 3.3.0
**Effort:** Medium
**Dependencies:** P1-MS8, P3-MS2

**Description:**
Parse `nvidia-smi` output to detect installed NVIDIA drivers, their version, and GPU status. This provides runtime GPU information when drivers are installed.

**Files to Create:**
```
internal/gpu/smi/parser.go       # nvidia-smi output parser
internal/gpu/smi/types.go        # nvidia-smi data types
internal/gpu/smi/parser_test.go  # Parser tests with sample output
```

**SMI Data Structure:**
```go
// SMIInfo represents nvidia-smi output
type SMIInfo struct {
    DriverVersion  string
    CUDAVersion    string
    GPUs           []SMIGPUInfo
    Available      bool // true if nvidia-smi works
}

// SMIGPUInfo represents per-GPU info from nvidia-smi
type SMIGPUInfo struct {
    Index           int
    Name            string
    UUID            string
    MemoryTotal     string
    MemoryUsed      string
    MemoryFree      string
    Temperature     int
    PowerDraw       string
    PowerLimit      string
    UtilizationGPU  int
    UtilizationMem  int
    ComputeMode     string
    PersistenceMode bool
}

// Parser interface
type Parser interface {
    // Parse runs nvidia-smi and parses output
    Parse(ctx context.Context) (*SMIInfo, error)
    
    // IsAvailable checks if nvidia-smi is available
    IsAvailable(ctx context.Context) bool
    
    // GetDriverVersion returns just the driver version
    GetDriverVersion(ctx context.Context) (string, error)
}
```

**Implementation Notes:**
- Use `nvidia-smi --query-gpu=... --format=csv,noheader,nounits` for parseable output
- Handle case where nvidia-smi is not installed
- Handle case where driver is installed but not loaded

**Acceptance Criteria:**
- [ ] Parser interface defined
- [ ] Parse nvidia-smi CSV output format
- [ ] Extract driver version, CUDA version
- [ ] Extract per-GPU info (name, memory, utilization)
- [ ] Handle nvidia-smi not found gracefully
- [ ] Handle nvidia-smi errors (driver not loaded)
- [ ] Uses exec.Executor for testability
- [ ] >90% test coverage with sample outputs

---

#### P3-MS4: Create Nouveau Driver Detector

**Status:** `COMPLETED`
**Version:** 3.4.0
**Effort:** Small
**Dependencies:** P3-MS1, P1-MS8

**Description:**
Detect if the Nouveau open-source NVIDIA driver is currently loaded and bound to GPU devices. This is critical because Nouveau must be blacklisted before installing proprietary drivers.

**Files to Create:**
```
internal/gpu/nouveau/detector.go      # Nouveau detection logic
internal/gpu/nouveau/detector_test.go # Detector tests
```

**Detector Interface:**
```go
// Status represents Nouveau driver status
type Status struct {
    Loaded          bool     // Kernel module loaded
    InUse           bool     // Currently in use by GPU
    BoundDevices    []string // PCI addresses bound to nouveau
    BlacklistExists bool     // /etc/modprobe.d blacklist exists
}

// Detector interface
type Detector interface {
    // Detect checks Nouveau driver status
    Detect(ctx context.Context) (*Status, error)
    
    // IsLoaded checks if nouveau module is loaded
    IsLoaded(ctx context.Context) (bool, error)
    
    // IsBlacklisted checks if nouveau is blacklisted
    IsBlacklisted(ctx context.Context) (bool, error)
    
    // GetBoundDevices returns PCI addresses using nouveau
    GetBoundDevices(ctx context.Context) ([]string, error)
}
```

**Implementation Notes:**
- Check `/sys/module/nouveau` for loaded module
- Check `/proc/modules` for module in use count
- Check PCI devices' driver symlinks for "nouveau"
- Check `/etc/modprobe.d/*.conf` for nouveau blacklist

**Acceptance Criteria:**
- [ ] Detector interface defined
- [ ] Detect if nouveau kernel module is loaded
- [ ] Detect if nouveau is in use (has bound devices)
- [ ] List PCI devices bound to nouveau
- [ ] Check for existing nouveau blacklist configuration
- [ ] Uses exec.Executor for testability
- [ ] >90% test coverage

---

#### P3-MS5: Implement Kernel Version and Module Detection

**Status:** `COMPLETED`
**Version:** 3.5.0
**Effort:** Medium
**Dependencies:** P1-MS8, P2-MS8

**Description:**
Detect kernel version, loaded kernel modules, and kernel headers installation status. This is needed to verify DKMS compatibility and kernel module requirements.

**Files to Create:**
```
internal/gpu/kernel/detector.go      # Kernel detection logic
internal/gpu/kernel/modules.go       # Kernel module operations
internal/gpu/kernel/detector_test.go # Detector tests
```

**Kernel Info Structure:**
```go
// KernelInfo represents kernel information
type KernelInfo struct {
    Version        string // e.g., "6.5.0-44-generic"
    Release        string // e.g., "6.5.0"
    Architecture   string // e.g., "x86_64"
    HeadersPath    string // e.g., "/usr/src/linux-headers-6.5.0-44-generic"
    HeadersInstalled bool
    SecureBootEnabled bool
}

// ModuleInfo represents a kernel module
type ModuleInfo struct {
    Name       string
    Size       int64
    UsedBy     []string
    State      string // "Live", "Loading", "Unloading"
}

// Detector interface
type Detector interface {
    // GetKernelInfo returns kernel information
    GetKernelInfo(ctx context.Context) (*KernelInfo, error)
    
    // IsModuleLoaded checks if a module is loaded
    IsModuleLoaded(ctx context.Context, name string) (bool, error)
    
    // GetLoadedModules returns all loaded modules
    GetLoadedModules(ctx context.Context) ([]ModuleInfo, error)
    
    // AreHeadersInstalled checks if kernel headers are installed
    AreHeadersInstalled(ctx context.Context) (bool, error)
    
    // GetHeadersPackage returns the package name for kernel headers
    GetHeadersPackage(ctx context.Context) (string, error)
    
    // IsSecureBootEnabled checks Secure Boot status
    IsSecureBootEnabled(ctx context.Context) (bool, error)
}
```

**Implementation Notes:**
- Use `uname -r` for kernel version
- Parse `/proc/modules` for loaded modules
- Check `/usr/src/linux-headers-$(uname -r)` for headers
- Use `mokutil --sb-state` for Secure Boot detection
- Kernel headers package varies by distro

**Acceptance Criteria:**
- [ ] Detector interface defined
- [ ] Parse kernel version from uname
- [ ] Parse /proc/modules for loaded modules
- [ ] Detect kernel headers installation
- [ ] Distribution-aware headers package name detection
- [ ] Secure Boot detection
- [ ] Uses exec.Executor for testability
- [ ] >90% test coverage

---

#### P3-MS6: Create System Requirements Validator

**Status:** `COMPLETED`
**Version:** 3.6.0
**Effort:** Medium
**Dependencies:** P3-MS5, P2-MS8

**Description:**
Validate system requirements for NVIDIA driver installation including disk space, kernel compatibility, required packages, and Secure Boot status.

**Files to Create:**
```
internal/gpu/validator/validator.go      # Validation logic
internal/gpu/validator/checks.go         # Individual check implementations
internal/gpu/validator/validator_test.go # Validator tests
```

**Validation Structure:**
```go
// CheckResult represents a single validation check result
type CheckResult struct {
    Name        string
    Passed      bool
    Message     string
    Severity    Severity // Error, Warning, Info
    Remediation string   // How to fix if failed
}

type Severity string

const (
    SeverityError   Severity = "error"   // Installation cannot proceed
    SeverityWarning Severity = "warning" // May cause issues
    SeverityInfo    Severity = "info"    // Informational only
)

// ValidationReport represents all check results
type ValidationReport struct {
    Passed      bool
    Checks      []CheckResult
    Errors      []CheckResult
    Warnings    []CheckResult
    Timestamp   time.Time
}

// Validator interface
type Validator interface {
    // Validate runs all checks and returns report
    Validate(ctx context.Context) (*ValidationReport, error)
    
    // ValidateKernel checks kernel compatibility
    ValidateKernel(ctx context.Context) (*CheckResult, error)
    
    // ValidateDiskSpace checks available disk space
    ValidateDiskSpace(ctx context.Context, requiredMB int64) (*CheckResult, error)
    
    // ValidateSecureBoot checks Secure Boot configuration
    ValidateSecureBoot(ctx context.Context) (*CheckResult, error)
    
    // ValidateKernelHeaders checks kernel headers
    ValidateKernelHeaders(ctx context.Context) (*CheckResult, error)
    
    // ValidateBuildTools checks for gcc, make, etc.
    ValidateBuildTools(ctx context.Context) (*CheckResult, error)
}
```

**Implementation Notes:**
- Minimum 5GB disk space recommended for CUDA toolkit
- Kernel headers required for DKMS
- gcc, make required for module building
- Secure Boot requires signed modules

**Acceptance Criteria:**
- [ ] Validator interface defined
- [ ] Check disk space availability (configurable threshold)
- [ ] Check kernel headers installation
- [ ] Check build tools (gcc, make) presence
- [ ] Check Secure Boot status with appropriate warning
- [ ] Check kernel version compatibility
- [ ] Aggregate results into ValidationReport
- [ ] Severity levels for different issues
- [ ] Remediation suggestions for failed checks
- [ ] >90% test coverage

---

#### P3-MS7: Build GPU Detection Orchestrator

**Status:** `IN_PROGRESS`
**Version:** 3.7.0
**Effort:** Medium
**Dependencies:** P3-MS1 through P3-MS6

**Description:**
Orchestrate all GPU detection components into a unified interface that provides comprehensive system GPU information and installation readiness assessment.

**Files to Create:**
```
internal/gpu/orchestrator.go      # Main orchestrator
internal/gpu/types.go             # Shared GPU types
internal/gpu/orchestrator_test.go # Orchestrator tests
```

**Orchestrator Interface:**
```go
// GPUInfo represents complete GPU information
type GPUInfo struct {
    // Hardware detection
    PCIDevices      []pci.PCIDevice
    NVIDIAGPUs      []NVIDIAGPUInfo
    
    // Driver status
    InstalledDriver *DriverInfo
    NouveauStatus   *nouveau.Status
    
    // System info
    KernelInfo      *kernel.KernelInfo
    ValidationReport *validator.ValidationReport
    
    // Detection metadata
    DetectionTime   time.Time
    Errors          []error
}

// NVIDIAGPUInfo combines hardware and database info
type NVIDIAGPUInfo struct {
    PCIDevice     pci.PCIDevice
    Model         *nvidia.GPUModel // nil if not in database
    SMIInfo       *smi.SMIGPUInfo  // nil if nvidia-smi unavailable
}

// DriverInfo represents installed driver information
type DriverInfo struct {
    Installed     bool
    Type          string // "nvidia", "nouveau", "none"
    Version       string
    CUDAVersion   string
}

// Orchestrator interface
type Orchestrator interface {
    // DetectAll runs all detection components
    DetectAll(ctx context.Context) (*GPUInfo, error)
    
    // DetectGPUs detects GPU hardware only
    DetectGPUs(ctx context.Context) ([]NVIDIAGPUInfo, error)
    
    // GetDriverStatus gets current driver status
    GetDriverStatus(ctx context.Context) (*DriverInfo, error)
    
    // ValidateSystem validates installation requirements
    ValidateSystem(ctx context.Context) (*validator.ValidationReport, error)
    
    // IsReadyForInstall checks if system is ready for driver install
    IsReadyForInstall(ctx context.Context) (bool, []string, error)
}
```

**Implementation Notes:**
- Aggregate results from all detection components
- Handle partial failures gracefully (one component failure shouldn't stop others)
- Provide clear readiness assessment
- Cache results where appropriate

**Acceptance Criteria:**
- [ ] Orchestrator interface defined
- [ ] Integrates PCI scanner, GPU database, nvidia-smi parser
- [ ] Integrates Nouveau detector, kernel detector, validator
- [ ] Aggregates all GPU info into unified struct
- [ ] Handles partial failures gracefully
- [ ] IsReadyForInstall returns clear yes/no with reasons
- [ ] Detection timeout support
- [ ] Uses dependency injection for all components
- [ ] >90% test coverage

---

## 5. Activity Log

### Log Format

```markdown
### Session YYYY-MM-DD HH:MM - DESCRIPTION

**Sprint:** P{X}-MS{Y}
**Version:** X.Y.Z
**Status:** IN_PROGRESS | COMPLETED | BLOCKED

**Activities:**
- [x] Completed activity
- [ ] Pending activity

**Code Review Notes:**
- Issue: Description (FIXED/PENDING)

**Test Results:**
- `go test ./...`: PASS/FAIL (N passed, M failed)
- `go build ./...`: PASS/FAIL

**Human Validation:** PENDING | APPROVED | CHANGES_REQUESTED

**Commits:**
- `abc1234` - Commit message

**Blockers:**
- Blocker description (if any)

**Notes:**
Additional context
```

---

### Session Log

#### Session 2026-01-03 14:00 - Project Planning

**Sprint:** N/A (Planning Phase)
**Version:** 0.0.0
**Status:** COMPLETED

**Activities:**
- [x] Read existing bash script (install_nvidia.sh)
- [x] Research Bubble Tea framework documentation
- [x] Research Bubbles component library
- [x] Research multi-distro package management approaches
- [x] Research NVIDIA driver/CUDA installation per distro
- [x] Invoke high-level-planner agent for strategic plan
- [x] Invoke microsprint-analyst agent for detailed sprints
- [x] Create 7-phase implementation plan
- [x] Define 62 detailed microsprints
- [x] Document development pipeline with version management
- [x] Create this project documentation (IGOR_PROJECT.md)

**Human Validation:** APPROVED

**Notes:**
- User approved comprehensive plan with pipeline definition
- Ready to begin Phase 1 implementation
- GitHub repository to be initialized in P1-MS1
- Version format: x.y.z (phase.sprint.iteration)

---

#### Session 2026-01-03 14:40 - P1-MS1 Implementation

**Sprint:** P1-MS1
**Version:** 1.1.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created Go module (github.com/tungetti/igor)
- [x] Created complete directory structure (17 internal packages)
- [x] Created main.go with version info
- [x] Created Makefile with build, test, clean targets
- [x] Created README.md, LICENSE (MIT), CHANGELOG.md
- [x] Created VERSION file with "1.1.0"
- [x] Created build.sh and test.sh scripts
- [x] Code review by code-reviewer agent
- [x] Fixed review issues (Go version alignment, removed os.Exit(0))
- [x] All tests passed (go test, go build, go vet, make build, make test)

**Code Review Notes:**
- Issue: Go version mismatch (go.mod had 1.25.5, README had 1.21) - FIXED
- Issue: Unnecessary os.Exit(0) in main.go - FIXED

**Test Results:**
- `go test ./...`: PASS (no test files yet)
- `go build ./...`: PASS
- `go vet ./...`: PASS (no issues)
- `make build`: PASS
- `make test`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 1] Initialize project structure` - v1.1.0

**Notes:**
- All 28 files created as specified
- Repository pushed to https://github.com/tungetti/igor.git
- Ready for P1-MS2 (Add Core Dependencies)

---

#### Session 2026-01-03 14:50 - P1-MS2 Implementation

**Sprint:** P1-MS2
**Version:** 1.2.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Added 6 core dependencies to go.mod
- [x] Created imports_test.go to verify all imports work
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed (go mod verify, go build, go test, go vet)

**Dependencies Added:**
- github.com/charmbracelet/bubbletea v0.25.0
- github.com/charmbracelet/bubbles v0.18.0
- github.com/charmbracelet/lipgloss v0.9.1
- github.com/charmbracelet/log v0.3.1
- github.com/stretchr/testify v1.8.4
- gopkg.in/yaml.v3 v3.0.1

**Test Results:**
- `go mod verify`: all modules verified
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 2] Add core dependencies` - v1.2.0

**Notes:**
- go.sum generated with 65 lines of checksums
- Ready for P1-MS3 (Create Error Types and Constants)

---

#### Session 2026-01-03 15:00 - P1-MS3 Implementation

**Sprint:** P1-MS3
**Version:** 1.3.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/errors/errors.go with 14 error codes
- [x] Created internal/errors/errors_test.go with 30+ tests
- [x] Created internal/constants/constants.go with typed constants
- [x] Created internal/constants/constants_test.go with 11 tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 100% coverage

**Test Results:**
- `internal/errors`: 100% coverage
- `internal/constants`: 100% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 3] Create error types and constants` - v1.3.0

**Notes:**
- Error package provides comprehensive error handling with wrapping support
- Constants package provides typed constants for entire application
- Ready for P1-MS4 (Implement Logging Infrastructure)

---

#### Session 2026-01-03 15:10 - P1-MS4 Implementation

**Sprint:** P1-MS4
**Version:** 1.4.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/logging/levels.go with Level type
- [x] Created internal/logging/logger.go with Logger interface
- [x] Created internal/logging/logger_test.go with 30 tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 98.9% coverage

**Features Implemented:**
- Logger interface (mockable for testing)
- Log levels with filtering (Debug, Info, Warn, Error)
- Thread-safe implementation with sync.RWMutex
- File logging (TUI-safe, no colors)
- Structured logging (key-value pairs)
- WithPrefix/WithFields fluent API
- NopLogger and MultiLogger

**Test Results:**
- `internal/logging`: 98.9% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 4] Implement logging infrastructure` - v1.4.0

**Notes:**
- Uses charmbracelet/log as underlying implementation
- Ready for P1-MS5 (Create Configuration Management System)

---

#### Session 2026-01-03 15:20 - P1-MS5 Implementation

**Sprint:** P1-MS5
**Version:** 1.5.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/config/config.go with Config struct
- [x] Created internal/config/defaults.go with XDG compliance
- [x] Created internal/config/loader.go with YAML and env loading
- [x] Created internal/config/validator.go with validation logic
- [x] Created internal/config/config_test.go with 40+ tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 89.4% coverage

**Features Implemented:**
- YAML configuration file loading
- Environment variable overrides (IGOR_* prefix)
- XDG Base Directory compliance
- Configuration validation
- Helper methods (Clone, ConfigPath, IsVerbose, etc.)

**Test Results:**
- `internal/config`: 89.4% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 5] Create configuration management system` - v1.5.0

**Notes:**
- Configuration integrates with logging and errors packages
- Ready for P1-MS6 (Implement CLI Argument Parser)

---

#### Session 2026-01-03 15:30 - P1-MS6 Implementation

**Sprint:** P1-MS6
**Version:** 1.6.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/cli/flags.go with flag structures
- [x] Created internal/cli/commands.go with command definitions
- [x] Created internal/cli/parser.go with argument parser
- [x] Created internal/cli/parser_test.go with 852 lines of tests
- [x] Created cmd/igor/cli.go and cmd/igor/version.go
- [x] Updated cmd/igor/main.go for CLI integration
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 97% coverage

**Features Implemented:**
- Commands: install, uninstall, detect, list, version, help
- Command aliases (i, u, d, l, v, h)
- Global flags: --verbose, --quiet, --dry-run, --config, --no-color
- Command-specific flags for each subcommand
- Integrated help system

**Test Results:**
- `internal/cli`: 97% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 6] Implement CLI argument parser` - v1.6.0

**Notes:**
- CLI integrates with config package for merged settings
- Ready for P1-MS7 (Create Root Privilege Handler)

---

#### Session 2026-01-03 15:40 - P1-MS7 Implementation

**Sprint:** P1-MS7
**Version:** 1.7.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/privilege/privilege.go with Manager
- [x] Created internal/privilege/sudo.go with sudo support
- [x] Created internal/privilege/pkexec.go with pkexec support
- [x] Created internal/privilege/privilege_test.go with 28 tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 91% coverage

**Features Implemented:**
- Root detection (os.Geteuid())
- Elevation methods: sudo, pkexec, doas
- Non-interactive sudo (-n flag)
- Environment sanitization (removes LD_PRELOAD, etc.)
- Safe PATH enforcement
- Context support for timeout/cancellation

**Test Results:**
- `internal/privilege`: 91% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 7] Create root privilege handler` - v1.7.0

**Notes:**
- Security-hardened implementation
- Ready for P1-MS8 (Implement Command Executor Interface)

---

#### Session 2026-01-03 15:50 - P1-MS8 Implementation

**Sprint:** P1-MS8
**Version:** 1.8.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/exec/output.go with Result struct
- [x] Created internal/exec/executor.go with Executor interface
- [x] Created internal/exec/mock.go with MockExecutor
- [x] Created internal/exec/executor_test.go with 67 tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 98.4% coverage

**Features Implemented:**
- Executor interface (Execute, ExecuteElevated, ExecuteWithInput, Stream)
- Result struct with output capture and helpers
- MockExecutor with call recording and response configuration
- Integration with privilege.Manager
- Context support for timeout/cancellation

**Test Results:**
- `internal/exec`: 98.4% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 8] Implement command executor interface` - v1.8.0

**Notes:**
- Executor is mockable for testing throughout the project
- Ready for P1-MS9 (Create Application Bootstrap and Lifecycle)

---

#### Session 2026-01-03 16:00 - P1-MS9 Implementation

**Sprint:** P1-MS9
**Version:** 1.9.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/app/app.go with Application struct
- [x] Created internal/app/container.go with dependency injection
- [x] Created internal/app/lifecycle.go with signal handling
- [x] Created internal/app/app_test.go with comprehensive tests
- [x] Updated cmd/igor/main.go to optionally use app package
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 93% coverage

**Features Implemented:**
- Application struct with Run(), Shutdown() methods
- Container with Config, Logger, Executor, Privilege injection
- Signal handling (SIGINT, SIGTERM) with graceful shutdown
- Panic recovery with stack trace logging
- LIFO shutdown order for cleanup functions
- Context propagation throughout application

**Test Results:**
- `internal/app`: 93% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 1 Sprint 9] Create application bootstrap and lifecycle` - v1.9.0

**Notes:**
- Phase 1 Foundation Layer is now COMPLETE
- Ready for Phase 2 (Distribution Detection & Package Manager Abstraction)

---

#### Session 2026-01-03 16:30 - P2-MS1 Implementation

**Sprint:** P2-MS1
**Version:** 2.1.0
**Status:** COMPLETED

**Activities:**
- [x] Created Phase 2 sprint specifications in IGOR_PROJECT.md
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/types.go with Package, Repository, Options types
- [x] Created internal/pkg/errors.go with package manager errors
- [x] Created internal/pkg/manager.go with Manager interface
- [x] Created internal/pkg/manager_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 100% coverage

**Features Implemented:**
- Manager interface with all CRUD operations for packages
- Extended interfaces: RepositoryManager, LockableManager, TransactionalManager, HistoryManager
- ManagerFactory interface for dependency injection
- Package struct with Name, Version, Installed, Repository, Description, Size
- Repository struct with Name, URL, Enabled, GPGKey, Type
- InstallOptions, UpdateOptions, RemoveOptions, SearchOptions with defaults
- PackageError type with errors.Is/As support and igor integration
- 14 sentinel errors for package operations

**Test Results:**
- `internal/pkg`: 100% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 1] Define package manager interface` - v2.1.0

**Notes:**
- Phase 2 has begun
- Ready for P2-MS2 (Implement Distribution Detection Core)

---

#### Session 2026-01-03 17:00 - P2-MS2 Implementation

**Sprint:** P2-MS2
**Version:** 2.2.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/distro/types.go with Distribution struct
- [x] Created internal/distro/osrelease.go with parsers
- [x] Created internal/distro/detector.go with 8-level fallback chain
- [x] Created internal/distro/detector_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 96% coverage

**Features Implemented:**
- Distribution struct with ID, Name, Version, Family, IDLike fields
- Helper methods: IsDebian(), IsRHEL(), IsArch(), IsSUSE(), MajorVersion(), IsRolling()
- /etc/os-release and /etc/lsb-release parsers
- FileReader interface for mockable filesystem access
- 8-level fallback detection chain
- Support for 15+ distributions with family detection

**Test Results:**
- `internal/distro`: 96% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 2] Implement distribution detection core` - v2.2.0

**Notes:**
- Detection works for Ubuntu, Debian, Fedora, RHEL, Rocky, AlmaLinux, CentOS, Arch, Manjaro, EndeavourOS, openSUSE Leap/Tumbleweed, Linux Mint, Pop!_OS
- Ready for P2-MS3 (Implement APT Package Manager)

---

#### Session 2026-01-03 17:30 - P2-MS3 Implementation

**Sprint:** P2-MS3
**Version:** 2.3.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/apt/apt.go with APT Manager
- [x] Created internal/pkg/apt/repository.go with repo management
- [x] Created internal/pkg/apt/parser.go with output parsers
- [x] Created internal/pkg/apt/apt_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED WITH CHANGES
- [x] Fixed deprecated --force-yes flag (replaced with modern options)
- [x] Fixed shell injection vulnerability in AddGPGKey
- [x] All tests passed with 94% coverage

**Features Implemented:**
- Full pkg.Manager interface for APT
- apt-get install/remove/update/upgrade with DEBIAN_FRONTEND=noninteractive
- dpkg-query for package status checks
- apt-cache for search/info operations
- PPA support via add-apt-repository
- Modern GPG key handling (/etc/apt/keyrings) with apt-key fallback
- Repository CRUD operations
- Input validation for security

**Test Results:**
- `internal/pkg/apt`: 94% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 3] Implement APT package manager` - v2.3.0

**Notes:**
- First package manager implementation complete
- Ready for P2-MS4 (Implement DNF Package Manager)

---

#### Session 2026-01-03 18:00 - P2-MS4 Implementation

**Sprint:** P2-MS4
**Version:** 2.4.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/dnf/dnf.go with DNF Manager
- [x] Created internal/pkg/dnf/repository.go with repo/RPM Fusion support
- [x] Created internal/pkg/dnf/parser.go with rpm/dnf output parsers
- [x] Created internal/pkg/dnf/dnf_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED WITH CHANGES
- [x] Fixed duplicate --allowerasing flag issue
- [x] All tests passed with 93.8% coverage

**Features Implemented:**
- Full pkg.Manager interface for DNF
- dnf install/remove/upgrade with -y flag for non-interactive
- Correct handling of dnf check-update exit code 100
- rpm -q for package status checks
- RPM Fusion support (AddRPMFusion, AddRPMFusionEL)
- GPG key import via rpm --import
- Repository management via dnf config-manager

**Test Results:**
- `internal/pkg/dnf`: 93.8% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 4] Implement DNF package manager` - v2.4.0

**Notes:**
- Second package manager implementation complete
- Ready for P2-MS5 (Implement YUM Package Manager)

---

#### Session 2026-01-03 18:30 - P2-MS5 Implementation

**Sprint:** P2-MS5
**Version:** 2.5.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/yum/yum.go with YUM Manager
- [x] Created internal/pkg/yum/repository.go with EPEL/ELRepo support
- [x] Created internal/pkg/yum/parser.go with yum/rpm output parsers
- [x] Created internal/pkg/yum/yum_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 87% coverage

**Features Implemented:**
- Full pkg.Manager interface for YUM
- Uses yum update (not upgrade) correctly
- yum-config-manager for repository management
- EPEL support with RHEL fallback URL
- ELRepo and NVIDIA CUDA repo support
- RPM Fusion EL7 repository support
- Handles exit code 100 from yum check-update
- Fallbacks for older YUM without autoremove

**Test Results:**
- `internal/pkg/yum`: 87% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 5] Implement YUM package manager` - v2.5.0

**Notes:**
- Third package manager implementation complete (APT, DNF, YUM)
- Ready for P2-MS6 (Implement Pacman Package Manager)

---

#### Session 2026-01-03 19:00 - P2-MS6 Implementation

**Sprint:** P2-MS6
**Version:** 2.6.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/pacman/pacman.go with Pacman Manager
- [x] Created internal/pkg/pacman/repository.go with pacman.conf management
- [x] Created internal/pkg/pacman/parser.go with pacman output parsers
- [x] Created internal/pkg/pacman/pacman_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED WITH CHANGES
- [x] Fixed duplicate --overwrite flag
- [x] Fixed misleading SkipVerify implementation
- [x] All tests passed with 90.6% coverage

**Features Implemented:**
- Full pkg.Manager interface for Pacman
- Uses -S, -R, -Q with --noconfirm for non-interactive
- Repository management via pacman.conf
- GPG key management via pacman-key
- Orphan removal via pacman -Qdtq
- Supports Arch, Manjaro, EndeavourOS, Garuda, Artix

**Test Results:**
- `internal/pkg/pacman`: 90.6% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 6] Implement Pacman package manager` - v2.6.0

**Notes:**
- Fourth package manager implementation complete (APT, DNF, YUM, Pacman)
- Ready for P2-MS7 (Implement Zypper Package Manager)

---

#### Session 2026-01-03 19:30 - P2-MS7 Implementation

**Sprint:** P2-MS7
**Version:** 2.7.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/zypper/zypper.go with Zypper Manager
- [x] Created internal/pkg/zypper/repository.go with NVIDIA repo support
- [x] Created internal/pkg/zypper/parser.go with zypper/rpm output parsers
- [x] Created internal/pkg/zypper/zypper_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED WITH CHANGES
- [x] Added --non-interactive to Update()
- [x] Fixed argument order in Upgrade()
- [x] All tests passed with 90.7% coverage

**Features Implemented:**
- Full pkg.Manager interface for Zypper
- Uses --non-interactive for unattended operation
- Uses dist-upgrade for full system upgrade
- rpm -q for package status (shared with DNF/YUM)
- NVIDIA repo support for Tumbleweed and Leap
- zypper addrepo/modifyrepo for repository management

**Test Results:**
- `internal/pkg/zypper`: 90.7% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 7] Implement Zypper package manager` - v2.7.0

**Notes:**
- Fifth and final package manager implementation complete (APT, DNF, YUM, Pacman, Zypper)
- Ready for P2-MS8 (Create Package Manager Factory)

---

#### Session 2026-01-03 20:00 - P2-MS8 Implementation

**Sprint:** P2-MS8
**Version:** 2.8.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/factory/factory.go with Factory
- [x] Created internal/pkg/factory/factory_test.go with tests
- [x] All tests passed with 97% coverage

**Features Implemented:**
- Factory.Create() auto-detects distribution
- Factory.CreateForFamily() for explicit selection
- Factory.CreateForDistribution() for version-based selection
- Handles YUM vs DNF for CentOS/RHEL 7 vs 8+
- Fedora always returns DNF
- AvailableManagers() and SupportedFamilies() helpers

**Test Results:**
- `internal/pkg/factory`: 97% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 8] Create package manager factory` - v2.8.0

**Notes:**
- Factory unifies all 5 package managers
- Ready for P2-MS9 (Create NVIDIA Package Mappings)

---

#### Session 2026-01-03 20:30 - P2-MS9 Implementation

**Sprint:** P2-MS9
**Version:** 2.9.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/pkg/nvidia/packages.go with package mappings
- [x] Created internal/pkg/nvidia/repository.go with repo definitions
- [x] Created internal/pkg/nvidia/packages_test.go
- [x] Created internal/pkg/nvidia/repository_test.go
- [x] All tests passed with 97.9% coverage

**Features Implemented:**
- Component types: driver, driver-dkms, cuda, cudnn, nvcc, utils, settings, opencl, vulkan
- Package mappings for all 4 distribution families
- Distribution-specific overrides (Ubuntu, Pop!_OS, Fedora, Manjaro, Tumbleweed, Leap)
- Version-specific driver packages (550, 545, 535 LTS, 525, 470 legacy)
- Repository URLs with GPG keys
- Helper functions: GetAllPackages, GetMinimalPackages, GetDevelopmentPackages, GetGraphicsPackages

**Test Results:**
- `internal/pkg/nvidia`: 97.9% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 2 Sprint 9] Create NVIDIA package mappings` - v2.9.0

**Notes:**
- **PHASE 2 COMPLETE!**
- All 9 sprints of Phase 2 finished
- Ready for Phase 3 (GPU Detection & System Analysis)

---

#### Session 2026-01-04 - P3-MS1 Implementation

**Sprint:** P3-MS1
**Version:** 3.1.0
**Status:** COMPLETED

**Activities:**
- [x] Added detailed Phase 3 sprint specifications to IGOR_PROJECT.md
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/pci/device.go with PCIDevice struct
- [x] Created internal/gpu/pci/scanner.go with Scanner interface
- [x] Created internal/gpu/pci/scanner_test.go with 58 tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 96.2% coverage

**Features Implemented:**
- PCI device scanner reading from /sys/bus/pci/devices
- Scanner interface: ScanAll, ScanByVendor, ScanByClass, ScanNVIDIA
- PCIDevice struct with Address, VendorID, DeviceID, Class, SubVendorID, SubDeviceID, Driver, Revision
- FileSystem abstraction for testability (RealFileSystem, MockFileSystem)
- Helper methods: IsNVIDIA(), IsGPU(), IsNVIDIAGPU(), HasDriver(), IsUsingProprietaryDriver(), IsUsingNouveau(), IsUsingVFIO()
- Constants for NVIDIA vendor ID (0x10de) and GPU class codes

**Test Results:**
- `internal/gpu/pci`: 96.2% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 1] Implement PCI device scanner` - v3.1.0

**Notes:**
- **PHASE 3 STARTED!**
- First GPU detection component complete
- Foundation for GPU detection orchestrator
- Ready for P3-MS2 (Create NVIDIA GPU Database)

---

#### Session 2026-01-04 - P3-MS2 Implementation

**Sprint:** P3-MS2
**Version:** 3.2.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/nvidia/models.go with Architecture constants
- [x] Created internal/gpu/nvidia/database.go with Database interface
- [x] Created internal/gpu/nvidia/database_test.go with 26 tests
- [x] Code review by code-reviewer agent - APPROVED WITH CHANGES
- [x] Fixed duplicate device IDs (B200/H200 and A40/A10 conflicts)
- [x] Updated error handling to use internal/errors package
- [x] Added TestNoDuplicateDeviceIDs test
- [x] All tests passed with 100% coverage

**Features Implemented:**
- Database interface: Lookup, LookupByName, ListByArchitecture, GetMinDriverVersion, AllModels, Count
- 54 GPU models from 9 architectures (Blackwell through Kepler)
- Consumer GPUs: RTX 40xx, 30xx, 20xx, GTX 16xx, 10xx
- Data center GPUs: B200, B100, H200, H100, A100, A40, A30, A10, V100
- Architecture methods: String, IsValid, MinDriverVersion, ComputeCapability
- Thread-safe implementation with sync.RWMutex
- Convenience functions: LookupDevice, LookupDeviceByName

**Test Results:**
- `internal/gpu/nvidia`: 100% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 2] Create NVIDIA GPU database` - v3.2.0

**Notes:**
- Second GPU detection component complete
- Maps PCI device IDs to GPU models and driver requirements
- Ready for P3-MS3 (Implement nvidia-smi Parser)

---

#### Session 2026-01-04 - P3-MS3 Implementation

**Sprint:** P3-MS3
**Version:** 3.3.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/smi/types.go with SMIInfo and SMIGPUInfo structs
- [x] Created internal/gpu/smi/parser.go with Parser interface
- [x] Created internal/gpu/smi/parser_test.go with 65 tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 93.7% coverage

**Features Implemented:**
- Parser interface: Parse, IsAvailable, GetDriverVersion, GetCUDAVersion, GetGPUCount
- SMIInfo struct with driver/CUDA version and GPU list
- SMIGPUInfo struct with 13 fields (memory, temp, power, utilization, etc.)
- Robust CSV parsing with quoted field handling
- Error handling: nvidia-smi not found, driver not loaded, no devices
- Helper methods: MemoryUsagePercent, PowerUsagePercent, IsIdle, TotalMemory
- Uses exec.Executor for testability

**Test Results:**
- `internal/gpu/smi`: 93.7% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 3] Implement nvidia-smi parser` - v3.3.0

**Notes:**
- Third GPU detection component complete
- Provides runtime GPU info when drivers are installed
- Ready for P3-MS4 (Create Nouveau Driver Detector)

---

#### Session 2026-01-04 - P3-MS4 Implementation

**Sprint:** P3-MS4
**Version:** 3.4.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/nouveau/detector.go with Detector interface
- [x] Created internal/gpu/nouveau/detector_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 94.9% coverage

**Features Implemented:**
- Detector interface: Detect, IsLoaded, IsBlacklisted, GetBoundDevices
- Status struct: Loaded, InUse, BoundDevices, BlacklistExists, BlacklistFiles
- Module detection via /sys/module/nouveau
- Bound device detection via pci.Scanner integration
- Blacklist detection scanning /etc/modprobe.d/*.conf
- FileSystem abstraction for testability

**Test Results:**
- `internal/gpu/nouveau`: 94.9% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 4] Create Nouveau driver detector` - v3.4.0

**Notes:**
- Fourth GPU detection component complete
- Critical for detecting if Nouveau needs to be blacklisted
- Ready for P3-MS5 (Implement Kernel Version and Module Detection)

---

#### Session 2026-01-04 - P3-MS5 Implementation

**Sprint:** P3-MS5
**Version:** 3.5.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/kernel/modules.go with ModuleInfo and parsing
- [x] Created internal/gpu/kernel/detector.go with Detector interface
- [x] Created internal/gpu/kernel/detector_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 92.2% coverage

**Features Implemented:**
- Detector interface: GetKernelInfo, IsModuleLoaded, GetLoadedModules, GetModule
- AreHeadersInstalled, GetHeadersPackage, IsSecureBootEnabled methods
- KernelInfo struct: Version, Release, Architecture, HeadersPath, HeadersInstalled, SecureBootEnabled
- ModuleInfo struct: Name, Size, UsedBy, UsedCount, State
- /proc/modules parsing with error handling
- Distribution-aware kernel headers package names (Debian, RHEL, Arch, SUSE)
- Secure Boot detection via mokutil and EFI variables fallback

**Test Results:**
- `internal/gpu/kernel`: 92.2% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 5] Implement kernel version and module detection` - v3.5.0

**Notes:**
- Fifth GPU detection component complete
- Provides kernel info needed for DKMS compatibility
- Ready for P3-MS6 (Create System Requirements Validator)

---

#### Session 2026-01-04 - P3-MS6 Implementation

**Sprint:** P3-MS6
**Version:** 3.6.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/validator/checks.go with individual validation checks
- [x] Created internal/gpu/validator/validator.go with Validator interface
- [x] Created internal/gpu/validator/validator_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 97.7% coverage

**Features Implemented:**
- Validator interface: Validate, ValidateKernel, ValidateDiskSpace, ValidateSecureBoot
- ValidateKernelHeaders, ValidateBuildTools, ValidateNouveauStatus methods
- CheckResult struct: Name, Passed, Message, Severity, Remediation
- ValidationReport with aggregated Errors, Warnings, Infos
- Severity levels: Error, Warning, Info
- Configurable thresholds via ValidatorOptions
- Integration with kernel.Detector and nouveau.Detector
- Remediation suggestions for all failed checks

**Test Results:**
- `internal/gpu/validator`: 97.7% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 6] Create system requirements validator` - v3.6.0

**Notes:**
- Sixth GPU detection component complete
- Validates system readiness for driver installation
- Ready for P3-MS7 (Build GPU Detection Orchestrator)

---

#### Session 2026-01-04 - P3-MS7 Implementation

**Sprint:** P3-MS7
**Version:** 3.7.0
**Status:** COMPLETED

**Activities:**
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/gpu/types.go with shared GPU types (DriverInfo, NVIDIAGPUInfo, GPUInfo)
- [x] Created internal/gpu/orchestrator.go with Orchestrator interface and implementation
- [x] Created internal/gpu/orchestrator_test.go with comprehensive tests
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 95.8% coverage

**Features Implemented:**
- Orchestrator interface: DetectAll, DetectGPUs, GetDriverStatus, ValidateSystem, IsReadyForInstall
- GPUInfo struct aggregating: PCIDevices, NVIDIAGPUs, InstalledDriver, NouveauStatus, KernelInfo, ValidationReport
- NVIDIAGPUInfo combining PCIDevice with database lookup and SMI info
- DriverInfo struct: Installed, Type, Version, CUDAVersion
- Concurrent detection with graceful partial failure handling
- Integration of all Phase 3 components (pci, nvidia, smi, nouveau, kernel, validator)
- Dependency injection for all components via OrchestratorDeps

**Test Results:**
- `internal/gpu`: 95.8% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 3 Sprint 7] Build GPU detection orchestrator` - v3.7.0

**Notes:**
- **PHASE 3 COMPLETE!**
- All 7 sprints of Phase 3 finished
- GPU detection layer fully implemented
- Ready for Phase 4 (TUI Framework & Core Views)

---

#### Session 2026-01-04 - P5-MS4 Implementation

**Sprint:** P5-MS4
**Version:** 5.4.0
**Status:** COMPLETED

**Activities:**
- [x] Updated VERSION file to 5.4.0
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/install/steps/nouveau.go (357 lines)
- [x] Created internal/install/steps/nouveau_test.go (1463 lines, 54+ tests)
- [x] Code review by code-reviewer agent - APPROVED
- [x] All tests passed with 98.3% coverage

**Features Implemented:**
- NouveauBlacklistStep implementing install.Step interface
- Creates `/etc/modprobe.d/blacklist-nouveau.conf` with blacklist entries
- Distribution-specific initramfs regeneration:
  - Debian/Ubuntu: `update-initramfs -u`
  - Fedora/RHEL/SUSE: `dracut --force`
  - Arch: `mkinitcpio -P`
- Uses `tee` command for privileged file writing
- Full rollback support (removes file, regenerates initramfs)
- Skips if Nouveau already blacklisted
- Functional options: WithBlacklistPath, WithNouveauDetector, WithSkipInitramfs, WithFileWriter
- State keys: StateNouveauBlacklisted, StateNouveauBlacklistFile
- Dry-run mode support
- Cancellation handling at multiple checkpoints

**Test Results:**
- `internal/install/steps`: 98.3% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 5 Sprint 4] Implement Nouveau blacklist step` - v5.4.0

**Notes:**
- Fourth installation step complete
- Critical step for blacklisting Nouveau before NVIDIA driver installation
- Ready for P5-MS5 (Package Installation Step)

---

#### Session 2026-01-04 - P5-MS5 Implementation

**Sprint:** P5-MS5
**Version:** 5.5.0
**Status:** COMPLETED

**Activities:**
- [x] Updated VERSION file to 5.5.0
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/install/steps/packages.go
- [x] Created internal/install/steps/packages_test.go (60+ tests)
- [x] Code review by code-reviewer agent - APPROVED WITH MINOR CHANGES
- [x] Fixed rollback logging issues (added LogWarn for failed rollbacks)
- [x] Added TODO for unused skipDependencies field
- [x] All tests passed with 95.9% coverage

**Features Implemented:**
- PackageInstallationStep implementing install.Step interface
- Integrates with nvidia.GetPackageSet for distribution-specific packages
- Supports version-specific driver packages via GetPackagesForVersion
- Maps component strings to nvidia.Component for package lookup
- Automatic package deduplication
- Functional options:
  - WithAdditionalPackages: Add extra packages beyond computed ones
  - WithSkipDependencies: Skip dependency checking (placeholder)
  - WithBatchSize: Install in configurable batches
  - WithPreInstallHook: Hook before installation
  - WithPostInstallHook: Hook after installation
- State keys: StatePackagesInstalled, StateInstalledPackages, StatePackageInstallTime
- Multi-distro support (Debian, RHEL, Arch, SUSE)
- Full rollback capability (removes installed packages)
- Batch installation support with cancellation checks
- Dry-run mode support
- Cancellation handling at multiple checkpoints

**Test Results:**
- `internal/install/steps`: 95.9% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 5 Sprint 5] Implement package installation step` - v5.5.0

**Notes:**
- Fifth installation step complete
- Core package installation functionality implemented
- Ready for P5-MS6 (DKMS Module Build Step)

---

#### Session 2026-01-04 - P5-MS6 Implementation

**Sprint:** P5-MS6
**Version:** 5.6.0
**Status:** COMPLETED

**Activities:**
- [x] Updated VERSION file to 5.6.0
- [x] Delegated implementation to code-implementator agent
- [x] Created internal/install/steps/dkms.go
- [x] Created internal/install/steps/dkms_test.go (55+ tests)
- [x] Code review by code-reviewer agent - APPROVED WITH CHANGES
- [x] Added input validation for module names, versions, kernel versions (security)
- [x] All tests passed with 93.7% coverage

**Features Implemented:**
- DKMSBuildStep implementing install.Step interface
- DKMS commands: status, build, install, remove
- Functional options: WithModuleName, WithModuleVersion, WithKernelVersion, WithSkipStatusCheck, WithKernelDetector, WithDKMSTimeout
- State keys: StateDKMSBuilt, StateDKMSModuleName, StateDKMSModuleVersion, StateDKMSKernelVersion, StateDKMSBuildTime
- Checks if DKMS is available, skips gracefully if not
- Auto-detects nvidia module version from dkms status
- Skips if module already built for current kernel
- Configurable timeout for long builds (default: 10 minutes)
- Full rollback capability (dkms remove)
- Input validation for security (module names, versions, kernel versions)
- Dry-run mode support
- Cancellation handling at multiple checkpoints

**Test Results:**
- `internal/install/steps`: 93.7% coverage
- `go build ./...`: PASS
- `go test ./...`: PASS
- `go vet ./...`: PASS

**Human Validation:** APPROVED

**Commits:**
- `[Phase 5 Sprint 6] Implement DKMS module build step` - v5.6.0

**Notes:**
- Sixth installation step complete
- DKMS module building with security validation
- Ready for P5-MS7 (Module Loading Step)

---

## 6. Git Workflow

### Repository Setup

```bash
# Initialize repository (done in P1-MS1)
git init
git remote add origin https://github.com/tungetti/igor.git
```

### Branch Strategy

| Branch | Purpose |
|--------|---------|
| `main` | Stable releases, sprint completions |
| `develop` | Integration (optional, for larger teams) |
| `feature/p{X}-ms{Y}` | Individual sprint work |

### Commit Message Format

```
[Phase X Sprint Y] Brief description

- Detailed change 1
- Detailed change 2
- Detailed change 3

Version: X.Y.Z
```

**Example:**
```
[Phase 1 Sprint 1] Initialize project structure

- Created Go module with all dependencies
- Set up directory structure per specification
- Added Makefile with build targets
- Created initial main.go with version info

Version: 1.1.0
```

### Tag Format

```
vX.Y.Z
```

**Examples:** `v1.1.0`, `v1.1.1`, `v2.1.0`

### Sprint Completion Workflow

```bash
# After human validation approved
git add -A
git commit -m "[Phase X Sprint Y] Description

- Change 1
- Change 2

Version: X.Y.Z"

git tag vX.Y.Z
git push origin main --tags
```

---

## 7. Appendices

### Appendix A: Quick Reference Commands

```bash
# Build
make build

# Test
make test

# Test with coverage
go test -cover ./...

# Test with race detection
go test -race ./...

# Vet
go vet ./...

# Run
./igor --help
./igor version
./igor detect
./igor install --dry-run

# Clean
make clean
```

### Appendix B: Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `IGOR_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `IGOR_LOG_FILE` | Log file path | `""` (disabled) |
| `IGOR_CONFIG` | Config file path | `~/.config/igor/config.yaml` |
| `IGOR_DRY_RUN` | Enable dry-run mode | `false` |

### Appendix C: Config File Format

```yaml
# ~/.config/igor/config.yaml
log_level: info
log_file: /var/log/igor/igor.log
dry_run: false
cache_dir: ~/.cache/igor
timeout: 300  # seconds
```

### Appendix D: Distribution Test Matrix

| Distribution | Version | Priority | Status |
|--------------|---------|----------|--------|
| Ubuntu | 24.04 LTS | P1 | `NOT_TESTED` |
| Ubuntu | 22.04 LTS | P1 | `NOT_TESTED` |
| Fedora | 40 | P1 | `NOT_TESTED` |
| Arch Linux | Rolling | P1 | `NOT_TESTED` |
| Debian | 12 | P2 | `NOT_TESTED` |
| openSUSE TW | Rolling | P2 | `NOT_TESTED` |
| Rocky Linux | 9 | P2 | `NOT_TESTED` |
| Linux Mint | 21.3 | P3 | `NOT_TESTED` |
| Pop!_OS | 22.04 | P3 | `NOT_TESTED` |
| Manjaro | Rolling | P3 | `NOT_TESTED` |
| AlmaLinux | 9 | P3 | `NOT_TESTED` |
| EndeavourOS | Rolling | P3 | `NOT_TESTED` |
| openSUSE Leap | 15.5 | P3 | `NOT_TESTED` |
| CentOS | 7 | P4 | `NOT_TESTED` |
| RHEL | 7 | P4 | `NOT_TESTED` |

### Appendix E: Risk Register

| Risk | Probability | Impact | Mitigation | Owner |
|------|-------------|--------|------------|-------|
| Package naming changes | Medium | High | Version-specific mappings | P2 |
| System-breaking install | Low | Critical | Dry-run, rollback | P5 |
| Secure Boot issues | Medium | Medium | Detection + docs | P3 |
| Test matrix explosion | Medium | Medium | Focus on LTS | P7 |
| Cross-compile issues | Low | Medium | Early CI testing | P1 |

---

## 8. Glossary

| Term | Definition |
|------|------------|
| **Sprint** | A single unit of work (microsprint) |
| **Phase** | A major milestone containing multiple sprints |
| **Iteration** | A code-review cycle within a sprint (z in version) |
| **TUI** | Terminal User Interface |
| **DKMS** | Dynamic Kernel Module Support |
| **PPA** | Personal Package Archive (Ubuntu) |
| **RPM Fusion** | Community RPM repository for Fedora/RHEL |
| **AUR** | Arch User Repository |
| **Nouveau** | Open-source NVIDIA driver |

---

## Document Metadata

| Property | Value |
|----------|-------|
| **Document Version** | 4.0.0 |
| **Created** | 2026-01-03 |
| **Last Updated** | 2026-01-04 |
| **Author** | OpenCode Assistant |
| **Status** | Active |
| **Project Version** | 5.10.0 |
| **Next Sprint** | P5-MS11 (Distribution-Specific Workflow Builders) |

---

## 9. Continuation Prompt

> **Use this section after context compaction to resume development.**

### Current State (Updated: 2026-01-04)

**Project:** Igor - NVIDIA Driver Installer TUI  
**Repository:** https://github.com/tungetti/igor.git  
**Working Directory:** `/home/tommasomariaungetti/Git/igor`  
**Module:** `github.com/tungetti/igor`  
**Go Version:** 1.21  
**Current Version:** 5.10.0

### Progress Summary

| Metric | Value |
|--------|-------|
| Total Commits | 48 |
| Total Tags | 46 (v1.1.0 - v5.10.0) |
| Phases Complete | 4 of 7 |
| Sprints Complete | 46 of 62 (74%) |

### Completed Phases

| Phase | Description | Sprints | Status |
|-------|-------------|---------|--------|
| Phase 1 | Project Foundation | 9/9 | COMPLETED (v1.1.0-v1.9.0) |
| Phase 2 | Distribution Detection & Package Manager | 9/9 | COMPLETED (v2.1.0-v2.9.0) |
| Phase 3 | GPU Detection & System Analysis | 7/7 | COMPLETED (v3.1.0-v3.7.0) |
| Phase 4 | TUI Framework & Core Views | 11/11 | COMPLETED (v4.1.0-v4.11.0) |

### Current Phase: Phase 5 - Installation Workflow Engine

**Status:** IN_PROGRESS (10 of 11 sprints complete)

| Sprint | Description | Status | Version |
|--------|-------------|--------|---------|
| P5-MS1 | Define Installation Workflow Interface | COMPLETED | v5.1.0 |
| P5-MS2 | Implement Pre-Installation Validation Step | COMPLETED | v5.2.0 |
| P5-MS3 | Implement Repository Configuration Step | COMPLETED | v5.3.0 |
| P5-MS4 | Implement Nouveau Blacklist Step | COMPLETED | v5.4.0 |
| P5-MS5 | Implement Package Installation Step | COMPLETED | v5.5.0 |
| P5-MS6 | Implement DKMS Module Build Step | COMPLETED | v5.6.0 |
| P5-MS7 | Implement Module Loading Step | COMPLETED | v5.7.0 |
| P5-MS8 | Implement X.org Configuration Step | COMPLETED | v5.8.0 |
| P5-MS9 | Implement Post-Installation Verification Step | COMPLETED | v5.9.0 |
| P5-MS10 | Implement Workflow Orchestrator | COMPLETED | v5.10.0 |
| P5-MS11 | Create Distribution-Specific Workflow Builders | **NEXT** | v5.11.0 |

### Sprint Pipeline (MUST FOLLOW)

For each sprint, follow this exact pipeline:

1. **PRE-SPRINT:**
   - Update VERSION file to new version
   - Mark sprint as `IN_PROGRESS` in IGOR_PROJECT.md

2. **IMPLEMENT:**
   - Delegate to `code-implementator` subagent with detailed spec
   - Include file paths, interfaces, and test requirements

3. **REVIEW:**
   - Run `go build ./...` and `go vet ./...`
   - Delegate to `code-reviewer` subagent
   - Fix any issues found

4. **TEST:**
   - Run `go test -cover ./...`
   - Target >85% coverage for new packages

5. **VALIDATE:**
   - Present summary to human with coverage stats
   - Wait for explicit "yes" approval

6. **FINALIZE:**
   - Update CHANGELOG.md with new version entry
   - Mark sprint as `COMPLETED` in IGOR_PROJECT.md
   - `git add -A && git commit -m "[Phase X Sprint Y] Description"`
   - `git tag -a vX.Y.0 -m "vX.Y.0 - Description"`
   - `git push && git push --tags`

### Key Package Structure

```
internal/
├── install/                    # Installation workflow (Phase 5)
│   ├── types.go               # StepStatus, WorkflowStatus, StepResult
│   ├── step.go                # Step interface, BaseStep, FuncStep
│   ├── workflow.go            # Workflow interface, BaseWorkflow
│   ├── context.go             # Installation context with state
│   └── steps/                 # Concrete step implementations
│       ├── validation.go      # Pre-installation validation step
│       ├── repository.go      # Repository configuration step
│       ├── nouveau.go         # Nouveau blacklist step
│       ├── packages.go        # Package installation step
│       └── dkms.go            # DKMS module build step
├── ui/                        # TUI framework (Phase 4)
│   ├── app.go                 # Main model with navigation state machine
│   ├── messages.go            # Message types
│   ├── keys.go                # Key bindings
│   ├── theme/                 # Theme and styles
│   ├── components/            # Reusable UI components
│   └── views/                 # Screen views (welcome, detection, etc.)
├── gpu/                       # GPU detection (Phase 3)
│   ├── orchestrator.go        # Main GPU orchestrator
│   ├── pci/                   # PCI device scanning
│   ├── nvidia/                # NVIDIA model database
│   ├── smi/                   # nvidia-smi parser
│   ├── nouveau/               # Nouveau driver detection
│   ├── kernel/                # Kernel module detection
│   └── validator/             # System validation checks
├── pkg/                       # Package managers (Phase 2)
│   ├── manager.go             # Manager interface
│   ├── factory/               # Factory for creating managers
│   ├── apt/, dnf/, yum/       # Distribution-specific managers
│   ├── pacman/, zypper/
│   └── nvidia/                # NVIDIA package mappings
├── distro/                    # Distribution detection
├── config/                    # Configuration management
├── logging/                   # Logging interface
├── exec/                      # Command execution
├── errors/                    # Custom error types
├── constants/                 # Project constants
├── privilege/                 # Privilege escalation
└── app/                       # Application lifecycle
```

### Build & Test Commands

```bash
cd /home/tommasomariaungetti/Git/igor

# Build
go build ./...

# Vet
go vet ./...

# Test with coverage
go test -cover ./...

# Test specific package
go test -cover ./internal/install/...

# Run
./igor version
```

### Resume Instructions

1. Read this file (IGOR_PROJECT.md) to understand current state
2. Check `cat VERSION` to confirm current version (should be 5.6.0)
3. Continue with next sprint: **P5-MS7 (Module Loading Step)**
4. Follow the Sprint Pipeline exactly as documented above
5. After each sprint, get human approval before committing

### Important Notes

- Always get human approval with explicit "yes" before committing
- Use subagents: `code-implementator` for implementation, `code-reviewer` for review
- Target >85% test coverage for each package
- Update CHANGELOG.md after each sprint
- Follow existing patterns in the codebase

### Implemented Steps Reference (for patterns)

Each step follows the same pattern. Use these as reference when implementing new steps:

1. **ValidationStep** (`validation.go`) - Validates system requirements
2. **RepositoryStep** (`repository.go`) - Configures NVIDIA repositories
3. **NouveauBlacklistStep** (`nouveau.go`) - Blacklists Nouveau driver, regenerates initramfs
4. **PackageInstallationStep** (`packages.go`) - Installs NVIDIA packages via package manager
5. **DKMSBuildStep** (`dkms.go`) - Builds kernel modules via DKMS

**Common Step Pattern:**
```go
// State keys
const (
    State<Step>Done = "<step>_done"
    // ... other state keys
)

// Step struct
type <Step>Step struct {
    install.BaseStep
    // step-specific fields
}

// Functional options
type <Step>StepOption func(*<Step>Step)
func With<Option>(...) <Step>StepOption { ... }

// Constructor
func New<Step>Step(opts ...<Step>StepOption) *<Step>Step { ... }

// Execute - main logic with cancellation checks, dry-run, state storage
func (s *<Step>Step) Execute(ctx *install.Context) install.StepResult { ... }

// Rollback - undo changes
func (s *<Step>Step) Rollback(ctx *install.Context) error { ... }

// Validate - check prerequisites
func (s *<Step>Step) Validate(ctx *install.Context) error { ... }

// CanRollback - usually returns true
func (s *<Step>Step) CanRollback() bool { return true }

// Interface compliance
var _ install.Step = (*<Step>Step)(nil)
```

### Remaining Phase 5 Sprints

| Sprint | Description | Key Details |
|--------|-------------|-------------|
| P5-MS7 | Module Loading Step | Load nvidia kernel module via `modprobe nvidia`, unload via `modprobe -r nvidia` |
| P5-MS8 | X.org Configuration Step | Configure `/etc/X11/xorg.conf.d/` for NVIDIA, handle Wayland |
| P5-MS9 | Post-Installation Verification | Verify driver via `nvidia-smi`, check module loaded, validate X.org |
| P5-MS10 | Workflow Orchestrator | Execute steps in order, handle rollback on failure, progress tracking |
| P5-MS11 | Distribution-Specific Builders | Create workflows per distro family with appropriate steps |

---

*End of Document*
