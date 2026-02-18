import type { Task } from "@/lib/api";

export function TaskChart({ tasks }: { tasks: Task[] }) {
  const counts = { todo: 0, in_progress: 0, done: 0, failed: 0 };
  for (const t of tasks) {
    if (t.status in counts) (counts as Record<string, number>)[t.status]++;
  }
  const max = Math.max(1, ...Object.values(counts));

  return (
    <div className="space-y-4">
      <h2 className="font-semibold text-lg">Task counts</h2>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {(["todo", "in_progress", "done", "failed"] as const).map((status) => (
          <div key={status} className="rounded-lg border border-[var(--border)] bg-[var(--card)] p-4">
            <div className="text-sm font-medium text-[var(--muted)] mb-2">{status.replace("_", " ")}</div>
            <div className="flex items-end gap-1 h-24">
              <div
                className="flex-1 rounded-t bg-[var(--foreground)]/20 min-w-[8px] transition-all"
                style={{ height: `${(counts[status] / max) * 100}%` }}
              />
            </div>
            <div className="text-2xl font-semibold mt-2">{counts[status]}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
