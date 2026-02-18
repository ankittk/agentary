import { useEffect, useState } from "react";
import { fetchDiff } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function DiffViewer({ team, taskId, onClose }: { team: string; taskId: number; onClose: () => void }) {
  const [diff, setDiff] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchDiff(team, taskId)
      .then((res) => {
        if (!cancelled) setDiff(res.diff || "(no diff)");
      })
      .catch(() => {
        if (!cancelled) setDiff("(failed to load diff)");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [team, taskId]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-[var(--card)] border border-[var(--border)] rounded-lg shadow-lg max-w-4xl w-full max-h-[80vh] overflow-hidden flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="p-3 border-b border-[var(--border)] flex justify-between items-center">
          <span className="font-medium">Diff — Task #{taskId}</span>
          <Button variant="ghost" size="sm" onClick={onClose}>Close</Button>
        </div>
        <pre className="p-4 overflow-auto text-xs font-mono whitespace-pre-wrap flex-1 bg-[#0d1117] text-[#c9d1d9]">
          {loading ? "Loading…" : diff}
        </pre>
      </div>
    </div>
  );
}
