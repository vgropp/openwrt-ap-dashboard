import { beforeEach, describe, expect, it } from 'vitest';
import { applyTheme, setMode, setTheme, storedMode, storedTheme } from './theme';

describe('theme', () => {
  beforeEach(() => {
    localStorage.clear();
    delete document.documentElement.dataset.theme;
    delete document.documentElement.dataset.mode;
  });

  it('defaults to oldschool, which stays light', () => {
    applyTheme();
    expect(document.documentElement.dataset.theme).toBe('oldschool');
    expect(document.documentElement.dataset.mode).toBe('light');
    setMode('dark');
    expect(document.documentElement.dataset.mode).toBe('light');
  });

  it('applies and remembers theme and mode', () => {
    setTheme('modern');
    setMode('dark');
    expect(storedTheme()).toBe('modern');
    expect(storedMode()).toBe('dark');
    expect(document.documentElement.dataset.theme).toBe('modern');
    expect(document.documentElement.dataset.mode).toBe('dark');
  });

  it('ignores unknown stored values', () => {
    localStorage.setItem('theme', 'fancy');
    localStorage.setItem('mode', 'pink');
    expect(storedTheme()).toBe('oldschool');
    expect(storedMode()).toBe('auto');
  });
});
