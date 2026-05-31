package automation

import _ "embed"

const runnerScriptFileName = "runner.cjs"

//go:embed assets/runner.cjs
var runnerScriptContent []byte
