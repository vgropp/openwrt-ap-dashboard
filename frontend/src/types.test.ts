import { describe, it, expect } from 'vitest';
import type { Client } from './types';

describe('types', () => {
  it('Client type should exist', () => {
    // Type-only test - verify Client type is importable
    const _typeCheck: Client | undefined = undefined;
    expect(_typeCheck).toBeUndefined();
  });
});
