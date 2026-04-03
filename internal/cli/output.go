package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/a2aproject/a2a-go/v2/a2a"
)

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func printCard(w io.Writer, card *a2a.AgentCard) {
	fmt.Fprintf(w, "Name:         %s\n", card.Name)
	if card.Description != "" {
		fmt.Fprintf(w, "Description:  %s\n", card.Description)
	}
	fmt.Fprintf(w, "Version:      %s\n", card.Version)

	if len(card.SupportedInterfaces) > 0 {
		fmt.Fprintln(w, "Interfaces:")
		for _, iface := range card.SupportedInterfaces {
			fmt.Fprintf(w, "  %-12s %s\n", iface.ProtocolBinding, iface.URL)
		}
	}

	caps := card.Capabilities
	fmt.Fprintf(w, "Streaming:    %v\n", caps.Streaming)

	if len(card.Skills) > 0 {
		fmt.Fprintln(w, "Skills:")
		for _, s := range card.Skills {
			fmt.Fprintf(w, "  %-20s %s\n", s.ID, s.Name)
		}
	}
}

func printTask(w io.Writer, task *a2a.Task) {
	fmt.Fprintf(w, "Task:     %s\n", task.ID)
	if task.ContextID != "" {
		fmt.Fprintf(w, "Context:  %s\n", task.ContextID)
	}
	fmt.Fprintf(w, "Status:   %s", shortState(task.Status.State))
	if task.Status.Timestamp != nil {
		fmt.Fprintf(w, " (%s)", task.Status.Timestamp.Format("2006-01-02T15:04:05Z07:00"))
	}
	fmt.Fprintln(w)
	if task.Status.Message != nil {
		fmt.Fprintf(w, "  %s\n", messageText(task.Status.Message))
	}

	if len(task.Artifacts) > 0 {
		fmt.Fprintln(w, "Artifacts:")
		for _, art := range task.Artifacts {
			label := string(art.ID)
			if art.Name != "" {
				label = art.Name
			}
			fmt.Fprintf(w, "  [%s] %s\n", label, partsText(art.Parts))
		}
	}

	if len(task.History) > 0 {
		fmt.Fprintln(w, "History:")
		for _, msg := range task.History {
			role := "user"
			if msg.Role == a2a.MessageRoleAgent {
				role = "agent"
			}
			fmt.Fprintf(w, "  [%s] %s\n", role, messageText(msg))
		}
	}
}

func printMessage(w io.Writer, msg *a2a.Message) {
	role := "user"
	if msg.Role == a2a.MessageRoleAgent {
		role = "agent"
	}
	fmt.Fprintf(w, "[%s] %s\n", role, messageText(msg))
}

func printSendResult(w io.Writer, result a2a.SendMessageResult) {
	switch r := result.(type) {
	case *a2a.Task:
		printTask(w, r)
	case *a2a.Message:
		printMessage(w, r)
	}
}

func printEvent(w io.Writer, event a2a.Event) {
	switch e := event.(type) {
	case *a2a.TaskStatusUpdateEvent:
		state := shortState(e.Status.State)
		if e.Status.Message != nil {
			text := messageText(e.Status.Message)
			fmt.Fprintf(w, "[status] %s: %s\n", state, text)
		} else {
			fmt.Fprintf(w, "[status] %s\n", state)
		}
	case *a2a.TaskArtifactUpdateEvent:
		text := partsText(e.Artifact.Parts)
		if e.Append {
			fmt.Fprintf(w, "[artifact+] %s\n", text)
		} else {
			fmt.Fprintf(w, "[artifact] %s\n", text)
		}
	case *a2a.Task:
		printTask(w, e)
	case *a2a.Message:
		printMessage(w, e)
	}
}

func printTaskList(w io.Writer, resp *a2a.ListTasksResponse) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tSTATUS\tCONTEXT")
	for _, t := range resp.Tasks {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", t.ID, shortState(t.Status.State), t.ContextID)
	}
	_ = tw.Flush()
	if resp.NextPageToken != "" {
		fmt.Fprintf(w, "\nNext page token: %s\n", resp.NextPageToken)
	}
}

func messageText(msg *a2a.Message) string {
	return partsText(msg.Parts)
}

func partsText(parts a2a.ContentParts) string {
	var sb strings.Builder
	for i, p := range parts {
		if i > 0 {
			sb.WriteString(" ")
		}
		if t := p.Text(); t != "" {
			sb.WriteString(t)
			continue
		}
		if u := p.URL(); u != "" {
			sb.WriteString("[file: ")
			sb.WriteString(string(u))
			sb.WriteString("]")
			continue
		}
		if p.Raw() != nil {
			fmt.Fprintf(&sb, "[binary %d bytes]", len(p.Raw()))
			continue
		}
		if p.Data() != nil {
			b, _ := json.Marshal(p.Data())
			sb.WriteString(string(b))
			continue
		}
	}
	return sb.String()
}

func shortState(state a2a.TaskState) string {
	switch state {
	case a2a.TaskStateSubmitted:
		return "submitted"
	case a2a.TaskStateWorking:
		return "working"
	case a2a.TaskStateCompleted:
		return "completed"
	case a2a.TaskStateFailed:
		return "failed"
	case a2a.TaskStateCanceled:
		return "canceled"
	case a2a.TaskStateRejected:
		return "rejected"
	case a2a.TaskStateInputRequired:
		return "input-required"
	case a2a.TaskStateAuthRequired:
		return "auth-required"
	default:
		return string(state)
	}
}

func parseTaskState(s string) (a2a.TaskState, error) {
	switch strings.ToLower(s) {
	case "submitted":
		return a2a.TaskStateSubmitted, nil
	case "working":
		return a2a.TaskStateWorking, nil
	case "completed":
		return a2a.TaskStateCompleted, nil
	case "failed":
		return a2a.TaskStateFailed, nil
	case "canceled":
		return a2a.TaskStateCanceled, nil
	case "rejected":
		return a2a.TaskStateRejected, nil
	case "input-required":
		return a2a.TaskStateInputRequired, nil
	case "auth-required":
		return a2a.TaskStateAuthRequired, nil
	default:
		return "", fmt.Errorf("unknown task state %q", s)
	}
}
