import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";

type Theme = "light" | "dark" | "system";

const ThemeContext = createContext<{
  theme: Theme;
  setTheme: (t: Theme) => void;
  resolved: "light" | "dark";
} | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(() => (typeof localStorage !== "undefined" ? (localStorage.getItem("agentary-theme") as Theme) || "system" : "system"));
  const [resolved, setResolved] = useState<"light" | "dark">("light");

  const setTheme = useCallback((t: Theme) => {
    setThemeState(t);
    if (typeof localStorage !== "undefined") localStorage.setItem("agentary-theme", t);
  }, []);

  useEffect(() => {
    const root = document.documentElement;
    const isDark = theme === "dark" || (theme === "system" && typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches);
    setResolved(isDark ? "dark" : "light");
    root.classList.toggle("dark", isDark);
  }, [theme]);

  return (
    <ThemeContext.Provider value={{ theme, setTheme, resolved }}>
      {children}
    </ThemeContext.Provider>
  );
}

export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error("useTheme must be used within ThemeProvider");
  return ctx;
}

export function ThemeToggle({ className }: { className?: string }) {
  const { setTheme, resolved } = useTheme();
  return (
    <button
      type="button"
      aria-label="Toggle theme"
      className={cn("rounded-md p-2 hover:bg-[var(--border)]/50", className)}
      onClick={() => setTheme(resolved === "dark" ? "light" : "dark")}
    >
      {resolved === "dark" ? "☀️" : "🌙"}
    </button>
  );
}
