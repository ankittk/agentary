import { useEffect, useState } from "react";
import { fetchCharter, putCharter } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function CharterEditor({ team, onUpdate }: Readonly<{ team: string; onUpdate?: () => void }>) {
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    fetchCharter(team)
      .then((res) => {
        if (!cancelled) setContent(res.content ?? "");
      })
      .catch(() => {
        if (!cancelled) setContent("");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, [team]);

  async function handleSave(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    try {
      await putCharter(team, content);
      onUpdate?.();
    } catch (err) {
      console.error(err);
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 max-w-2xl" aria-busy="true" aria-label="Loading charter">
        <div className="h-8 w-48 rounded bg-[var(--muted)]/20 animate-pulse" />
        <div className="h-64 rounded bg-[var(--muted)]/20 animate-pulse" />
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <h2 className="font-semibold text-lg" id="charter-heading">Team charter</h2>
      <p className="text-sm text-[var(--muted)]">
        Markdown document describing team goals and conventions. Shown to agents as context.
      </p>

      <form onSubmit={handleSave} className="space-y-3">
        <textarea
          value={content}
          onChange={(e) => setContent(e.target.value)}
          className="w-full min-h-64 rounded-md border border-[var(--border)] bg-[var(--card)] px-3 py-2 text-sm font-mono resize-y"
          placeholder="# Team charter\n\nDescribe your team's mission and conventions..."
          aria-label="Charter content (markdown)"
          spellCheck="false"
        />
        <Button type="submit" disabled={saving}>
          {saving ? "Saving…" : "Save charter"}
        </Button>
      </form>
    </div>
  );
}
