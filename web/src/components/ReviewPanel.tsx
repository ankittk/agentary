import { useEffect, useState } from "react";
import type { Task, TaskReview } from "@/lib/api";
import { fetchTasks, fetchTaskReviews, approveTask } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { DiffViewer } from "@/components/DiffViewer";

export function ReviewPanel({ team, onUpdate }: Readonly<{ team: string; onUpdate?: () => void }>) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [diffTaskId, setDiffTaskId] = useState<number | null>(null);
  const [reviewsByTask, setReviewsByTask] = useState<Record<number, TaskReview[]>>({});
  const [actionLoading, setActionLoading] = useState<number | null>(null);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchTasks(team)
      .then((list) => {
        if (!cancelled) setTasks(list.filter((t) => t.status === "in_approval" || t.current_stage === "InApproval"));
      })
      .catch(() => {
        if (!cancelled) setTasks([]);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [team]);

  useEffect(() => {
    if (tasks.length === 0) return;
    const ids = tasks.map((t) => t.task_id);
    Promise.all(ids.map((id) => fetchTaskReviews(team, id)))
      .then((results) => {
        const map: Record<number, TaskReview[]> = {};
        ids.forEach((id, i) => { map[id] = results[i] ?? []; });
        setReviewsByTask(map);
      })
      .catch(() => {});
  }, [team, tasks]);

  async function handleApprove(taskId: number, outcome: string) {
    setActionLoading(taskId);
    try {
      await approveTask(team, taskId, outcome);
      onUpdate?.();
      setTasks((prev) => prev.filter((t) => t.task_id !== taskId));
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(null);
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 max-w-2xl" aria-busy="true" aria-label="Loading reviews">
        <div className="h-8 w-48 rounded bg-[var(--muted)]/20 animate-pulse" />
        <div className="h-24 rounded bg-[var(--muted)]/20 animate-pulse" />
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <h2 className="font-semibold text-lg" id="review-panel-heading">Tasks in approval</h2>
      {tasks.length === 0 ? (
        <p className="text-[var(--muted)] text-sm" aria-live="polite">No tasks waiting for approval.</p>
      ) : (
        <ul className="space-y-3" aria-labelledby="review-panel-heading">
          {tasks.map((task) => (
            <li key={task.task_id}>
              <Card className="border border-[var(--border)]">
                <CardHeader className="p-3 pb-0">
                  <div className="flex justify-between items-start gap-2">
                    <span className="font-medium text-sm">{task.title}</span>
                    <span className="text-xs text-[var(--muted)]">#{task.task_id}</span>
                  </div>
                  {task.assignee && (
                    <p className="text-xs text-[var(--muted)] mt-1">Assignee: @{task.assignee}</p>
                  )}
                </CardHeader>
                <CardContent className="p-3 pt-1 space-y-2">
                  {(reviewsByTask[task.task_id] ?? []).length > 0 && (
                    <div className="text-xs text-[var(--muted)]">
                      {reviewsByTask[task.task_id].length} review(s)
                    </div>
                  )}
                  <div className="flex flex-wrap gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setDiffTaskId(task.task_id)}
                      aria-label={`View diff for task ${task.task_id}`}
                    >
                      View diff
                    </Button>
                    <Button
                      size="sm"
                      disabled={actionLoading === task.task_id}
                      onClick={() => handleApprove(task.task_id, "approved")}
                      aria-label={`Approve task ${task.task_id}`}
                    >
                      Approve
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={actionLoading === task.task_id}
                      onClick={() => handleApprove(task.task_id, "changes_requested")}
                      aria-label={`Request changes for task ${task.task_id}`}
                    >
                      Request changes
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </li>
          ))}
        </ul>
      )}

      {diffTaskId !== null && (
        <DiffViewer team={team} taskId={diffTaskId} onClose={() => setDiffTaskId(null)} />
      )}
    </div>
  );
}
