package handlers

import (
	"encoding/json"
	"gorm.io/datatypes"
	"school-oj/apps/api/internal/models"
	"time"
)

func ticketEditableStatus(status models.ProblemChangeStatus) bool {
	return status == models.ProblemChangePending || status == models.ProblemChangeReturned
}

func ticketCanResolve(ticket models.ProblemChangeTicket) bool {
	return ticket.Action == models.ProblemChangeSuggest || ticket.Action == models.ProblemChangeReplace && (ticket.ProblemID == nil || ticket.ExamID != nil || ticket.ContestID != nil)
}

type ticketHistoryEntry struct {
	Action   string    `json:"action"`
	Note     string    `json:"note"`
	Actor    string    `json:"actor"`
	Revision int       `json:"revision"`
	At       time.Time `json:"at"`
}

// A revision-checked transaction writes history together with the state change.
func ticketHistory(ticket models.ProblemChangeTicket, user models.User, action, note string) datatypes.JSON {
	var entries []ticketHistoryEntry
	_ = json.Unmarshal(ticket.History, &entries)
	entries = append(entries, ticketHistoryEntry{Action: action, Note: note, Actor: user.Name, Revision: ticket.Revision + 1, At: time.Now()})
	body, _ := json.Marshal(entries)
	return body
}
