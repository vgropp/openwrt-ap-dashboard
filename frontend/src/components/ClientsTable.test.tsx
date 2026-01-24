import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import ClientsTable from './ClientsTable';
import * as api from '../api';

const nowIso = new Date().toISOString();
const oldIso = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();

const clients = [
  {
    mac: 'AA:BB:CC:DD:EE:01',
    signal: -40,
    signal_avg: -42,
    noise: -90,
    inactive: 0,
    connected_time: 1000,
    thr: 100,
    authorized: true,
    authenticated: true,
    preamble: '',
    wme: true,
    mfp: false,
    tdls: false,
    "mesh llid": 0,
    "mesh plid": 0,
    "mesh plink": '',
    "mesh local PS": '',
    "mesh peer PS": '',
    "mesh non-peer PS": '',
    rx: { drop_misc: 0, packets: 10, bytes: 1000, ht: true, vht: false, he: false, eht: false, mhz: 20, rate: 1000 },
    tx: { failed: 0, retries: 0, packets: 10, bytes: 1000, ht: true, vht: false, he: false, eht: false, mhz: 20, rate: 1000 },
    station_id: 'sta1',
    iface: 'wlan0',
    name: 'AP1',
    last_seen: nowIso,
    device_info: { ipaddrs: ['192.168.1.2'], ip6addrs: [], name: 'host1' },
  },
  {
    mac: 'AA:BB:CC:DD:EE:02',
    signal: -70,
    signal_avg: -72,
    noise: -90,
    inactive: 0,
    connected_time: 1000,
    thr: 50,
    authorized: true,
    authenticated: true,
    preamble: '',
    wme: true,
    mfp: false,
    tdls: false,
    "mesh llid": 0,
    "mesh plid": 0,
    "mesh plink": '',
    "mesh local PS": '',
    "mesh peer PS": '',
    "mesh non-peer PS": '',
    rx: { drop_misc: 0, packets: 10, bytes: 1000, ht: true, vht: false, he: false, eht: false, mhz: 20, rate: 1000 },
    tx: { failed: 0, retries: 0, packets: 10, bytes: 1000, ht: true, vht: false, he: false, eht: false, mhz: 20, rate: 1000 },
    station_id: 'sta2',
    iface: 'wlan1',
    name: 'AP2',
    last_seen: oldIso,
    device_info: { ipaddrs: ['192.168.1.3'], ip6addrs: [], name: 'host2' },
  },
];

describe('ClientsTable', () => {
  it('renders all clients when "Hide > 1 hr inactive" is unchecked', () => {
    render(<ClientsTable clients={clients} onRefresh={() => {}} />);
    const hideOld = screen.getByLabelText(/Hide > 1 hr inactive/i);
    fireEvent.click(hideOld); // uncheck to show all
    expect(screen.getByText('AP1')).toBeInTheDocument();
    expect(screen.getByText('AP2')).toBeInTheDocument();
  });

  it('hides old clients when "Hide > 1 hr inactive" is checked', () => {
    render(<ClientsTable clients={clients} onRefresh={() => {}} />);
    // By default, old clients are hidden
    expect(screen.getByText('AP1')).toBeInTheDocument();
    expect(screen.queryByText('AP2')).toBeNull();
    // Uncheck and check again to verify toggle
    const hideOld = screen.getByLabelText(/Hide > 1 hr inactive/i);
    fireEvent.click(hideOld); // uncheck (show all)
    expect(screen.getByText('AP2')).toBeInTheDocument();
    fireEvent.click(hideOld); // check (hide old)
    expect(screen.queryByText('AP2')).toBeNull();
  });

  it('filters clients by MAC', () => {
    render(<ClientsTable clients={clients} onRefresh={() => {}} />);
    const input = screen.getByPlaceholderText(/Filter clients/i);
    fireEvent.change(input, { target: { value: 'ee:01' } });
    expect(screen.getByText('AP1')).toBeInTheDocument();
    expect(screen.queryByText('AP2')).toBeNull();
  });

  it('sorts clients by signal', () => {
    render(<ClientsTable clients={clients} onRefresh={() => {}} />);
    // Uncheck hide old to show both clients
    const hideOld = screen.getByLabelText(/Hide > 1 hr inactive/i);
    fireEvent.click(hideOld);
    const signalHeader = screen.getByText(/Signal/i);
    fireEvent.click(signalHeader); // sort asc
    // AP2 (-70) should be before AP1 (-40) in asc order
    const rows = screen.getAllByRole('row');
    const apNames = rows.map(row => row.textContent).filter(Boolean);
    expect(apNames.join(' ')).toMatch(/AP2.*AP1/);
    fireEvent.click(signalHeader); // sort desc
    const rowsDesc = screen.getAllByRole('row');
    const apNamesDesc = rowsDesc.map(row => row.textContent).filter(Boolean);
    expect(apNamesDesc.join(' ')).toMatch(/AP1.*AP2/);
  });

  it('shows disconnect button for each client', () => {
    render(<ClientsTable clients={clients} onRefresh={() => {}} />);
    const hideOld = screen.getByLabelText(/Hide > 1 hr inactive/i);
    fireEvent.click(hideOld);
    expect(screen.getAllByText('Disconnect').length).toBe(2);
  });

  it('calls onRefresh after disconnect', async () => {
    const onRefresh = vi.fn();
    vi.spyOn(api, 'disconnectClient').mockResolvedValue(undefined);
    render(<ClientsTable clients={clients} onRefresh={onRefresh} />);
    const hideOld = screen.getByLabelText(/Hide > 1 hr inactive/i);
    fireEvent.click(hideOld);
    const btn = screen.getAllByText('Disconnect')[0];
    await fireEvent.click(btn);
    expect(onRefresh).toHaveBeenCalled();
  });
});
