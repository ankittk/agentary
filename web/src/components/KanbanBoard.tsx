import { useState } from "react";
import type { Task } from "@/lib/api";
import { createTask, patchTask } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { DiffViewer } from "@/components/DiffViewer";

const COLUMNS = ["todo", "in_progress", "in_review", "in_approval", "merging", "done", "failed"] as const;

const TRANSITIONS: Record<string, string[]> = {
  todo: ["in_progress"],
  in_progress: ["in_review", "failed"],
  in_review: ["in_approval", "in_progress"],
  in_approval: ["merging", "in_progress"],
  merging: ["done", "failed"],
  done: [],
  failed: ["todo"],
};

export function KanbanBoard({
  team,
  tasks,
  onUpdate,
}: {
  team: string;
  tasks: Task[];
  onUpdate: () => void;
}) {
  const [newTitle, setNewTitle] = useState("");
  const [loading, setLoading] = useState(false);
  const [diffTaskId, setDiffTaskId] = useState<number | null>(null);

  const byStatus = (status: string) => tasks.filter((t) => t.status === status);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    if (!newTitle.trim()) return;
    setLoading(true);
    try {
      await createTask(team, newTitle.trim());
      setNewTitle("");
      onUpdate();
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  async function handleStatusChange(taskId: number, status: string) {
    try {
      await patchTask(team, taskId, { status });
      onUpdate();
    } catch (err) {
      console.error(err);
    }
  }

  return (
    <div className="space-y-4">
      <form onSubmit={handleCreate} className="flex gap-2 items-center flex-wrap">
        <input
          type="text"
          placeholder="New task title"
          value={newTitle}
          onChange={(e) => setNewTitle(e.target.value)}
          className="rounded-md border border-[var(--border)] bg-[var(--card)] px-3 py-2 text-sm w-64"
        />
        <Button type="submit" disabled={loading}>Create</Button>
      </form>

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-7 gap-4">
        {COLUMNS.map((status) => (
          <div key={status} className="rounded-lg border border-[var(--border)] bg-[var(--card)]/50 p-3">
            <h3 className="font-medium text-sm uppercase tracking-wider text-[var(--muted)] mb-3">
              {status.replace("_", " ")}
            </h3>
            <div className="space-y-2">
              {byStatus(status).map((task) => (
                <Card key={task.task_id} className="cursor-pointer hover:border-[var(--muted)]/50">
                  <CardHeader className="p-3 pb-0">
                    <div className="flex justify-between items-start gap-2">
                      <span className="text-sm font-medium line-clamp-2">{task.title}</span>
                      <span className="text-xs text-[var(--muted)]">#{task.task_id}</span>
                    </div>
                    {task.assignee && (
                      <p className="text-xs text-[var(--muted)] mt-1">@{task.assignee}</p>
                    )}
                  </CardHeader>
                  <CardContent className="p-3 pt-1">
                    <div className="flex flex-wrap gap-1">
                      {(TRANSITIONS[status] ?? []).map((s) => (
                        <Button
                          key={s}
                          variant="ghost"
                          size="sm"
                          className="text-xs h-7"
                          onClick={() => handleStatusChange(task.task_id, s)}
                        >
                          → {s.replace("_", " ")}
                        </Button>
                      ))}
                      <Button
                        variant="outline"
                        size="sm"
                        className="text-xs h-7"
                        onClick={() => setDiffTaskId(task.task_id)}
                      >
                        Diff
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          </div>
        ))}
      </div>

      {diffTaskId !== null && (
        <DiffViewer team={team} taskId={diffTaskId} onClose={() => setDiffTaskId(null)} />
      )}
    </div>
  );
}
