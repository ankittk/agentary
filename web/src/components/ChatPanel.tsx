import { useEffect, useState } from "react";
import { fetchMessages, sendMessage } from "@/lib/api";
import { Button } from "@/components/ui/button";

const DEFAULT_RECIPIENT = "manager";

export function ChatPanel({ team }: { team: string }) {
  const [messages, setMessages] = useState<{ sender: string; content: string; created_at: string }[]>([]);
  const [recipient, setRecipient] = useState(DEFAULT_RECIPIENT);
  const [input, setInput] = useState("");
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    fetchMessages(team, recipient)
      .then((res) => {
        if (!cancelled) setMessages(res);
      })
      .catch(() => {
        if (!cancelled) setMessages([]);
      });
    return () => { cancelled = true; };
  }, [team, recipient]);

  async function handleSend(e: React.FormEvent) {
    e.preventDefault();
    if (!input.trim()) return;
    setLoading(true);
    try {
      await sendMessage(team, recipient, input.trim());
      setInput("");
      const res = await fetchMessages(team, recipient);
      setMessages(res);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-4 max-w-2xl">
      <h2 className="font-semibold text-lg">Messages</h2>
      <div className="flex gap-2 items-center">
        <label className="text-sm text-[var(--muted)]">To:</label>
        <input
          type="text"
          value={recipient}
          onChange={(e) => setRecipient(e.target.value)}
          className="rounded-md border border-[var(--border)] bg-[var(--card)] px-3 py-2 text-sm w-32"
        />
      </div>
      <div className="rounded-lg border border-[var(--border)] bg-[var(--card)] min-h-[200px] max-h-[400px] overflow-auto p-3 space-y-2">
        {messages.length === 0 && <p className="text-[var(--muted)] text-sm">No messages</p>}
        {messages.map((m, i) => (
          <div key={i} className="text-sm">
            <span className="font-medium text-[var(--muted)]">{m.sender}:</span> {m.content}
          </div>
        ))}
      </div>
      <form onSubmit={handleSend} className="flex gap-2">
        <input
          type="text"
          placeholder="Type a message…"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          className="flex-1 rounded-md border border-[var(--border)] bg-[var(--card)] px-3 py-2 text-sm"
        />
        <Button type="submit" disabled={loading}>Send</Button>
      </form>
    </div>
  );
}
