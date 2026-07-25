package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"adversarychef/kali/internal/job"
)

// Register registers all MCP tools.
func Register(server *mcp.Server, mgr *job.Manager) {
	registerExec(server, mgr)
	registerListJobs(server, mgr)
	registerGetJob(server, mgr)
	registerJobWait(server, mgr)
	registerKillJob(server, mgr)
}
