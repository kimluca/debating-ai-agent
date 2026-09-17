export interface Turn {
  persona: string;
  content: string;
}

export interface Debate {
  question: string;
  personas: string[];
  turns: Turn[];
  synthesis: string;
}

export interface DebateSummary {
  id: number;
  question: string;
}

interface GraphQLResponse<T> {
  data?: T;
  errors?: { message: string }[];
}

async function gql<T>(query: string, variables: Record<string, unknown>): Promise<T> {
  const res = await fetch("/graphql", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });
  const json: GraphQLResponse<T> = await res.json();
  if (json.errors?.length) throw new Error(json.errors.map((e) => e.message).join("; "));
  if (!json.data) throw new Error("empty response");
  return json.data;
}

export async function startDebate(question: string): Promise<{ id: number; debate: Debate }> {
  const data = await gql<{ startDebate: { id: number; debate: Debate } }>(
    `mutation { startDebate(question: $question) { id debate { question } } }`,
    { question }
  );
  return data.startDebate;
}

export async function listDebates(limit = 20): Promise<DebateSummary[]> {
  const data = await gql<{ debates: DebateSummary[] }>(`query { debates(limit: $limit) }`, { limit });
  return data.debates;
}
