import { useEffect, useState } from "react";
import { fetchClients } from "./api";
import type { Client } from "./types";
import ClientsTable from "./components/ClientsTable";

function App() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

// State as a Map
const [clients, setClients] = useState<Map<string, Client>>(new Map());

  const load = async () => {
    try {
      setError(null);
      const t = setTimeout(() => setLoading(true), 750);

      const data = await fetchClients();
      setClients((prev) => {
        const next = new Map(prev); // copy old map

        let changed = false;
        for (const c of data) {
          const existing = prev.get(c.mac);
          if (!existing || JSON.stringify(existing) !== JSON.stringify(c)) {
            next.set(c.mac, c);
            changed = true;
          }
        }

        // Remove any clients that no longer exist
        for (const mac of prev.keys()) {
          if (!data.find((c) => c.mac === mac)) {
            next.delete(mac);
            changed = true;
          }
        }
        clearTimeout(t);
        return changed ? next : prev;
      });
    } catch (e: unknown) {
      if (e instanceof Error) {
        setError(e.message);
      } else {
        setError(String(e));
      }
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
    const interval = setInterval(load, 2500); //
    return () => clearInterval(interval); // cleanup on unmount    
  }, []);

  return (
    <div className="container mx-auto mt-8">
      <div className={`absolute inset-0 bg-white/70 flex items-center justify-center z-10 transition-opacity duration-200 ${loading ? "opacity-100" : "opacity-0 pointer-events-none"}`}>
        <div className={`flex items-center space-x-2 animate-pulse`}>
          <div className="w-4 h-4 bg-blue-500 rounded-full"></div>
          <div className="w-4 h-4 bg-blue-500 rounded-full"></div>
          <div className="w-4 h-4 bg-blue-500 rounded-full"></div>
          <span className="ml-2 font-medium text-blue-700">Loading...</span>
        </div>
      </div>
      {error && <p className="text-red-500">{error}</p>}
      {!error && (
        <ClientsTable clients={Array.from(clients.values())} onRefresh={load} />
      )}
    </div>
  );
}

export default App;
