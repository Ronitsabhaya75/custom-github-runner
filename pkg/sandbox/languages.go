package sandbox

import (
	"strings"
)

// Language represents a target programming language runtime env
type Language string

const (
	// Tier 1 — Most popular languages (highest priority detection)
	Go         Language = "go"
	Python     Language = "python"
	Node       Language = "node"
	Rust       Language = "rust"
	Java       Language = "java"
	Typescript Language = "typescript"

	// Tier 2 — Widely used compiled and scripted languages
	Csharp Language = "csharp"
	Cpp    Language = "cpp"
	Ruby   Language = "ruby"
	PHP    Language = "php"
	Swift  Language = "swift"
	Kotlin Language = "kotlin"
	Scala  Language = "scala"

	// Tier 3 — Emerging, niche, and systems-level languages
	Dart    Language = "dart"
	Elixir  Language = "elixir"
	Haskell Language = "haskell"
	Clojure Language = "clojure"
	Perl    Language = "perl"
	Lua     Language = "lua"
	Zig     Language = "zig"
	Nim     Language = "nim"
	OCaml   Language = "ocaml"
	Julia   Language = "julia"
	R       Language = "r"
	Groovy  Language = "groovy"
	Erlang  Language = "erlang"
	Fortran Language = "fortran"

	// Tier 4 — Infrastructure / DevOps / Scripting
	Shell      Language = "shell"
	Terraform  Language = "terraform"
	Ansible    Language = "ansible"
	Powershell Language = "powershell"

	// Fallback
	Lightweight Language = "generic"
)

// RuntimeConfig describes the optimal container setup for a given language
type RuntimeConfig struct {
	Image        string   // Lightweight OCI container image
	SizeMB       int      // Approximate pulled image size
	CacheDir     string   // Where this language stores dependency caches
	PackageCmd   string   // How to install dependencies (for auto-caching)
	LockFiles    []string // Files that indicate dependency fingerprint
	BuildCmd     string   // Standard build command
	TestCmd      string   // Standard test command
	MinMemoryMB  int64    // Recommended minimum memory
	NeedsNetwork bool     // Whether builds typically need network for deps
}

// LanguageRegistry is the complete mapping of every supported language to its optimal runtime
var LanguageRegistry = map[Language]RuntimeConfig{
	// ===== Tier 1 =====
	Go: {
		Image: "golang:1.22-alpine", SizeMB: 78,
		CacheDir: "/go/pkg/mod", PackageCmd: "go mod download",
		LockFiles: []string{"go.sum"}, BuildCmd: "go build ./...", TestCmd: "go test ./...",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Python: {
		Image: "python:3.12-alpine", SizeMB: 38,
		CacheDir: "/root/.cache/pip", PackageCmd: "pip install -r requirements.txt",
		LockFiles: []string{"requirements.txt", "Pipfile.lock", "poetry.lock"},
		BuildCmd: "python setup.py build", TestCmd: "pytest",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	Node: {
		Image: "node:20-alpine", SizeMB: 42,
		CacheDir: "/root/.npm", PackageCmd: "npm ci",
		LockFiles: []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml"},
		BuildCmd: "npm run build", TestCmd: "npm test",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Typescript: {
		Image: "node:20-alpine", SizeMB: 42,
		CacheDir: "/root/.npm", PackageCmd: "npm ci",
		LockFiles: []string{"package-lock.json", "tsconfig.json"},
		BuildCmd: "npx tsc", TestCmd: "npm test",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Rust: {
		Image: "rust:1.77-alpine", SizeMB: 80,
		CacheDir: "/usr/local/cargo/registry", PackageCmd: "cargo fetch",
		LockFiles: []string{"Cargo.lock"}, BuildCmd: "cargo build --release", TestCmd: "cargo test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Java: {
		Image: "eclipse-temurin:21-alpine", SizeMB: 85,
		CacheDir: "/root/.m2/repository", PackageCmd: "mvn dependency:resolve",
		LockFiles: []string{"pom.xml", "build.gradle", "build.gradle.kts"},
		BuildCmd: "mvn package -DskipTests", TestCmd: "mvn test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},

	// ===== Tier 2 =====
	Csharp: {
		Image: "mcr.microsoft.com/dotnet/sdk:8.0-alpine", SizeMB: 110,
		CacheDir: "/root/.nuget/packages", PackageCmd: "dotnet restore",
		LockFiles: []string{"*.csproj", "packages.lock.json"},
		BuildCmd: "dotnet build", TestCmd: "dotnet test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Cpp: {
		Image: "alpine:latest", SizeMB: 5,
		CacheDir: "", PackageCmd: "apk add --no-cache g++ cmake make",
		LockFiles: []string{"CMakeLists.txt", "Makefile"},
		BuildCmd: "cmake --build .", TestCmd: "ctest",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Ruby: {
		Image: "ruby:3.3-alpine", SizeMB: 35,
		CacheDir: "/usr/local/bundle", PackageCmd: "bundle install",
		LockFiles: []string{"Gemfile.lock"}, BuildCmd: "rake build", TestCmd: "rake test",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	PHP: {
		Image: "php:8.3-alpine", SizeMB: 30,
		CacheDir: "/root/.composer/cache", PackageCmd: "composer install",
		LockFiles: []string{"composer.lock"}, BuildCmd: "php artisan build", TestCmd: "phpunit",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	Swift: {
		Image: "swift:5.10", SizeMB: 350,
		CacheDir: "/root/.swiftpm", PackageCmd: "swift package resolve",
		LockFiles: []string{"Package.resolved"}, BuildCmd: "swift build", TestCmd: "swift test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Kotlin: {
		Image: "eclipse-temurin:21-alpine", SizeMB: 85,
		CacheDir: "/root/.gradle/caches", PackageCmd: "gradle dependencies",
		LockFiles: []string{"build.gradle.kts", "gradle.lockfile"},
		BuildCmd: "gradle build", TestCmd: "gradle test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Scala: {
		Image: "eclipse-temurin:21-alpine", SizeMB: 85,
		CacheDir: "/root/.cache/coursier", PackageCmd: "sbt update",
		LockFiles: []string{"build.sbt"}, BuildCmd: "sbt compile", TestCmd: "sbt test",
		MinMemoryMB: 768, NeedsNetwork: true,
	},

	// ===== Tier 3 =====
	Dart: {
		Image: "dart:stable", SizeMB: 120,
		CacheDir: "/root/.pub-cache", PackageCmd: "dart pub get",
		LockFiles: []string{"pubspec.lock"}, BuildCmd: "dart compile exe", TestCmd: "dart test",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Elixir: {
		Image: "elixir:1.16-alpine", SizeMB: 45,
		CacheDir: "/root/.mix", PackageCmd: "mix deps.get",
		LockFiles: []string{"mix.lock"}, BuildCmd: "mix compile", TestCmd: "mix test",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Haskell: {
		Image: "haskell:9.8", SizeMB: 250,
		CacheDir: "/root/.cabal/store", PackageCmd: "cabal update && cabal build --only-dependencies",
		LockFiles: []string{"cabal.project.freeze"}, BuildCmd: "cabal build", TestCmd: "cabal test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Clojure: {
		Image: "clojure:temurin-21-alpine", SizeMB: 90,
		CacheDir: "/root/.m2/repository", PackageCmd: "clojure -P",
		LockFiles: []string{"deps.edn"}, BuildCmd: "clojure -M:build", TestCmd: "clojure -M:test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Perl: {
		Image: "perl:5.38-slim", SizeMB: 40,
		CacheDir: "/root/.cpanm", PackageCmd: "cpanm --installdeps .",
		LockFiles: []string{"cpanfile.snapshot"}, BuildCmd: "perl Makefile.PL && make", TestCmd: "prove -l t/",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	Lua: {
		Image: "alpine:latest", SizeMB: 5,
		CacheDir: "/usr/local/lib/luarocks", PackageCmd: "apk add --no-cache lua5.4 luarocks",
		LockFiles: []string{}, BuildCmd: "lua main.lua", TestCmd: "busted",
		MinMemoryMB: 64, NeedsNetwork: true,
	},
	Zig: {
		Image: "alpine:latest", SizeMB: 5,
		CacheDir: "/root/.cache/zig", PackageCmd: "",
		LockFiles: []string{"build.zig.zon"}, BuildCmd: "zig build", TestCmd: "zig build test",
		MinMemoryMB: 256, NeedsNetwork: false,
	},
	Nim: {
		Image: "nimlang/nim:alpine", SizeMB: 60,
		CacheDir: "/root/.nimble/pkgs", PackageCmd: "nimble install -y",
		LockFiles: []string{"*.nimble"}, BuildCmd: "nim c -d:release main.nim", TestCmd: "nimble test",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	OCaml: {
		Image: "ocaml/opam:alpine", SizeMB: 80,
		CacheDir: "/root/.opam", PackageCmd: "opam install . --deps-only -y",
		LockFiles: []string{"*.opam", "dune-project"}, BuildCmd: "dune build", TestCmd: "dune test",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Julia: {
		Image: "julia:1.10-alpine", SizeMB: 90,
		CacheDir: "/root/.julia/packages", PackageCmd: "julia -e 'using Pkg; Pkg.instantiate()'",
		LockFiles: []string{"Manifest.toml"}, BuildCmd: "", TestCmd: "julia -e 'using Pkg; Pkg.test()'",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	R: {
		Image: "r-base:4.3.2", SizeMB: 100,
		CacheDir: "/usr/local/lib/R/site-library", PackageCmd: "Rscript -e 'install.packages(\"renv\"); renv::restore()'",
		LockFiles: []string{"renv.lock"}, BuildCmd: "R CMD build .", TestCmd: "R CMD check .",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Groovy: {
		Image: "eclipse-temurin:21-alpine", SizeMB: 85,
		CacheDir: "/root/.gradle/caches", PackageCmd: "gradle dependencies",
		LockFiles: []string{"build.gradle"}, BuildCmd: "gradle build", TestCmd: "gradle test",
		MinMemoryMB: 512, NeedsNetwork: true,
	},
	Erlang: {
		Image: "erlang:26-alpine", SizeMB: 55,
		CacheDir: "/root/.cache/rebar3", PackageCmd: "rebar3 get-deps",
		LockFiles: []string{"rebar.lock"}, BuildCmd: "rebar3 compile", TestCmd: "rebar3 eunit",
		MinMemoryMB: 256, NeedsNetwork: true,
	},
	Fortran: {
		Image: "alpine:latest", SizeMB: 5,
		CacheDir: "", PackageCmd: "apk add --no-cache gfortran",
		LockFiles: []string{}, BuildCmd: "gfortran -o main main.f90", TestCmd: "./main",
		MinMemoryMB: 128, NeedsNetwork: true,
	},

	// ===== Tier 4 — Infrastructure =====
	Shell: {
		Image: "alpine:latest", SizeMB: 5,
		CacheDir: "", PackageCmd: "",
		LockFiles: []string{}, BuildCmd: "", TestCmd: "shellcheck *.sh",
		MinMemoryMB: 64, NeedsNetwork: false,
	},
	Terraform: {
		Image: "hashicorp/terraform:1.7", SizeMB: 50,
		CacheDir: "/root/.terraform.d/plugin-cache", PackageCmd: "terraform init",
		LockFiles: []string{".terraform.lock.hcl"}, BuildCmd: "terraform plan", TestCmd: "terraform validate",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	Ansible: {
		Image: "python:3.12-alpine", SizeMB: 38,
		CacheDir: "", PackageCmd: "pip install ansible",
		LockFiles: []string{"requirements.yml"}, BuildCmd: "", TestCmd: "ansible-lint",
		MinMemoryMB: 128, NeedsNetwork: true,
	},
	Powershell: {
		Image: "mcr.microsoft.com/powershell:lts-alpine-3.17", SizeMB: 65,
		CacheDir: "/root/.local/share/powershell/Modules", PackageCmd: "",
		LockFiles: []string{}, BuildCmd: "", TestCmd: "pwsh -Command Invoke-Pester",
		MinMemoryMB: 128, NeedsNetwork: false,
	},

	// ===== Fallback =====
	Lightweight: {
		Image: "alpine:latest", SizeMB: 5,
		CacheDir: "", PackageCmd: "",
		LockFiles: []string{}, BuildCmd: "", TestCmd: "",
		MinMemoryMB: 64, NeedsNetwork: false,
	},
}

// ResolveRuntimeImage returns the ultra-lightweight OCI image for the requested language
func ResolveRuntimeImage(lang Language) string {
	if cfg, ok := LanguageRegistry[lang]; ok {
		return cfg.Image
	}
	return "alpine:latest"
}

// ResolveRuntimeConfig returns the full runtime configuration for a language
func ResolveRuntimeConfig(lang Language) RuntimeConfig {
	if cfg, ok := LanguageRegistry[lang]; ok {
		return cfg
	}
	return LanguageRegistry[Lightweight]
}

// DetectLanguageFromCommands analyzes the job step commands to auto-detect the target language environment
func DetectLanguageFromCommands(commands []string) Language {
	for _, cmd := range commands {
		c := strings.ToLower(cmd)

		// Tier 1
		if containsAny(c, "go run ", "go test ", "go build ", "go mod ") {
			return Go
		}
		if containsAny(c, "python ", "python3 ", "pip install ", "pip3 ", "poetry ", "pipenv ") {
			return Python
		}
		if containsAny(c, "node ", "npm ", "yarn ", "pnpm ", "npx ") {
			return Node
		}
		if containsAny(c, "tsc ", "ts-node ", "tsconfig") {
			return Typescript
		}
		if containsAny(c, "cargo ", "rustc ", "rustup ") {
			return Rust
		}
		if containsAny(c, "java ", "javac ", "mvn ", "maven ", "gradle ") {
			// Differentiate Java vs Kotlin vs Scala vs Groovy
			if containsAny(c, ".kts", "kotlin") {
				return Kotlin
			}
			if containsAny(c, "sbt ", "scala") {
				return Scala
			}
			if containsAny(c, "groovy") {
				return Groovy
			}
			return Java
		}

		// Tier 2
		if containsAny(c, "dotnet ", "nuget ", "csproj") {
			return Csharp
		}
		if containsAny(c, "g++ ", "gcc ", "cmake ", "make ", "clang++ ") {
			return Cpp
		}
		if containsAny(c, "ruby ", "gem ", "bundle ", "rake ", "rails ") {
			return Ruby
		}
		if containsAny(c, "php ", "composer ", "artisan ", "phpunit") {
			return PHP
		}
		if containsAny(c, "swift ", "swiftc ", "xcodebuild") {
			return Swift
		}

		// Tier 3
		if containsAny(c, "dart ", "flutter ", "pub get") {
			return Dart
		}
		if containsAny(c, "elixir ", "mix ", "iex ") {
			return Elixir
		}
		if containsAny(c, "ghc ", "cabal ", "stack ", "haskell") {
			return Haskell
		}
		if containsAny(c, "clojure ", "clj ", "lein ") {
			return Clojure
		}
		if containsAny(c, "perl ", "cpan ", "cpanm ") {
			return Perl
		}
		if containsAny(c, "lua ", "luarocks ") {
			return Lua
		}
		if containsAny(c, "zig ") {
			return Zig
		}
		if containsAny(c, "nim ", "nimble ") {
			return Nim
		}
		if containsAny(c, "ocaml ", "opam ", "dune ") {
			return OCaml
		}
		if containsAny(c, "julia ") {
			return Julia
		}
		if containsAny(c, "rscript ", "r -e ", "renv") {
			return R
		}
		if containsAny(c, "rebar3 ", "erlc ") {
			return Erlang
		}
		if containsAny(c, "gfortran ", "f90", "fortran") {
			return Fortran
		}

		// Tier 4 — Infrastructure
		if containsAny(c, "terraform ", "tofu ") {
			return Terraform
		}
		if containsAny(c, "ansible ", "ansible-playbook ") {
			return Ansible
		}
		if containsAny(c, "pwsh ", "powershell ") {
			return Powershell
		}
		if containsAny(c, "bash ", "sh ", "#!/bin/") {
			return Shell
		}
	}
	return Lightweight
}

// containsAny checks if the string contains any of the given substrings
func containsAny(s string, substrs ...string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ListSupportedLanguages returns all registered language names
func ListSupportedLanguages() []Language {
	langs := make([]Language, 0, len(LanguageRegistry))
	for lang := range LanguageRegistry {
		langs = append(langs, lang)
	}
	return langs
}
