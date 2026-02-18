import type { Agent } from "@/lib/api";
import { Card, CardContent } from "@/components/ui/card";

export function AgentMonitor({ agents }: { agents: Agent[] }) {
  return (
    <div className="space-y-4">
      <h2 className="font-semibold text-lg">Agents</h2>
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {agents.map((a) => (
          <Card key={a.Name}>
            <CardContent className="p-4">
              <div className="font-medium">{a.Name}</div>
              <div className="text-sm text-[var(--muted)] mt-0.5">{a.Role}</div>
            </CardContent>
          </Card>
        ))}
      </div>
      {agents.length === 0 && (
        <p className="text-[var(--muted)]">No agents. Add agents via API or CLI.</p>
      )}
    </div>
  );
}
