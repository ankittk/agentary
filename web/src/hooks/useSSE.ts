import { useEffect, useRef, useState } from "react";
import { streamURL } from "@/lib/api";

export function useSSE(onMessage: (data: unknown) => void) {
  const [connected, setConnected] = useState(false);
  const onMessageRef = useRef(onMessage);
  onMessageRef.current = onMessage;

  useEffect(() => {
    const url = streamURL();
    const es = new EventSource(url);

    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        if (data?.type === "connected") {
          setConnected(true);
          return;
        }
        onMessageRef.current(data);
      } catch {
        // ignore parse errors
      }
    };

    return () => {
      es.close();
      setConnected(false);
    };
  }, []);

  return connected;
}
