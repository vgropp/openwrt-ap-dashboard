import { describe, it, expect } from 'vitest';
import * as api from './api';

describe('api', () => {
  it('should export fetchClients', () => {
    expect(api.fetchClients).toBeInstanceOf(Function);
  });
});
