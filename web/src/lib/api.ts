import { getApiBase } from "./utils";

const base = () => getApiBase();

export interface Config {
  human_name: string;
  hc_home: string;
  bootstrap_id: string;
}

export interface TeamSummary {
  TeamID: string;
  Name: string;
  CreatedAt: string;
  AgentCount: number;
  TaskCount: number;
}

export interface Bootstrap {
  config?: Config;
  teams?: TeamSummary[];
  initial_team?: string | null;
  tasks?: Task[];
  agents?: Agent[];
  repos?: unknown[];
  workflows?: unknown[];
  network?: { allowlist?: string[] };
}

export interface Agent {
  Name: string;
  Role: string;
  CreatedAt: string;
}

export interface Task {
  task_id: number;
  title: string;
  status: string;
  assignee?: string | null;
  dri?: string | null;
  workflow_id?: string | null;
  current_stage?: string | null;
  created_at: string;
  updated_at: string;
}

export async function fetchConfig(): Promise<Config> {
  const r = await fetch(`${base()}/config`);
  if (!r.ok) throw new Error("Failed to fetch config");
  return r.json();
}

function normalizeTask(t: Record<string, unknown>): Task {
  return {
    task_id: (t.TaskID ?? t.task_id) as number,
    title: (t.Title ?? t.title) as string,
    status: (t.Status ?? t.status) as string,
    assignee: (t.Assignee ?? t.assignee) as string | null | undefined,
    dri: (t.DRI ?? t.dri) as string | null | undefined,
    workflow_id: (t.WorkflowID ?? t.workflow_id) as string | null | undefined,
    current_stage: (t.CurrentStage ?? t.current_stage) as string | null | undefined,
    created_at: (t.CreatedAt ?? t.created_at) as string,
    updated_at: (t.UpdatedAt ?? t.updated_at) as string,
  };
}

export async function fetchBootstrap(): Promise<Bootstrap> {
  const r = await fetch(`${base()}/bootstrap`);
  if (!r.ok) throw new Error("Failed to fetch bootstrap");
  const data = await r.json();
  if (Array.isArray(data.tasks)) data.tasks = data.tasks.map(normalizeTask);
  return data;
}

export async function fetchTeams(): Promise<TeamSummary[]> {
  const r = await fetch(`${base()}/teams`);
  if (!r.ok) throw new Error("Failed to fetch teams");
  return r.json();
}

export async function fetchTasks(team: string): Promise<Task[]> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks?limit=200`);
  if (!r.ok) throw new Error("Failed to fetch tasks");
  const list = await r.json();
  return Array.isArray(list) ? list.map((t: Record<string, unknown>) => normalizeTask(t)) : [];
}

export async function fetchAgents(team: string): Promise<Agent[]> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/agents`);
  if (!r.ok) throw new Error("Failed to fetch agents");
  return r.json();
}

export async function createTask(team: string, title: string, status = "todo"): Promise<{ task_id: number }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ title, status }),
  });
  if (!r.ok) throw new Error("Failed to create task");
  return r.json();
}

export async function patchTask(team: string, taskId: number, patch: { status?: string; assignee?: string | null }): Promise<Task> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(patch),
  });
  if (!r.ok) throw new Error("Failed to update task");
  return normalizeTask(await r.json());
}

export async function fetchTask(team: string, taskId: number): Promise<Task> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}`);
  if (!r.ok) throw new Error("Failed to fetch task");
  return normalizeTask(await r.json());
}

export async function fetchDiff(team: string, taskId: number): Promise<{ diff: string }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}/diff`);
  if (!r.ok) throw new Error("Failed to fetch diff");
  return r.json();
}

// Default workflow stages for visualizer (matches server seedDefaultWorkflowStages)
export const DEFAULT_WORKFLOW_STAGES = [
  { name: "Coding", type: "agent", outcomes: "submit_for_review,done" },
  { name: "InReview", type: "agent", outcomes: "approved,changes_requested" },
  { name: "InApproval", type: "human", outcomes: "approved,changes_requested" },
  { name: "Merging", type: "merge", outcomes: "done" },
  { name: "Done", type: "terminal", outcomes: "" },
];
export const DEFAULT_WORKFLOW_TRANSITIONS = [
  { from: "Coding", outcome: "submit_for_review", to: "InReview" },
  { from: "Coding", outcome: "done", to: "Done" },
  { from: "InReview", outcome: "approved", to: "InApproval" },
  { from: "InReview", outcome: "changes_requested", to: "Coding" },
  { from: "InApproval", outcome: "approved", to: "Merging" },
  { from: "InApproval", outcome: "changes_requested", to: "Coding" },
  { from: "Merging", outcome: "done", to: "Done" },
];

export interface Message {
  message_id?: number;
  sender: string;
  recipient?: string;
  content: string;
  created_at: string;
}

export async function fetchMessages(team: string, recipient: string): Promise<Message[]> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/messages?recipient=${encodeURIComponent(recipient)}`);
  if (!r.ok) return [];
  const data = await r.json();
  return Array.isArray(data) ? data : (data.messages || []);
}

export async function sendMessage(team: string, recipient: string, content: string): Promise<void> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/messages`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ recipient, content }),
  });
  if (!r.ok) throw new Error("Failed to send message");
}

export function streamURL(): string {
  return `${base()}/stream`;
}

// Charter
export async function fetchCharter(team: string): Promise<{ content: string }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/charter`);
  if (!r.ok) throw new Error("Failed to fetch charter");
  return r.json();
}

export async function putCharter(team: string, content: string): Promise<{ ok: boolean }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/charter`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ content }),
  });
  if (!r.ok) throw new Error("Failed to save charter");
  return r.json();
}

// Agent journal
export async function fetchJournal(team: string, agent: string, limitBytes?: number): Promise<{ content: string }> {
  let url = `${base()}/teams/${encodeURIComponent(team)}/agents/${encodeURIComponent(agent)}/journal`;
  if (limitBytes != null && limitBytes > 0) url += `?limit=${limitBytes}`;
  const r = await fetch(url);
  if (!r.ok) throw new Error("Failed to fetch journal");
  return r.json();
}

// Agent config
export async function fetchAgentConfig(team: string, agent: string): Promise<{ model: string; max_tokens: number }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/agents/${encodeURIComponent(agent)}/config`);
  if (!r.ok) throw new Error("Failed to fetch agent config");
  return r.json();
}

// Network allowlist
export async function fetchNetwork(): Promise<{ allowlist: string[] }> {
  const r = await fetch(`${base()}/network`);
  if (!r.ok) throw new Error("Failed to fetch network");
  return r.json();
}

export async function networkAllow(domain: string): Promise<{ ok: boolean }> {
  const r = await fetch(`${base()}/network/allow`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ domain }),
  });
  if (!r.ok) throw new Error("Failed to allow domain");
  return r.json();
}

export async function networkDisallow(domain: string): Promise<{ ok: boolean }> {
  const r = await fetch(`${base()}/network/disallow`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ domain }),
  });
  if (!r.ok) throw new Error("Failed to disallow domain");
  return r.json();
}

export async function networkReset(): Promise<{ ok: boolean }> {
  const r = await fetch(`${base()}/network/reset`, { method: "POST" });
  if (!r.ok) throw new Error("Failed to reset network");
  return r.json();
}

// Task reviews
export interface TaskReview {
  review_id: number;
  task_id: number;
  reviewer_agent: string;
  outcome: string;
  comments: string;
  created_at: string;
}

export async function fetchTaskReviews(team: string, taskId: number): Promise<TaskReview[]> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}/reviews`);
  if (!r.ok) throw new Error("Failed to fetch reviews");
  const data = await r.json();
  return (data.reviews as TaskReview[]) ?? [];
}

export async function submitReview(
  team: string,
  taskId: number,
  body: { reviewer_agent: string; outcome: string; comments?: string }
): Promise<{ ok: boolean }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}/submit-review`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!r.ok) throw new Error("Failed to submit review");
  return r.json();
}

export async function approveTask(team: string, taskId: number, outcome: string): Promise<{ ok: boolean; current_stage?: string }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}/approve`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ outcome: outcome || "approved" }),
  });
  if (!r.ok) throw new Error("Failed to approve task");
  return r.json();
}

export async function requestReview(team: string, taskId: number): Promise<{ ok: boolean }> {
  const r = await fetch(`${base()}/teams/${encodeURIComponent(team)}/tasks/${taskId}/request-review`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
  if (!r.ok) throw new Error("Failed to request review");
  return r.json();
}
