// Kali MCP Server — async job execution for Kali toolchain.
package main

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/kali/internal/job"
	"adversarychef/kali/internal/tools"
	"adversarychef/mcputil"
)

func main() {
	cfg := mcputil.ParseConfig("kali", "0.3.0", 8080)
	mgr := job.NewManager(job.DefaultMaxOutput, job.DefaultTimeout)
	mcputil.Run(cfg, func(s *mcp.Server) { tools.Register(s, mgr) }, nil)
}
