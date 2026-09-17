import { useState } from "react";
import { startDebate, Debate } from "./graphqlClient";

const PERSONA_COLORS: Record<string, string> = {
  Advocate: "#2563eb",
  Skeptic: "#dc2626",
  Pragmatist: "#059669",
};

export default function App() {
  const [question, setQuestion] = useState("");
  const [debate, setDebate] = useState<Debate | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function run() {
    if (!question.trim()) return;
    setLoading(true);
    setError(null);
    setDebate(null);
    try {
      const { debate } = await startDebate(question.trim());
      setDebate(debate);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ maxWidth: 760, margin: "40px auto", fontFamily: "system-ui, sans-serif" }}>
      <h1>🗣️ Multi-Agent Debate</h1>
      <p style={{ color: "#555" }}>
        Three LLM personas — Advocate, Skeptic, Pragmatist — argue a question over
        several rounds, each seeing the full transcript so far, then a neutral
        moderator pass synthesizes where they agreed and disagreed.
      </p>

      <div style={{ display: "flex", gap: 8, margin: "16px 0" }}>
        <input
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="Should companies default to a 4-day work week?"
          style={{ flex: 1, padding: 8 }}
          onKeyDown={(e) => e.key === "Enter" && run()}
        />
        <button onClick={run} disabled={loading}>
          {loading ? "Debating..." : "Start debate"}
        </button>
      </div>

      {error && <p style={{ color: "crimson" }}>{error}</p>}

      {debate && (
        <div>
          <h3>{debate.question}</h3>
          <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
            {debate.turns.map((t, i) => (
              <div
                key={i}
                style={{
                  borderLeft: `4px solid ${PERSONA_COLORS[t.persona] ?? "#999"}`,
                  paddingLeft: 12,
                }}
              >
                <strong style={{ color: PERSONA_COLORS[t.persona] ?? "#999" }}>{t.persona}</strong>
                <p style={{ margin: "4px 0" }}>{t.content}</p>
              </div>
            ))}
          </div>

          <div style={{ marginTop: 24, padding: 16, background: "#f5f5f5", borderRadius: 8 }}>
            <strong>Moderator synthesis</strong>
            <p style={{ margin: "8px 0 0" }}>{debate.synthesis}</p>
          </div>
        </div>
      )}
    </div>
  );
}
