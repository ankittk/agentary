import { useEffect, useState } from "react";
import { fetchNetwork, networkAllow, networkDisallow, networkReset } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function NetworkConfig() {
  const [allowlist, setAllowlist] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [newDomain, setNewDomain] = useState("");
  const [actionLoading, setActionLoading] = useState(false);

  function load() {
    setLoading(true);
    fetchNetwork()
      .then((res) => setAllowlist(res.allowlist ?? []))
      .catch(() => setAllowlist([]))
      .finally(() => setLoading(false));
  }

  useEffect(() => {
    load();
  }, []);

  async function handleAllow(e: React.FormEvent) {
    e.preventDefault();
    const domain = newDomain.trim();
    if (!domain) return;
    setActionLoading(true);
    try {
      await networkAllow(domain);
      setNewDomain("");
      load();
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(false);
    }
  }

  async function handleDisallow(domain: string) {
    setActionLoading(true);
    try {
      await networkDisallow(domain);
      load();
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(false);
    }
  }

  async function handleReset() {
    setActionLoading(true);
    try {
      await networkReset();
      load();
    } catch (err) {
      console.error(err);
    } finally {
      setActionLoading(false);
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 max-w-xl" aria-busy="true" aria-label="Loading network config">
        <div className="h-8 w-48 rounded bg-[var(--muted)]/20 animate-pulse" />
        <div className="h-32 rounded bg-[var(--muted)]/20 animate-pulse" />
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-xl">
      <h2 className="font-semibold text-lg" id="network-heading">Network allowlist</h2>
      <p className="text-sm text-[var(--muted)]">
        Domains agents are allowed to access. Add domains to permit outbound requests.
      </p>

      <form onSubmit={handleAllow} className="flex gap-2 items-center flex-wrap">
        <input
          type="text"
          placeholder="example.com"
          value={newDomain}
          onChange={(e) => setNewDomain(e.target.value)}
          className="rounded-md border border-[var(--border)] bg-[var(--card)] px-3 py-2 text-sm w-48"
          aria-label="Domain to allow"
        />
        <Button type="submit" disabled={actionLoading}>Allow</Button>
      </form>

      {allowlist.length === 0 ? (
        <p className="text-sm text-[var(--muted)]" aria-live="polite">No domains in allowlist.</p>
      ) : (
        <ul className="space-y-2" aria-labelledby="network-heading">
          {allowlist.map((d) => (
            <li key={d} className="flex items-center justify-between gap-2 rounded border border-[var(--border)] px-3 py-2">
              <span className="text-sm font-mono">{d}</span>
              <Button
                variant="ghost"
                size="sm"
                disabled={actionLoading}
                onClick={() => handleDisallow(d)}
                aria-label={`Remove ${d} from allowlist`}
              >
                Remove
              </Button>
            </li>
          ))}
        </ul>
      )}

      <Button variant="outline" size="sm" disabled={actionLoading} onClick={handleReset}>
        Reset allowlist
      </Button>
    </div>
  );
}
