// Package orchestrator drives a multi-round debate between several LLM
// personas and produces a final synthesis.
package orchestrator

import (
	"fmt"
	"strings"

	"debate/internal/llm"
)

// Persona is one voice in the debate, defined purely by its system prompt.
type Persona struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

// DefaultPersonas covers three archetypal reasoning styles - enough to
// surface disagreement without the transcript getting unwieldy.
func DefaultPersonas() []Persona {
	return []Persona{
		{
			Name:   "Advocate",
			Prompt: "You are a sharp advocate. Argue the strongest possible case FOR the motion. Be concise (3-5 sentences), concrete, and avoid hedging.",
		},
		{
			Name:   "Skeptic",
			Prompt: "You are a rigorous skeptic. Find the weakest assumption in the argument so far and press on it. Be concise (3-5 sentences). Steelman before you critique.",
		},
		{
			Name:   "Pragmatist",
			Prompt: "You are a pragmatist focused on real-world trade-offs, costs, and second-order effects. Be concise (3-5 sentences) and concrete.",
		},
	}
}

// Turn is one utterance in the debate, attributed to a persona.
type Turn struct {
	Persona string `json:"persona"`
	Content string `json:"content"`
}

// Debate is the full record of a run: the question, every turn, and a
// final synthesis produced by a neutral "moderator" pass.
type Debate struct {
	Question  string  `json:"question"`
	Personas  []string `json:"personas"`
	Turns     []Turn  `json:"turns"`
	Synthesis string  `json:"synthesis"`
}

type Orchestrator struct {
	Provider llm.Provider
	Rounds   int // how many times each persona speaks
}

func New(provider llm.Provider, rounds int) *Orchestrator {
	if rounds < 1 {
		rounds = 2
	}
	return &Orchestrator{Provider: provider, Rounds: rounds}
}

// Run executes the full debate: each persona speaks in turn for
// o.Rounds rounds, each seeing the full transcript so far, then a
// moderator pass synthesizes a final answer.
func (o *Orchestrator) Run(question string, personas []Persona) (*Debate, error) {
	if len(personas) == 0 {
		personas = DefaultPersonas()
	}

	debate := &Debate{Question: question}
	for _, p := range personas {
		debate.Personas = append(debate.Personas, p.Name)
	}

	// The shared transcript, expressed as alternating user/assistant
	// messages so any persona can be asked to "continue" it. We prefix
	// each turn with the speaker's name so personas can react to each
	// other by name even though the LLM API only sees user/assistant.
	var transcript []llm.Message
	transcript = append(transcript, llm.Message{Role: "user", Content: "Debate motion: " + question})

	for round := 0; round < o.Rounds; round++ {
		for _, p := range personas {
			systemPrompt := p.Prompt
			reply, err := o.Provider.Complete(systemPrompt, transcript)
			if err != nil {
				return nil, fmt.Errorf("persona %s failed: %w", p.Name, err)
			}
			turn := Turn{Persona: p.Name, Content: reply}
			debate.Turns = append(debate.Turns, turn)
			transcript = append(transcript,
				llm.Message{Role: "assistant", Content: fmt.Sprintf("[%s] %s", p.Name, reply)},
				llm.Message{Role: "user", Content: "Continue the debate, responding to the point above."},
			)
		}
	}

	synthesis, err := o.synthesize(debate)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}
	debate.Synthesis = synthesis
	return debate, nil
}

func (o *Orchestrator) synthesize(d *Debate) (string, error) {
	var sb strings.Builder
	for _, t := range d.Turns {
		sb.WriteString(fmt.Sprintf("%s: %s\n", t.Persona, t.Content))
	}
	moderatorPrompt := "You are a neutral moderator. Given a debate transcript, write a short " +
		"(4-6 sentence) synthesis: where the personas agreed, where they genuinely " +
		"disagreed, and what a reasonable person should conclude. Do not introduce new arguments."
	msgs := []llm.Message{{Role: "user", Content: sb.String()}}
	return o.Provider.Complete(moderatorPrompt, msgs)
}
