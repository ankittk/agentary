import { type ReactNode } from "react";
import { ThemeToggle, useTheme } from "@/ThemeProvider";
import { cn } from "@/lib/utils";

export function Layout({
  children,
  team,
  teams,
  onTeamChange,
  activeView,
  onViewChange,
}: {
  children: ReactNode;
  team: string;
  teams: { Name: string }[];
  onTeamChange: (t: string) => void;
  activeView: string;
  onViewChange: (v: string) => void;
}) {
  const { resolved } = useTheme();
  const views = [
    { id: "kanban", label: "Kanban" },
    { id: "agents", label: "Agents" },
    { id: "workflow", label: "Workflow" },
    { id: "charts", label: "Charts" },
    { id: "chat", label: "Chat" },
    { id: "reviews", label: "Reviews" },
    { id: "network", label: "Network" },
    { id: "charter", label: "Charter" },
    { id: "memory", label: "Memory" },
    { id: "settings", label: "Settings" },
  ];

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b border-[var(--border)] px-4 py-3 flex items-center justify-between">
        <img
          src={resolved === "dark" ? "/logo-dark.svg" : "/logo-light.svg"}
          alt="Agentary"
          className="h-8 w-auto"
        />
        <div className="flex items-center gap-2">
          <ThemeToggle />
        </div>
      </header>
      <main className="flex flex-1">
        <aside className="w-56 border-r border-[var(--border)] p-3 flex flex-col gap-4">
          <div>
            <h2 className="text-xs font-medium uppercase tracking-wider text-[var(--muted)] mb-2">Team</h2>
            <select
              value={team}
              onChange={(e) => onTeamChange(e.target.value)}
              className="w-full rounded-md border border-[var(--border)] bg-[var(--card)] px-3 py-2 text-sm"
              aria-label="Team"
            >
              {teams.map((t) => (
                <option key={t.Name} value={t.Name}>{t.Name}</option>
              ))}
            </select>
          </div>
          <nav className="flex flex-col gap-0.5">
            {views.map((v) => (
              <button
                key={v.id}
                type="button"
                onClick={() => onViewChange(v.id)}
                className={cn(
                  "text-left px-3 py-2 rounded-md text-sm",
                  activeView === v.id ? "bg-[var(--border)]/50 font-medium" : "hover:bg-[var(--border)]/30"
                )}
              >
                {v.label}
              </button>
            ))}
          </nav>
        </aside>
        <section className="flex-1 overflow-auto p-4">{children}</section>
      </main>
    </div>
  );
}
