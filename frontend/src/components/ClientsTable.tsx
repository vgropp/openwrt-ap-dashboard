import { useState } from "react";
import type { Client } from "../types";
import { disconnectClient } from "../api";

interface Props {
  clients: Client[];
  onRefresh: () => void;
}

type SortKey = keyof Client | null;

export default function ClientsTable({ clients, onRefresh }: Props) {
  const handleDisconnect = async (mac: string) => {
    await disconnectClient(mac);
    onRefresh();
  };
  const [filter, setFilter] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("asc");
  const filteredClients = clients.filter((c) => {
    const search = filter.toLowerCase();
    return (
      c.mac.toLowerCase().includes(search) ||
      c.device_info?.name?.toLowerCase().includes(search) ||
      c.device_info?.name?.toLowerCase().includes(search) ||
      c.device_info.ipaddrs.some((ip) => ip.includes(search)) ||
      c.device_info.ip6addrs.some((ip) => ip.includes(search)) ||
      c.iface.toLowerCase().includes(search) ||
      c.name.toLowerCase().includes(search) 
    );
  });

  const sortedClients = [...filteredClients].sort((a, b) => {
    if (!sortKey) return 0;

    const aVal = a[sortKey];
    const bVal = b[sortKey];

    if (typeof aVal === "number" && typeof bVal === "number") {
      return sortOrder === "asc" ? aVal - bVal : bVal - aVal;
    }

    return sortOrder === "asc"
      ? String(aVal).localeCompare(String(bVal))
      : String(bVal).localeCompare(String(aVal));
  });

  function handleSort(key: SortKey) {
    if (sortKey === key) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortOrder("asc");
    }
  }

  function SortIndicator({ column }: { column: SortKey }) {
    if (sortKey !== column) return null;
    return sortOrder === "asc" ? <span className="ml-1 text-blue-500">▲</span> : <span className="ml-1 text-blue-500">▼</span>;
  }
  return (
    <div className="space-y-3">

      <div className="flex items-center justify-between">
        <p className="text-sm text-gray-600">
          Showing <span className="font-semibold">{filteredClients.length}</span>{" "}
          of <span className="font-semibold">{clients.length}</span> clients
        </p>
        <input
          type="text"
          placeholder="Filter clients..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="px-3 py-1 border border-gray-300 rounded-lg shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-400"
        />
      </div>

      <div className="overflow-x-auto">
        <table className="min-w-full border border-gray-200 bg-white shadow-md rounded-lg">
          <thead className="bg-gray-100 text-gray-700 text-sm uppercase">
            <tr>
              <th className="px-4 py-2 text-left cursor-pointer whitespace-nowrap" onClick={() => handleSort("iface")}>
                Network <SortIndicator column="iface" />
              </th>
              <th className="px-4 py-2 text-left cursor-pointer whitespace-nowrap" onClick={() => handleSort("name")}>
                AP <SortIndicator column="name" />
              </th>
              <th className="px-4 py-2 text-left cursor-pointer whitespace-nowrap" onClick={() => handleSort("mac")}>
                MAC Address <SortIndicator column="mac" />
              </th>
              <th className="px-4 py-2 text-left cursor-pointer" onClick={() => handleSort("device_info")}>
                Host {/* tricky: device_info is nested, see note below */}
              </th>
              <th className="px-4 py-2 text-left cursor-pointer whitespace-nowrap" onClick={() => handleSort("signal")}>
                Signal <SortIndicator column="signal" />
              </th>
              <th className="px-4 py-2 text-left cursor-pointer whitespace-nowrap" onClick={() => handleSort("thr")}>
                Throughput <SortIndicator column="thr" />
              </th>
              <th className="px-4 py-2"></th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 text-sm">
            {sortedClients.map((c) => (
              <tr key={c.mac} className="hover:bg-gray-50 transition-colors duration-150">
                <td className="px-4 py-2">
                  <div className="flex items-center space-x-2">
                    <span>📡 {c.iface}</span>
                  </div>
                </td>
                <td className="px-4 py-2 font-mono">{c.name}</td>
                <td className="px-4 py-2 font-mono">{c.mac}</td>
                <td className="px-4 py-2 text-sm text-gray-700 relative group">
                  <span>
                    {c.device_info.name}
                    {(() => {
                      const ip4 = c.device_info.ipaddrs[0];
                      const ip6 = c.device_info.ip6addrs[0];
                      if (ip4 || ip6) {
                        return ` (${[ip4, ip6].filter(Boolean).join(", ")})`;
                      }
                      return "";
                    })()}
                  </span>
                  {(c.device_info.ipaddrs.length > 0 || c.device_info.ip6addrs.length > 0) && (
                    <div className="absolute z-10 hidden group-hover:block bg-gray-800 text-white text-xs rounded px-2 py-1 mt-1 left-1/2 -translate-x-1/2 whitespace-pre shadow-lg">
                      {[...c.device_info.ipaddrs, ...c.device_info.ip6addrs].join("\n")}
                    </div>
                  )}
                </td>
                <td className="px-4 py-2">
                  <div className="flex items-center space-x-2">
                    <SignalBar signal={c.signal} />
                    <span>
                      {c.signal} / {c.noise} dBm
                    </span>
                  </div>
                </td>
                <td className="px-4 py-2">
                  <div className="flex flex-col">
                    <span>
                      {formatRate(c.rx.rate)} Mbit/s, {c.rx.mhz} MHz
                    </span>
                    <span>
                      {formatRate(c.tx.rate)} Mbit/s, {c.tx.mhz} MHz
                    </span>
                  </div>
                </td>
                <td className="px-4 py-2 text-right">
                  <button
                    onClick={() => handleDisconnect(c.mac)}
                    className="px-3 py-1 text-red-600 border border-red-400 rounded-md hover:bg-red-50 hover:text-red-700 transition">
                    Disconnect
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function formatRate(rate: number): string {
  return (rate / 1000).toFixed(1);
}

function SignalBar({ signal }: { signal: number }) {
  const levels = [-30, -50, -60, -70, -80]; // thresholds
  const strength = levels.filter((lvl) => signal >= lvl).length;

  return (
    <div className="flex space-x-0.5 items-end">
      {[1, 2, 3, 4].map((bar) => (
        <div
          key={bar}
          className={`w-1.5 rounded-sm ${
            bar <= strength ? "bg-blue-600" : "bg-gray-300"
          }`}
          style={{ height: `${bar * 5}px` }}
        />
      ))}
    </div>
  );
}
