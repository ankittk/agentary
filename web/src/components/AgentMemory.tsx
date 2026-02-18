import { useEffect, useState } from "react";
import type { Agent } from "@/lib/api";
import { fetchJournal, fetchAgentConfig } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function AgentMemory({ team, agents }: Readonly<{ team: string; agents: Agent[] }>) {
  const [selectedAgent, setSelectedAgent] = useState<string | null>(null);
  const [journal, setJournal] = useState<string>("");
  const [config, setConfig] = useState<{ model: string; max_tokens: number } | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!selectedAgent) {
      setJournal("");
      setConfig(null);
      return;
    }
    setLoading(true);
    Promise.all([
      fetchJournal(team, selectedAgent),
      fetchAgentConfig(team, selectedAgent),
    ])
      .then(([j, c]) => {
        setJournal(j.content ?? "");
        setConfig(c);
      })
      .catch(() => {
        setJournal("(failed to load)");
        setConfig(null);
      })
      .finally(() => setLoading(false));
  }, [team, selectedAgent]);

  if (agents.length === 0) {
    return (
      <div className="space-y-4 max-w-2xl">
        <h2 className="font-semibold text-lg">Agent memory</h2>
        <p className="text-sm text-[var(--muted)]" aria-live="polite">No agents in this team. Add agents to browse journals.</p>
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <h2 className="font-semibold text-lg" id="memory-heading">Agent memory</h2>
      <p className="text-sm text-[var(--muted)]">
        View agent journals and config. Journals are appended after each agent turn.
      </p>

      <div className="flex gap-2 flex-wrap">
        {agents.map((a) => (
          <Button
            key={a.Name}
            variant={selectedAgent === a.Name ? "default" : "outline"}
            size="sm"
            onClick={() => setSelectedAgent(a.Name)}
            aria-pressed={selectedAgent === a.Name}
            aria-label={`View memory for ${a.Name}`}
          >
            {a.Name}
          </Button>
        ))}
      </div>

      {selectedAgent && (
        <div className="rounded-lg border border-[var(--border)] overflow-hidden">
          <div className="border-b border-[var(--border)] px-3 py-2 text-sm font-medium bg-[var(--card)]/50">
            {selectedAgent}
            {config && (config.model !== "" || config.max_tokens > 0) ? (
              <span className="text-[var(--muted)] font-normal ml-2">
                — {config.model || "default"} {config.max_tokens > 0 ? `(${config.max_tokens} tokens)` : ""}
              </span>
            ) : null}
          </div>
          <div className="p-3">
            {loading ? (
              <div className="h-32 rounded bg-[var(--muted)]/20 animate-pulse" aria-busy="true" />
            ) : (
              <pre className="text-xs font-mono whitespace-pre-wrap break-words bg-[var(--muted)]/10 rounded p-3 max-h-96 overflow-auto">
                {journal || "(no journal entries yet)"}
              </pre>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
