export interface ClientRx {
  drop_misc: number;
  packets: number;
  bytes: number;
  ht: boolean;
  vht: boolean;
  he: boolean;
  eht: boolean;
  mhz: number;
  rate: number;
}

export interface ClientTx {
  failed: number;
  retries: number;
  packets: number;
  bytes: number;
  ht: boolean;
  vht: boolean;
  he: boolean;
  eht: boolean;
  mhz: number;
  rate: number;
}

export interface DeviceInfo {
  ipaddrs: string[];
  ip6addrs: string[];
  name: string;
}

export interface Client {
  mac: string;
  signal: number;
  signal_avg: number;
  noise: number;
  inactive: number;
  connected_time: number;
  thr: number;
  authorized: boolean;
  authenticated: boolean;
  preamble: string;
  wme: boolean;
  mfp: boolean;
  tdls: boolean;
  "mesh llid": number;
  "mesh plid": number;
  "mesh plink": string;
  "mesh local PS": string;
  "mesh peer PS": string;
  "mesh non-peer PS": string;
  rx: ClientRx;
  tx: ClientTx;
  station_id: string;
  iface: string;
  name: string;
  last_seen: string; // ISO date string
  device_info: DeviceInfo;
}