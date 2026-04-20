import { SchedulesViewState } from './types';

type ScheduleViewStateParams = {
  authReady: boolean;
  authLoading: boolean;
  hasSession: boolean;
  isFetching: boolean;
  fetchError: string | null;
  scheduleCount: number;
};

export function getSchedulesViewState(params: ScheduleViewStateParams): SchedulesViewState {
  if (!params.authReady) {
    return 'setup_required';
  }

  if (params.authLoading) {
    return 'auth_loading';
  }

  if (!params.hasSession) {
    return 'signed_out';
  }

  if (params.isFetching && params.scheduleCount === 0) {
    return 'loading';
  }

  if (params.fetchError) {
    return 'error';
  }

  if (params.scheduleCount === 0) {
    return 'empty';
  }

  return 'ready';
}
