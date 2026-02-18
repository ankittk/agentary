import { useCallback, useEffect, useState } from "react";
import { ThemeProvider } from "@/ThemeProvider";
import { Layout } from "@/components/Layout";
import { KanbanBoard } from "@/components/KanbanBoard";
import { AgentMonitor } from "@/components/AgentMonitor";
import { WorkflowVisualizer } from "@/components/WorkflowVisualizer";
import { TaskChart } from "@/components/TaskChart";
import { ChatPanel } from "@/components/ChatPanel";
import { ReviewPanel } from "@/components/ReviewPanel";
import { NetworkConfig } from "@/components/NetworkConfig";
import { SettingsPanel } from "@/components/SettingsPanel";
import { AgentMemory } from "@/components/AgentMemory";
import { CharterEditor } from "@/components/CharterEditor";
import {
  fetchBootstrap,
  fetchTasks,
  fetchAgents,
  type Bootstrap,
  type Task,
  type Agent,
  type TeamSummary,
} from "@/lib/api";
import { useSSE } from "@/hooks/useSSE";

function AppContent() {
  const [team, setTeam] = useState("");
  const [teams, setTeams] = useState<TeamSummary[]>([]);
  const [tasks, setTasks] = useState<Task[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [view, setView] = useState("kanban");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    if (!team) return;
    setLoading(true);
    setError(null);
    try {
      const [tList, aList] = await Promise.all([fetchTasks(team), fetchAgents(team)]);
      setTasks(tList);
      setAgents(aList);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [team]);

  useEffect(() => {
    let cancelled = false;
    fetchBootstrap()
      .then((data: Bootstrap) => {
        if (cancelled) return;
        const teamList = data.teams || [];
        setTeams(teamList);
        const initial = data.initial_team || (teamList[0]?.Name ?? "");
        setTeam(initial);
        if (initial) {
          setTasks((data.tasks as Task[]) || []);
          setAgents((data.agents as Agent[]) || []);
        }
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : "Failed to load bootstrap");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    if (team && teams.length > 0) refresh();
  }, [team, teams.length, refresh]);

  useSSE((data: unknown) => {
    const d = data as { type?: string; team?: string; task_id?: number };
    if (d?.type === "task_update" && d?.team === team) refresh();
    if (d?.type === "team_update") {
      fetchBootstrap().then((b: Bootstrap) => setTeams(b.teams || []));
    }
  });

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center p-4">
        <div className="text-center text-red-500">
          <p className="font-medium">Error</p>
          <p className="text-sm mt-1">{error}</p>
        </div>
      </div>
    );
  }

  if (teams.length === 0 && !loading) {
    return (
      <div className="min-h-screen flex flex-col items-center justify-center p-4 gap-4">
        <img src="/logo-animated.svg" alt="" className="max-w-[280px] h-20 w-auto opacity-90" aria-hidden />
        <p className="text-[var(--muted)]">No teams. Create a team via API or CLI.</p>
      </div>
    );
  }

  return (
    <Layout
      team={team}
      teams={teams}
      onTeamChange={(t) => setTeam(t)}
      activeView={view}
      onViewChange={setView}
    >
      {loading && !tasks.length && !agents.length ? (
        <div className="flex-1 flex items-center justify-center p-8">
          <img src="/logo-animated.svg" alt="" className="max-w-full h-24 w-auto opacity-90" aria-hidden />
        </div>
      ) : (
        <>
          {view === "kanban" && <KanbanBoard team={team} tasks={tasks} onUpdate={refresh} />}
          {view === "agents" && <AgentMonitor agents={agents} />}
          {view === "workflow" && <WorkflowVisualizer />}
          {view === "charts" && <TaskChart tasks={tasks} />}
          {view === "chat" && <ChatPanel team={team} />}
          {view === "reviews" && <ReviewPanel team={team} onUpdate={refresh} />}
          {view === "network" && <NetworkConfig />}
          {view === "charter" && <CharterEditor team={team} onUpdate={refresh} />}
          {view === "memory" && <AgentMemory team={team} agents={agents} />}
          {view === "settings" && <SettingsPanel />}
        </>
      )}
    </Layout>
  );
}

export default function App() {
  return (
    <ThemeProvider>
      <AppContent />
    </ThemeProvider>
  );
}
