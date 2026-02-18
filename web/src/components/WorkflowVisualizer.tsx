import { DEFAULT_WORKFLOW_STAGES, DEFAULT_WORKFLOW_TRANSITIONS } from "@/lib/api";
import { cn } from "@/lib/utils";

export function WorkflowVisualizer() {
  return (
    <div className="space-y-4">
      <h2 className="font-semibold text-lg">Default workflow</h2>
      <div className="flex flex-wrap items-center gap-2">
        {DEFAULT_WORKFLOW_STAGES.map((s, i) => (
          <div key={s.name} className="flex items-center gap-2">
            <div
              className={cn(
                "rounded-lg border px-3 py-2 text-sm font-medium",
                s.type === "terminal" ? "border-emerald-500/50 bg-emerald-500/10" : "border-[var(--border)] bg-[var(--card)]"
              )}
            >
              {s.name}
              <span className="ml-2 text-xs text-[var(--muted)]">{s.type}</span>
            </div>
            {i < DEFAULT_WORKFLOW_STAGES.length - 1 && (
              <span className="text-[var(--muted)]">→</span>
            )}
          </div>
        ))}
      </div>
      <div className="mt-6">
        <h3 className="text-sm font-medium text-[var(--muted)] mb-2">Transitions</h3>
        <ul className="text-sm space-y-1">
          {DEFAULT_WORKFLOW_TRANSITIONS.map((t) => (
            <li key={`${t.from}-${t.outcome}-${t.to}`} className="font-mono">
              {t.from} --[{t.outcome}]→ {t.to}
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
