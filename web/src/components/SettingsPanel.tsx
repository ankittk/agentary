import { useEffect, useState } from "react";
import { fetchConfig, fetchBootstrap } from "@/lib/api";
import type { Config } from "@/lib/api";

export function SettingsPanel() {
  const [config, setConfig] = useState<Config | null>(null);
  const [bootstrapId, setBootstrapId] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    Promise.all([fetchConfig(), fetchBootstrap()])
      .then(([cfg, boot]) => {
        if (!cancelled) {
          setConfig(cfg);
          setBootstrapId(boot.config?.bootstrap_id ?? null);
        }
      })
      .catch(() => {
        if (!cancelled) setConfig(null);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, []);

  if (loading) {
    return (
      <div className="space-y-4 max-w-xl" aria-busy="true" aria-label="Loading settings">
        <div className="h-8 w-48 rounded bg-[var(--muted)]/20 animate-pulse" />
        <div className="h-24 rounded bg-[var(--muted)]/20 animate-pulse" />
      </div>
    );
  }

  return (
    <div className="space-y-4 max-w-xl">
      <h2 className="font-semibold text-lg" id="settings-heading">Settings</h2>

      <section className="rounded-lg border border-[var(--border)] p-4 space-y-2" aria-labelledby="settings-heading">
        <h3 className="text-sm font-medium text-[var(--muted)]">Config</h3>
        <dl className="text-sm space-y-1">
          <div>
            <dt className="text-[var(--muted)]">Human name</dt>
            <dd className="font-mono">{config?.human_name ?? "—"}</dd>
          </div>
          <div>
            <dt className="text-[var(--muted)]">Home</dt>
            <dd className="font-mono break-all">{config?.hc_home ?? "—"}</dd>
          </div>
          <div>
            <dt className="text-[var(--muted)]">Bootstrap ID</dt>
            <dd className="font-mono text-xs">{bootstrapId ?? "—"}</dd>
          </div>
        </dl>
      </section>

      <section className="rounded-lg border border-[var(--border)] p-4">
        <h3 className="text-sm font-medium text-[var(--muted)] mb-2">API</h3>
        <p className="text-sm text-[var(--muted)]">
          Use <code className="rounded bg-[var(--muted)]/20 px-1">X-API-Key</code> or <code className="rounded bg-[var(--muted)]/20 px-1">api_key</code> query param when the server requires authentication.
        </p>
      </section>

      <p className="text-xs text-[var(--muted)]" aria-live="polite">
        Agentary — single-binary daemon with embedded React UI.
      </p>
    </div>
  );
}
