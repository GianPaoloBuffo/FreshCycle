import { describe, expect, it } from 'vitest';

import { getSchedulesViewState } from './viewState';

describe('getSchedulesViewState', () => {
  it('returns setup_required when auth is not configured', () => {
    expect(
      getSchedulesViewState({
        authReady: false,
        authLoading: false,
        hasSession: false,
        isFetching: false,
        fetchError: null,
        scheduleCount: 0,
      })
    ).toBe('setup_required');
  });

  it('returns auth_loading while the session is being restored', () => {
    expect(
      getSchedulesViewState({
        authReady: true,
        authLoading: true,
        hasSession: false,
        isFetching: false,
        fetchError: null,
        scheduleCount: 0,
      })
    ).toBe('auth_loading');
  });

  it('returns signed_out when no session is available', () => {
    expect(
      getSchedulesViewState({
        authReady: true,
        authLoading: false,
        hasSession: false,
        isFetching: false,
        fetchError: null,
        scheduleCount: 0,
      })
    ).toBe('signed_out');
  });

  it('returns loading for an authenticated first fetch', () => {
    expect(
      getSchedulesViewState({
        authReady: true,
        authLoading: false,
        hasSession: true,
        isFetching: true,
        fetchError: null,
        scheduleCount: 0,
      })
    ).toBe('loading');
  });

  it('returns error when a fetch failed', () => {
    expect(
      getSchedulesViewState({
        authReady: true,
        authLoading: false,
        hasSession: true,
        isFetching: false,
        fetchError: 'boom',
        scheduleCount: 0,
      })
    ).toBe('error');
  });

  it('returns empty when the user has no schedules yet', () => {
    expect(
      getSchedulesViewState({
        authReady: true,
        authLoading: false,
        hasSession: true,
        isFetching: false,
        fetchError: null,
        scheduleCount: 0,
      })
    ).toBe('empty');
  });

  it('returns ready when schedules are available', () => {
    expect(
      getSchedulesViewState({
        authReady: true,
        authLoading: false,
        hasSession: true,
        isFetching: false,
        fetchError: null,
        scheduleCount: 2,
      })
    ).toBe('ready');
  });
});
