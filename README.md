# engine <!-- omit in toc -->

<div align="center">

<!-- BEGIN_KICKR_BADGES -->
[![GitLab Release](https://img.shields.io/gitlab/v/release/kickr-dev%2Fengine?gitlab_url=https%3A%2F%2Fgitlab.com&include_prereleases&sort=semver&style=for-the-badge)](https://gitlab.com/kickr-dev/engine/-/releases)
[![GitLab Issues](https://img.shields.io/gitlab/issues/open/kickr-dev%2Fengine?gitlab_url=https%3A%2F%2Fgitlab.com&style=for-the-badge)](https://gitlab.com/kickr-dev/engine/-/work_items)
[![GitLab License](https://img.shields.io/gitlab/license/kickr-dev%2Fengine?gitlab_url=https%3A%2F%2Fgitlab.com&style=for-the-badge)](https://gitlab.com/kickr-dev/engine/-/blob/HEAD/LICENSE)
[![GitLab CICD](https://img.shields.io/gitlab/pipeline-status/kickr-dev%2Fengine?gitlab_url=https%3A%2F%2Fgitlab.com&branch=main&style=for-the-badge)](https://gitlab.com/kickr-dev/engine/-/pipelines?ref=main)
[![Go Version](https://img.shields.io/gitlab/go-mod/go-version/kickr-dev/engine?style=for-the-badge)](https://gitlab.com/kickr-dev/engine/-/blob/HEAD/go.mod)
[![Plumber Score](https://img.shields.io/endpoint?url=https%3A%2F%2Fscore.getplumber.io%2Fgitlab.com%2Fkickr-dev%2Fengine.json&style=for-the-badge)](https://score.getplumber.io/gitlab.com/kickr-dev/engine)
<!-- END_KICKR_BADGES -->

</div>

---

- [How to use?](#how-to-use)
- [Initialize (pkg.go.dev)](#initialize-pkggodev)
- [Generate (pkg.go.dev)](#generate-pkggodev)
  - [Templates](#templates)
- [Helpers](#helpers)

A Go library to build projects scaffolding CLIs on three main features:
- [**Initialization**](#initialize-pkggodev): interactive user configuration with [**huh**](https://github.com/charmbracelet/huh) (*e.g.* license choice, maintainers list, etc.)
- [**Parsing**](#helpers): repository languages parsing
- [**Generation**](#generate-pkggodev): repository scaffolding with files generation based on Go template

## How to use?

```sh
go get -u github.com/kickr-dev/engine@latest
```

## Initialize ([pkg.go.dev](https://pkg.go.dev/github.com/kickr-dev/engine/pkg))

Runs an interactive form with [**huh**](https://github.com/charmbracelet/huh)
and alters the created configuration gradually with each `huh.Group`.

```mermaid
flowchart LR
    subgraph Groups["Form pages (each a set of related questions)"]
        direction TB
        G1["Page 1"] --> G2["Page 2"] --> Gn["Page N"]
    end

    Groups --> Form["Multi-step interactive form"]
    Form -->|"user answers each page"| Config["filled config"]
```

## Generate ([pkg.go.dev](https://pkg.go.dev/github.com/kickr-dev/engine/pkg))

Parses a repository with all its configured parsers
and then runs all generators to create the appropriate files regarding the repository configuration, technologies, languages, etc.

```mermaid
flowchart TB
    CFG["Repository config"] --> Parsing

    subgraph Parsing["Detect repository info (in order)"]
        direction LR
        P1["Parser 1"] --> P2["Parser 2"] --> Pn["Parser N"]
    end

    Parsing --> CHK{"Something failed?"}
    CHK -->|yes| ERR["Stop, nothing generated"]
    CHK -->|no| Generation

    subgraph Generation["Generate files (concurrently)"]
        direction LR
        G1["Generator 1"] ~~~ G2["Generator 2"] ~~~ Gn["Generator N"]
    end

    Generation --> CHK2{"Something failed?"}
    CHK2 -->|yes| ERRF["Report what failed"]
    CHK2 -->|no| OK["Repository generated"]
```

### Templates

Each file has a policy. It can be rewritten every time, removed or left alone,
but by default it's rewritten only while nobody has edited it by hand.

```mermaid
flowchart TB
    File["File to generate"] --> Policy{"What does its policy say?"}

    Policy -->|remove| Remove["Delete the file"]
    Policy -->|skip| Untouched["Leave the file alone"]
    Policy -->|always| Write
    Policy -->|default| Owned{"Forced, or file<br/>never edited by hand?"}

    Owned -->|yes| Write["Write the file from its templates"]
    Owned -->|no| Keep["Keep the user's file,<br/>refresh only its marked sections"]

    Write --> Patch["Apply patches on top"]
    Keep --> Patch
```

## Helpers

- [`pkg/files`](https://pkg.go.dev/github.com/kickr-dev/engine/pkg/files):
  read, write, validate (JSON schema) and glob files.
- [`pkg/generator`](https://pkg.go.dev/github.com/kickr-dev/engine/pkg/generator):
  fetch `.gitignore`, code of conduct and pinned container images.
- [`pkg/parser`](https://pkg.go.dev/github.com/kickr-dev/engine/pkg/parser):
  detect Git, Go (`go.mod`, `go.work`, `cmd`), Hugo and `package.json`.
