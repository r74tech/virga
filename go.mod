module github.com/r74tech/virga

go 1.24.1

toolchain go1.24.3

require (
	github.com/Qitmeer/llama.go v0.0.0-20250101000000-000000000000
	github.com/ThinkInAIXYZ/go-mcp v0.2.24
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/go-memdb v1.3.5
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/peterh/liner v1.2.2
	github.com/stretchr/testify v1.11.1
	golang.org/x/sys v0.39.0
	golang.org/x/term v0.38.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.5 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/ethereum/go-ethereum v1.15.8 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/orcaman/concurrent-map/v2 v2.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	github.com/urfave/cli/v2 v2.27.5 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

// Use local llama.go from libs directory (platform-specific)
// This will be updated dynamically at build time based on target platform
replace github.com/Qitmeer/llama.go => ./internal/implant/llama/libs/placeholder
