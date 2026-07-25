// cmd/acactl/display/table.go
package display

import "fmt"

func PrintTaskTable(tasks []TaskSummary) {
	fmt.Println("┌──────────────────────┬──────────┬─────────┬──────────────────────┬──────────┐")
	fmt.Println("│ TASK ID              │ AGENT    │ STATUS  │ TITLE                │ DURATION │")
	fmt.Println("├──────────────────────┼──────────┼─────────┼──────────────────────┼──────────┤")
	for _, t := range tasks {
		fmt.Printf("│ %-20s │ %-8s │ %-7s │ %-20s │ %-8s │\n",
			truncate(t.ID, 20), truncate(t.Agent, 8),
			t.Status, truncate(t.Title, 20), t.Duration)
	}
	fmt.Println("└──────────────────────┴──────────┴─────────┴──────────────────────┴──────────┘")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

type TaskSummary struct {
	ID, Agent, Status, Title, Duration string
}
