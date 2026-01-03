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

### 1.4 Project Structure

```
igor/
├── cmd/igor/                    # Main entry point
│   ├── main.go
│   ├── cli.go
│   └── version.go
├── internal/
│   ├── app/                     # Application lifecycle
│   │   ├── app.go
│   │   ├── container.go
│   │   └── lifecycle.go
│   ├── cli/                     # CLI parsing
│   │   ├── parser.go
│   │   ├── commands.go
│   │   └── flags.go
│   ├── config/                  # Configuration management
│   │   ├── config.go
│   │   ├── loader.go
│   │   ├── validator.go
│   │   └── defaults.go
│   ├── constants/               # Application constants
│   │   └── constants.go
│   ├── distro/                  # Distribution detection
│   │   ├── distro.go
│   │   ├── detect.go
│   │   └── types.go
│   ├── errors/                  # Custom error types
│   │   └── errors.go
│   ├── exec/                    # Command execution
│   │   ├── executor.go
│   │   ├── mock.go
│   │   └── output.go
│   ├── gpu/                     # GPU detection
│   │   ├── detector.go
│   │   ├── report.go
│   │   ├── pci/
│   │   │   ├── scanner.go
│   │   │   ├── device.go
│   │   │   └── sysfs.go
│   │   ├── nvidia/
│   │   │   ├── database.go
│   │   │   ├── smi.go
│   │   │   └── smi_parser.go
│   │   ├── nouveau/
│   │   │   └── detector.go
│   │   ├── kernel/
│   │   │   ├── kernel.go
│   │   │   ├── modules.go
│   │   │   └── dkms.go
│   │   └── requirements/
│   │       ├── validator.go
│   │       ├── checks.go
│   │       └── report.go
│   ├── install/                 # Installation orchestration
│   │   ├── workflow.go
│   │   ├── step.go
│   │   ├── context.go
│   │   ├── orchestrator.go
│   │   ├── steps/
│   │   │   ├── validate.go
│   │   │   ├── repository.go
│   │   │   ├── nouveau.go
│   │   │   ├── install.go
│   │   │   ├── dkms.go
│   │   │   ├── modprobe.go
│   │   │   ├── xorg.go
│   │   │   └── verify.go
│   │   └── builders/
│   │       ├── debian.go
│   │       ├── rhel.go
│   │       ├── arch.go
│   │       └── suse.go
│   ├── logging/                 # Logging infrastructure
│   │   ├── logger.go
│   │   └── levels.go
│   ├── pkg/                     # Package manager abstraction
│   │   ├── manager.go
│   │   ├── types.go
│   │   ├── factory.go
│   │   ├── apt/
│   │   │   ├── apt.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── dnf/
│   │   │   ├── dnf.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── yum/
│   │   │   ├── yum.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── pacman/
│   │   │   ├── pacman.go
│   │   │   ├── parser.go
│   │   │   └── repository.go
│   │   ├── zypper/
│   │   │   ├── zypper.go
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
│   │   ├── sudo.go
│   │   └── pkexec.go
│   ├── recovery/                # Recovery mode
│   │   ├── recovery.go
│   │   └── minimal.go
│   ├── testing/                 # Test utilities
│   │   ├── mocks/
│   │   ├── fixtures/
│   │   └── helpers.go
│   ├── ui/                      # TUI components
│   │   ├── app.go
│   │   ├── model.go
│   │   ├── update.go
│   │   ├── view.go
│   │   ├── messages.go
│   │   ├── navigation.go
│   │   ├── theme/
│   │   │   ├── theme.go
│   │   │   ├── colors.go
│   │   │   ├── styles.go
│   │   │   └── components.go
│   │   ├── components/
│   │   │   ├── header.go
│   │   │   ├── footer.go
│   │   │   ├── statusbar.go
│   │   │   ├── dialog.go
│   │   │   └── keyhints.go
│   │   └── views/
│   │       ├── welcome/
│   │       ├── detection/
│   │       ├── driverselect/
│   │       ├── confirm/
│   │       ├── progress/
│   │       ├── complete/
│   │       ├── error/
│   │       └── uninstall/
│   └── uninstall/               # Uninstallation logic
│       ├── workflow.go
│       ├── orchestrator.go
│       └── steps/
│           ├── discover.go
│           ├── remove.go
│           ├── modules.go
│           ├── config.go
│           └── fallback.go
├── pkg/                         # Public reusable packages
├── scripts/
│   ├── build.sh
│   └── test.sh
├── docs/
│   └── templates/
├── .gitignore
├── go.mod
├── go.sum
├── Makefile
├── VERSION
├── CHANGELOG.md
├── LICENSE
├── README.md
└── IGOR_PROJECT.md              # This file
```

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

**Status:** `NOT_STARTED`
**Objective:** Create robust distribution detection and package manager abstraction for all target distributions.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P2-MS1 | Define Package Manager Interface | `COMPLETED` | 2.1.0 | P1-MS8 | Small |
| P2-MS2 | Implement Distribution Detection Core | `NOT_STARTED` | 2.2.0 | P1-MS8, P1-MS3 | Medium |
| P2-MS3 | Implement APT Package Manager (Debian/Ubuntu) | `NOT_STARTED` | 2.3.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS4 | Implement DNF Package Manager (Fedora/RHEL) | `NOT_STARTED` | 2.4.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS5 | Implement YUM Package Manager (CentOS 7/RHEL 7) | `NOT_STARTED` | 2.5.0 | P2-MS1, P2-MS2, P1-MS7 | Medium |
| P2-MS6 | Implement Pacman Package Manager (Arch) | `NOT_STARTED` | 2.6.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS7 | Implement Zypper Package Manager (openSUSE) | `NOT_STARTED` | 2.7.0 | P2-MS1, P2-MS2, P1-MS7 | Large |
| P2-MS8 | Create Package Manager Factory | `NOT_STARTED` | 2.8.0 | P2-MS2 through P2-MS7 | Small |
| P2-MS9 | Create Distribution-Specific NVIDIA Package Mappings | `NOT_STARTED` | 2.9.0 | P2-MS8 | Medium |

### Phase 3: GPU Detection & System Analysis

**Status:** `NOT_STARTED`
**Objective:** Implement comprehensive GPU detection, driver status checking, and system compatibility analysis.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P3-MS1 | Implement PCI Device Scanner | `NOT_STARTED` | 3.1.0 | P1-MS8 | Medium |
| P3-MS2 | Create NVIDIA GPU Database | `NOT_STARTED` | 3.2.0 | P3-MS1 | Medium |
| P3-MS3 | Implement nvidia-smi Parser | `NOT_STARTED` | 3.3.0 | P1-MS8, P3-MS2 | Medium |
| P3-MS4 | Create Nouveau Driver Detector | `NOT_STARTED` | 3.4.0 | P3-MS1, P1-MS8 | Small |
| P3-MS5 | Implement Kernel Version and Module Detection | `NOT_STARTED` | 3.5.0 | P1-MS8, P2-MS8 | Medium |
| P3-MS6 | Create System Requirements Validator | `NOT_STARTED` | 3.6.0 | P3-MS5, P2-MS8 | Medium |
| P3-MS7 | Build GPU Detection Orchestrator | `NOT_STARTED` | 3.7.0 | P3-MS1 through P3-MS6 | Medium |

### Phase 4: TUI Framework & Core Views

**Status:** `NOT_STARTED`
**Objective:** Build the Bubble Tea-based TUI application with all core views and navigation.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P4-MS1 | Create Base TUI Application Structure | `NOT_STARTED` | 4.1.0 | P1-MS2, P1-MS9 | Medium |
| P4-MS2 | Implement Theme and Styling System | `NOT_STARTED` | 4.2.0 | P1-MS2 | Medium |
| P4-MS3 | Create Common UI Components | `NOT_STARTED` | 4.3.0 | P4-MS2 | Medium |
| P4-MS4 | Implement Welcome Screen View | `NOT_STARTED` | 4.4.0 | P4-MS1, P4-MS2, P4-MS3 | Small |
| P4-MS5 | Implement System Detection View | `NOT_STARTED` | 4.5.0 | P4-MS1, P4-MS3, P3-MS7 | Medium |
| P4-MS6 | Implement Driver Selection View | `NOT_STARTED` | 4.6.0 | P4-MS1, P4-MS3, P3-MS7, P2-MS9 | Large |
| P4-MS7 | Implement Installation Confirmation View | `NOT_STARTED` | 4.7.0 | P4-MS1, P4-MS3, P4-MS6 | Medium |
| P4-MS8 | Implement Installation Progress View | `NOT_STARTED` | 4.8.0 | P4-MS1, P4-MS3 | Large |
| P4-MS9 | Implement Completion/Result View | `NOT_STARTED` | 4.9.0 | P4-MS1, P4-MS3 | Small |
| P4-MS10 | Implement Error/Help View | `NOT_STARTED` | 4.10.0 | P4-MS1, P4-MS3 | Small |
| P4-MS11 | Implement View Navigation and State Machine | `NOT_STARTED` | 4.11.0 | P4-MS4 through P4-MS10 | Medium |

### Phase 5: Installation Workflow Engine

**Status:** `NOT_STARTED`
**Objective:** Implement core installation orchestration with steps, rollback, and progress tracking.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P5-MS1 | Define Installation Workflow Interface | `NOT_STARTED` | 5.1.0 | P2-MS8, P3-MS7 | Medium |
| P5-MS2 | Implement Pre-Installation Validation Step | `NOT_STARTED` | 5.2.0 | P5-MS1, P3-MS6 | Small |
| P5-MS3 | Implement Repository Configuration Step | `NOT_STARTED` | 5.3.0 | P5-MS1, P2-MS3 through P2-MS7 | Medium |
| P5-MS4 | Implement Nouveau Blacklist Step | `NOT_STARTED` | 5.4.0 | P5-MS1, P3-MS4 | Small |
| P5-MS5 | Implement Package Installation Step | `NOT_STARTED` | 5.5.0 | P5-MS1, P2-MS8, P2-MS9 | Large |
| P5-MS6 | Implement DKMS Module Build Step | `NOT_STARTED` | 5.6.0 | P5-MS1, P3-MS5 | Medium |
| P5-MS7 | Implement Module Loading Step | `NOT_STARTED` | 5.7.0 | P5-MS1, P3-MS5 | Small |
| P5-MS8 | Implement X.org Configuration Step | `NOT_STARTED` | 5.8.0 | P5-MS1 | Medium |
| P5-MS9 | Implement Post-Installation Verification Step | `NOT_STARTED` | 5.9.0 | P5-MS1, P3-MS3 | Small |
| P5-MS10 | Implement Workflow Orchestrator | `NOT_STARTED` | 5.10.0 | P5-MS2 through P5-MS9 | Large |
| P5-MS11 | Create Distribution-Specific Workflow Builders | `NOT_STARTED` | 5.11.0 | P5-MS10, P2-MS2 | Medium |

### Phase 6: Uninstallation & Recovery

**Status:** `NOT_STARTED`
**Objective:** Implement safe uninstallation and recovery mechanisms.

| Sprint ID | Title | Status | Version | Dependencies | Effort |
|-----------|-------|--------|---------|--------------|--------|
| P6-MS1 | Implement Uninstallation Workflow Interface | `NOT_STARTED` | 6.1.0 | P5-MS1 | Small |
| P6-MS2 | Implement Installed Package Discovery | `NOT_STARTED` | 6.2.0 | P6-MS1, P2-MS8 | Medium |
| P6-MS3 | Implement Package Removal Step | `NOT_STARTED` | 6.3.0 | P6-MS1, P6-MS2 | Medium |
| P6-MS4 | Implement Kernel Module Cleanup Step | `NOT_STARTED` | 6.4.0 | P6-MS1, P3-MS5 | Small |
| P6-MS5 | Implement Configuration Cleanup Step | `NOT_STARTED` | 6.5.0 | P6-MS1 | Small |
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

**Status:** `NOT_STARTED`
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

**Status:** `NOT_STARTED`
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

**Status:** `NOT_STARTED`
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

**Status:** `NOT_STARTED`
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

**Status:** `NOT_STARTED`
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

**Status:** `NOT_STARTED`
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
| **Document Version** | 1.0.0 |
| **Created** | 2026-01-03 |
| **Last Updated** | 2026-01-03 |
| **Author** | OpenCode Assistant |
| **Status** | Active |
| **Next Review** | After Phase 1 completion |

---

*End of Document*
