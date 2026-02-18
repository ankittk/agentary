import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function getApiBase(): string {
  const base = import.meta.env.VITE_API_BASE;
  if (base) return base;
  return `${window.location.protocol}//${window.location.host}`;
}
