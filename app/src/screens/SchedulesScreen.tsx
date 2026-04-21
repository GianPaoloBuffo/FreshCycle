import { Link } from 'expo-router';
import { startTransition, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { cancelLocalNotificationsForSchedule } from '@/features/notifications/localNotifications';
import { deleteSchedule } from '@/features/schedules/deleteSchedule';
import { fetchSchedules } from '@/features/schedules/fetchSchedules';
import { logSchedulesError, logSchedulesEvent } from '@/features/schedules/observability';
import { LaundrySchedule } from '@/features/schedules/types';
import { getSchedulesViewState } from '@/features/schedules/viewState';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

const env = getAppEnv();

export function SchedulesScreen() {
  const { authReady, loading, session } = useAuth();
  const [schedules, setSchedules] = useState<LaundrySchedule[]>([]);
  const [isFetching, setIsFetching] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [deletingScheduleId, setDeletingScheduleId] = useState<string | null>(null);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const lastLoadedAccessTokenRef = useRef<string | null>(null);
  const hasSession = Boolean(session);
  const viewState = getSchedulesViewState({
    authReady,
    authLoading: loading,
    hasSession,
    isFetching,
    fetchError,
    scheduleCount: schedules.length,
  });

  useEffect(() => {
    logSchedulesEvent('schedules_navigation_opened', {
      authReady,
      hasSession,
    });
  }, [authReady, hasSession]);

  useEffect(() => {
    if (!authReady || loading || !session) {
      setSchedules([]);
      setFetchError(null);
      setIsFetching(false);
      setIsRefreshing(false);
      lastLoadedAccessTokenRef.current = null;
      return;
    }

    const accessToken = session.access_token ?? null;
    if (!accessToken || lastLoadedAccessTokenRef.current === accessToken) {
      return;
    }

    lastLoadedAccessTokenRef.current = accessToken;
    void loadSchedules(accessToken, 'initial');
  }, [authReady, loading, session]);

  async function loadSchedules(accessToken: string, reason: 'initial' | 'refresh') {
    if (reason === 'refresh') {
      setIsRefreshing(true);
    } else {
      setIsFetching(true);
      setFetchError(null);
    }

    logSchedulesEvent('schedules_fetch_started', {
      reason,
      hasExistingSchedules: schedules.length > 0,
    });

    try {
      const nextSchedules = await fetchSchedules({
        accessToken,
      });

      startTransition(() => {
        setSchedules(nextSchedules);
        setFetchError(null);
        setDeleteError(null);
      });

      logSchedulesEvent('schedules_fetch_succeeded', {
        reason,
        scheduleCount: nextSchedules.length,
      });
    } catch (error) {
      setFetchError(
        reason === 'refresh'
          ? 'FreshCycle could not refresh your schedules right now. Try again in a moment.'
          : describeSchedulesError(error)
      );

      logSchedulesError('schedules_fetch_failed', error, {
        reason,
        hasExistingSchedules: schedules.length > 0,
      });
    } finally {
      setIsFetching(false);
      setIsRefreshing(false);
    }
  }

  async function handleDeleteSchedule(scheduleId: string) {
    if (!session?.access_token) {
      return;
    }

    setDeletingScheduleId(scheduleId);
    setDeleteError(null);
    logSchedulesEvent('schedule_delete_started', {
      scheduleId,
    });

    try {
      await deleteSchedule(scheduleId, {
        accessToken: session.access_token,
      });
      await cancelLocalNotificationsForSchedule(scheduleId);

      startTransition(() => {
        setSchedules((currentSchedules) =>
          currentSchedules.filter((schedule) => schedule.id !== scheduleId)
        );
      });

      logSchedulesEvent('schedule_delete_succeeded', {
        scheduleId,
      });
    } catch (error) {
      setDeleteError(describeDeleteError(error));
      logSchedulesError('schedule_delete_failed', error, {
        scheduleId,
      });
    } finally {
      setDeletingScheduleId(null);
    }
  }

  return (
    <AppScreen>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          hasSession ? (
            <RefreshControl
              refreshing={isRefreshing}
              onRefresh={() => {
                if (!session?.access_token) {
                  return;
                }

                void loadSchedules(session.access_token, 'refresh');
              }}
              tintColor={palette.accent}
            />
          ) : undefined
        }>
        <View style={styles.hero}>
          <Text style={styles.eyebrow}>Schedules</Text>
          <Text style={styles.title}>Keep repeat laundry jobs visible before they turn into pileups.</Text>
          <Text style={styles.body}>
            Review recurring laundry plans, see whether reminders are enabled, and jump into creating
            a new schedule from one place.
          </Text>
        </View>

        <View style={[styles.card, styles.bannerCard]}>
          <View style={styles.bannerCopy}>
            <Text style={styles.cardTitle}>New schedule</Text>
            <Text style={styles.meta}>
              Build a recurring plan for towels, bedding, uniforms, or anything else that needs a
              steady wash rhythm.
            </Text>
          </View>
          <Link href={'/new-schedule' as never} style={styles.primaryLink}>
            Create schedule
          </Link>
        </View>

        {viewState === 'setup_required' ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Setup required</Text>
            <Text style={styles.meta}>
              Add `EXPO_PUBLIC_SUPABASE_URL` and `EXPO_PUBLIC_SUPABASE_KEY` to enable auth.
            </Text>
            <Text style={styles.meta}>API base URL: {env.apiBaseUrl ?? 'not configured'}</Text>
          </View>
        ) : null}

        {viewState === 'auth_loading' ? (
          <View style={styles.card}>
            <ActivityIndicator color={palette.accent} />
            <Text style={styles.meta}>Checking your session...</Text>
          </View>
        ) : null}

        {viewState === 'signed_out' ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Sign in to view your schedules</Text>
            <Text style={styles.meta}>
              Schedules are private to your account, so authenticate before loading recurring plans.
            </Text>
            <Link href={'/auth' as never} style={styles.link}>
              Open auth screens
            </Link>
          </View>
        ) : null}

        {viewState === 'loading' ? (
          <View style={styles.card}>
            <ActivityIndicator color={palette.accent} />
            <Text style={styles.meta}>Loading your schedules...</Text>
          </View>
        ) : null}

        {viewState === 'error' ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Schedules unavailable</Text>
            <Text style={styles.meta}>{fetchError}</Text>
            <Pressable
              onPress={() => {
                if (!session?.access_token) {
                  return;
                }

                void loadSchedules(session.access_token, 'initial');
              }}
              style={styles.primaryButton}>
              <Text style={styles.primaryButtonText}>Try again</Text>
            </Pressable>
          </View>
        ) : null}

        {deleteError ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Schedule not deleted</Text>
            <Text style={styles.meta}>{deleteError}</Text>
          </View>
        ) : null}

        {viewState === 'empty' ? (
          <View style={[styles.card, styles.emptyStateCard]}>
            <Text style={styles.cardTitle}>No schedules yet</Text>
            <Text style={styles.meta}>
              Start with one repeat laundry rhythm so FreshCycle can surface it here once reminders
              and due-today views come online.
            </Text>
            <Link href={'/new-schedule' as never} style={styles.primaryLink}>
              Create your first schedule
            </Link>
          </View>
        ) : null}

        {viewState === 'ready' ? (
          <>
            <View style={styles.sectionHeader}>
              <Text style={styles.cardTitle}>Saved schedules</Text>
              <Text style={styles.meta}>
                {schedules.length} schedule{schedules.length === 1 ? '' : 's'}
              </Text>
            </View>

            {schedules.map((schedule) => (
              <View key={schedule.id} style={styles.card}>
                <View style={styles.scheduleHeader}>
                  <Text style={styles.scheduleName}>{schedule.name}</Text>
                  <Text
                    style={[
                      styles.badge,
                      schedule.reminders_enabled ? styles.badgeEnabled : styles.badgeDisabled,
                    ]}>
                    {schedule.reminders_enabled ? 'Reminders on' : 'Reminders off'}
                  </Text>
                </View>

                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Recurrence</Text>
                  <Text style={styles.detailValue}>{formatRecurrence(schedule.recurrence)}</Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Tracked garments</Text>
                  <Text style={styles.detailValue}>
                    {schedule.garment_ids.length} garment{schedule.garment_ids.length === 1 ? '' : 's'}
                  </Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Created</Text>
                  <Text style={styles.detailValue}>{formatCreatedAt(schedule.created_at)}</Text>
                </View>
                <Pressable
                  disabled={deletingScheduleId === schedule.id}
                  onPress={() => void handleDeleteSchedule(schedule.id)}
                  style={styles.deleteButton}>
                  <Text style={styles.deleteButtonText}>
                    {deletingScheduleId === schedule.id ? 'Deleting...' : 'Delete schedule'}
                  </Text>
                </Pressable>
              </View>
            ))}
          </>
        ) : null}

        <View style={styles.footerLinks}>
          <Link href={'/new-schedule' as never} style={styles.link}>
            Start a new schedule
          </Link>
          <Link href={'/' as never} style={styles.linkSecondary}>
            Back to wardrobe
          </Link>
        </View>
      </ScrollView>
    </AppScreen>
  );
}

function describeSchedulesError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'auth-required':
        return 'Sign in again before loading your schedules.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to load schedules from.';
      case 'not-ready':
        return 'Schedule sync is not live on this environment yet. The screen is ready, but the schedules API arrives later in Phase 4.';
      default:
        return 'FreshCycle could not load your schedules right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not load your schedules right now. Try again in a moment.';
}

function describeDeleteError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'auth-required':
        return 'Sign in again before deleting schedules.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to delete schedules.';
      case 'invalid-schedule-id':
        return 'FreshCycle could not delete that schedule because its id was invalid.';
      case 'schedule-not-found':
        return 'That schedule was already removed or no longer belongs to this account.';
      case 'local-notification-cancel-failed':
        return 'The schedule was deleted, but FreshCycle could not cancel local reminders on this device.';
      default:
        return 'FreshCycle could not delete that schedule right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not delete that schedule right now. Try again in a moment.';
}

function formatRecurrence(recurrence: string) {
  if (recurrence === 'daily') {
    return 'Daily';
  }

  if (recurrence === 'fortnightly') {
    return 'Fortnightly';
  }

  if (recurrence.startsWith('weekly:')) {
    const day = recurrence.split(':')[1] ?? 'day';
    return `Weekly on ${day.charAt(0).toUpperCase()}${day.slice(1)}`;
  }

  return recurrence;
}

function formatCreatedAt(value: string) {
  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return 'Recently added';
  }

  return date.toLocaleDateString(undefined, {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

const styles = StyleSheet.create({
  scrollContent: {
    paddingBottom: 24,
  },
  hero: {
    gap: 12,
    marginBottom: 20,
  },
  eyebrow: {
    color: palette.accent,
    fontSize: 13,
    fontWeight: '700',
    letterSpacing: 1.1,
    textTransform: 'uppercase',
  },
  title: {
    color: palette.ink,
    fontSize: 34,
    fontWeight: '700',
    lineHeight: 40,
  },
  body: {
    color: palette.inkMuted,
    fontSize: 16,
    lineHeight: 24,
  },
  card: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 20,
    borderWidth: 1,
    gap: 10,
    marginBottom: 16,
    padding: 20,
  },
  bannerCard: {
    backgroundColor: '#e5eddc',
    gap: 16,
  },
  bannerCopy: {
    gap: 6,
  },
  emptyStateCard: {
    backgroundColor: '#efe8da',
  },
  sectionHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 12,
    marginTop: 4,
  },
  cardTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  scheduleHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  scheduleName: {
    color: palette.ink,
    flex: 1,
    fontSize: 22,
    fontWeight: '700',
  },
  badge: {
    borderRadius: 999,
    color: palette.ink,
    fontSize: 12,
    fontWeight: '700',
    overflow: 'hidden',
    paddingHorizontal: 10,
    paddingVertical: 6,
    textTransform: 'uppercase',
  },
  badgeEnabled: {
    backgroundColor: palette.accentSoft,
  },
  badgeDisabled: {
    backgroundColor: '#e7dac2',
  },
  detailRow: {
    gap: 4,
  },
  detailLabel: {
    color: palette.inkMuted,
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.6,
    textTransform: 'uppercase',
  },
  detailValue: {
    color: palette.ink,
    fontSize: 15,
    lineHeight: 22,
  },
  meta: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  primaryButton: {
    alignSelf: 'flex-start',
    backgroundColor: palette.ink,
    borderRadius: 999,
    marginTop: 8,
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  primaryButtonText: {
    color: palette.canvas,
    fontSize: 16,
    fontWeight: '700',
  },
  deleteButton: {
    alignSelf: 'flex-start',
    borderColor: '#b86b5f',
    borderRadius: 999,
    borderWidth: 1,
    marginTop: 8,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  deleteButtonText: {
    color: '#9b4334',
    fontSize: 15,
    fontWeight: '700',
  },
  primaryLink: {
    alignSelf: 'flex-start',
    backgroundColor: palette.ink,
    borderRadius: 999,
    color: palette.canvas,
    fontSize: 16,
    fontWeight: '700',
    overflow: 'hidden',
    paddingHorizontal: 18,
    paddingVertical: 14,
  },
  link: {
    color: palette.ink,
    fontSize: 16,
    fontWeight: '600',
    marginTop: 8,
  },
  linkSecondary: {
    color: palette.inkMuted,
    fontSize: 15,
    fontWeight: '600',
    marginTop: 12,
  },
  footerLinks: {
    gap: 8,
    marginTop: 8,
  },
});
