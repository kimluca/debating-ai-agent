// Package graphql exposes the debate orchestrator over a small,
// dependency-free GraphQL endpoint (same approach as the Codebase
// Archaeologist project: a hand-rolled resolver with an identical wire
// contract to real GraphQL, kept minimal on purpose).
package graphql

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"debate/internal/db"
	"debate/internal/orchestrator"
)

type Server struct {
	Store        *db.Store
	Orchestrator *orchestrator.Orchestrator
}

type gqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type gqlResponse struct {
	Data   interface{} `json:"data,omitempty"`
	Errors []gqlError  `json:"errors,omitempty"`
}

type gqlError struct{ Message string `json:"message"` }

var operationRe = regexp.MustCompile(`(query|mutation)\s*{\s*(\w+)\s*\(([^)]*)\)`)
var noArgRe = regexp.MustCompile(`(query|mutation)\s*{\s*(\w+)`)

func (s *Server) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		var req gqlRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, "invalid JSON: "+err.Error())
			return
		}
		field, args, ok := parseOperation(req.Query)
		if !ok {
			writeErr(w, "could not parse operation")
			return
		}
		resolved := resolveArgs(args, req.Variables)

		switch field {
		case "startDebate":
			s.resolveStartDebate(w, resolved)
		case "debate":
			s.resolveGetDebate(w, resolved)
		case "debates":
			s.resolveListDebates(w, resolved)
		default:
			writeErr(w, fmt.Sprintf("unknown field %q", field))
		}
	}
}

func parseOperation(query string) (string, map[string]string, bool) {
	query = strings.TrimSpace(query)
	if m := operationRe.FindStringSubmatch(query); m != nil {
		args := map[string]string{}
		for _, pair := range strings.Split(m[3], ",") {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) == 2 {
				args[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
		return m[2], args, true
	}
	if m := noArgRe.FindStringSubmatch(query); m != nil {
		return m[2], map[string]string{}, true
	}
	return "", nil, false
}

func resolveArgs(args map[string]string, vars map[string]interface{}) map[string]string {
	out := map[string]string{}
	for k, v := range args {
		if strings.HasPrefix(v, "$") {
			if val, ok := vars[strings.TrimPrefix(v, "$")]; ok {
				out[k] = fmt.Sprintf("%v", val)
			}
			continue
		}
		out[k] = strings.Trim(v, `"`)
	}
	return out
}

func (s *Server) resolveStartDebate(w http.ResponseWriter, args map[string]string) {
	question := args["question"]
	if question == "" {
		writeErr(w, "missing required argument: question")
		return
	}
	d, err := s.Orchestrator.Run(question, nil)
	if err != nil {
		writeErr(w, "debate failed: "+err.Error())
		return
	}
	id, err := s.Store.SaveDebate(d)
	if err != nil {
		fmt.Println("warning: failed to persist debate:", err)
	}
	writeData(w, "startDebate", map[string]interface{}{"id": id, "debate": d})
}

func (s *Server) resolveGetDebate(w http.ResponseWriter, args map[string]string) {
	id, err := strconv.ParseInt(args["id"], 10, 64)
	if err != nil {
		writeErr(w, "invalid id")
		return
	}
	d, err := s.Store.GetDebate(id)
	if err != nil {
		writeErr(w, "not found: "+err.Error())
		return
	}
	writeData(w, "debate", d)
}

func (s *Server) resolveListDebates(w http.ResponseWriter, args map[string]string) {
	limit := 20
	if v, err := strconv.Atoi(args["limit"]); err == nil && v > 0 {
		limit = v
	}
	list, err := s.Store.ListDebates(limit)
	if err != nil {
		writeErr(w, err.Error())
		return
	}
	writeData(w, "debates", list)
}

func writeData(w http.ResponseWriter, field string, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gqlResponse{Data: map[string]interface{}{field: payload}})
}

func writeErr(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gqlResponse{Errors: []gqlError{{Message: msg}}})
}
