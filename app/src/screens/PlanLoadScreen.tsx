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
import { logLoadPlanningError, logLoadPlanningEvent } from '@/features/load-planning/observability';
import { planLoads } from '@/features/load-planning/planLoads';
import { LoadPlanningMode, PlannedLoadSummary } from '@/features/load-planning/types';
import { fetchGarments } from '@/features/wardrobe/fetchGarments';
import { logWardrobeError, logWardrobeEvent } from '@/features/wardrobe/observability';
import { WardrobeGarment } from '@/features/wardrobe/types';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

const env = getAppEnv();
const planningModes: Array<{ label: string; value: LoadPlanningMode }> = [
  { label: 'Smart loads', value: 'smart' },
  { label: 'By colour', value: 'colour' },
  { label: 'By temperature', value: 'temperature' },
  { label: 'By category', value: 'category' },
];

export function PlanLoadScreen() {
  const { authReady, loading, session } = useAuth();
  const [garments, setGarments] = useState<WardrobeGarment[]>([]);
  const [isFetching, setIsFetching] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [planningMode, setPlanningMode] = useState<LoadPlanningMode>('smart');
  const [loads, setLoads] = useState<PlannedLoadSummary[]>([]);
  const [planningError, setPlanningError] = useState<string | null>(null);
  const canLoadWardrobe = authReady && !loading && Boolean(session);
  const lastLoadedAccessTokenRef = useRef<string | null>(null);

  useEffect(() => {
    if (!canLoadWardrobe) {
      setGarments([]);
      setFetchError(null);
      setIsFetching(false);
      setIsRefreshing(false);
      lastLoadedAccessTokenRef.current = null;
      return;
    }

    const accessToken = session?.access_token ?? null;
    if (!accessToken || lastLoadedAccessTokenRef.current === accessToken) {
      return;
    }

    lastLoadedAccessTokenRef.current = accessToken;
    void loadGarments(accessToken, 'initial');
  }, [canLoadWardrobe, session?.access_token]);

  useEffect(() => {
    try {
      const nextLoads = planLoads(garments, planningMode);
      startTransition(() => {
        setLoads(nextLoads);
        setPlanningError(null);
      });

      logLoadPlanningEvent('load_planning_generated', {
        mode: planningMode,
        garmentCount: garments.length,
        loadCount: nextLoads.length,
      });
    } catch (error) {
      setLoads([]);
      setPlanningError('FreshCycle could not generate load summaries right now.');
      logLoadPlanningError('load_planning_generation_failed', error, {
        mode: planningMode,
        garmentCount: garments.length,
      });
    }
  }, [garments, planningMode]);

  async function loadGarments(accessToken: string, reason: 'initial' | 'refresh') {
    if (reason === 'refresh') {
      setIsRefreshing(true);
      logWardrobeEvent('wardrobe_refresh_started', {
        hasExistingGarments: garments.length > 0,
        surface: 'plan-load',
      });
    } else {
      setIsFetching(true);
      setFetchError(null);
      logWardrobeEvent('wardrobe_fetch_started', {
        hasExistingGarments: garments.length > 0,
        surface: 'plan-load',
      });
    }

    try {
      const nextGarments = await fetchGarments({
        accessToken,
      });

      startTransition(() => {
        setGarments(nextGarments);
        setFetchError(null);
      });

      logWardrobeEvent(reason === 'refresh' ? 'wardrobe_refresh_succeeded' : 'wardrobe_fetch_succeeded', {
        garmentCount: nextGarments.length,
        surface: 'plan-load',
      });
    } catch (error) {
      setFetchError(
        reason === 'refresh'
          ? 'FreshCycle could not refresh your wardrobe right now. Pull down again in a moment.'
          : describeWardrobeError(error)
      );
      logWardrobeError(reason === 'refresh' ? 'wardrobe_refresh_failed' : 'wardrobe_fetch_failed', error, {
        reason,
        surface: 'plan-load',
      });
    } finally {
      setIsFetching(false);
      setIsRefreshing(false);
    }
  }

  return (
    <AppScreen>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          canLoadWardrobe ? (
            <RefreshControl
              refreshing={isRefreshing}
              onRefresh={() => {
                if (!session?.access_token) {
                  return;
                }

                void loadGarments(session.access_token, 'refresh');
              }}
              tintColor={palette.accent}
            />
          ) : undefined
        }>
        <View style={styles.hero}>
          <Text style={styles.eyebrow}>Plan A Load</Text>
          <Text style={styles.title}>Turn your wardrobe into practical wash-day batches.</Text>
          <Text style={styles.body}>
            Choose how you want to organize the load summaries, then review the grouped batches
            FreshCycle generates from your saved garment care data.
          </Text>
        </View>

        {!authReady ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Setup required</Text>
            <Text style={styles.meta}>
              Add `EXPO_PUBLIC_SUPABASE_URL` and `EXPO_PUBLIC_SUPABASE_KEY` to enable auth.
            </Text>
            <Text style={styles.meta}>API base URL: {env.apiBaseUrl ?? 'not configured'}</Text>
          </View>
        ) : null}

        {authReady && loading ? (
          <View style={styles.card}>
            <ActivityIndicator color={palette.accent} />
            <Text style={styles.meta}>Checking your session...</Text>
          </View>
        ) : null}

        {authReady && !loading && !session ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Sign in to plan a load</Text>
            <Text style={styles.meta}>
              FreshCycle needs your private wardrobe data before it can suggest grouped loads.
            </Text>
            <Link href={'/auth' as never} style={styles.link}>
              Open auth screens
            </Link>
          </View>
        ) : null}

        {canLoadWardrobe && isFetching && garments.length === 0 ? (
          <View style={styles.card}>
            <ActivityIndicator color={palette.accent} />
            <Text style={styles.meta}>Loading your wardrobe...</Text>
          </View>
        ) : null}

        {canLoadWardrobe && fetchError ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Load planning unavailable</Text>
            <Text style={styles.meta}>{fetchError}</Text>
            <Pressable
              onPress={() => {
                if (!session?.access_token) {
                  return;
                }

                void loadGarments(session.access_token, 'initial');
              }}
              style={styles.primaryButton}>
              <Text style={styles.primaryButtonText}>Try again</Text>
            </Pressable>
          </View>
        ) : null}

        {canLoadWardrobe && !isFetching && garments.length === 0 && !fetchError ? (
          <View style={[styles.card, styles.emptyStateCard]}>
            <Text style={styles.cardTitle}>No garments to plan yet</Text>
            <Text style={styles.meta}>
              Add garments first so FreshCycle has real care labels to group into loads.
            </Text>
            <Link href={'/add-garment' as never} style={styles.primaryLink}>
              Add your first garment
            </Link>
          </View>
        ) : null}

        {canLoadWardrobe && garments.length > 0 ? (
          <>
            <View style={styles.sectionHeader}>
              <Text style={styles.cardTitle}>Grouping mode</Text>
              <Text style={styles.meta}>
                {garments.length} garment{garments.length === 1 ? '' : 's'}
              </Text>
            </View>

            <View style={styles.groupingBar}>
              {planningModes.map((option) => (
                <Pressable
                  key={option.value}
                  onPress={() => {
                    setPlanningMode(option.value);
                    logLoadPlanningEvent('load_planning_mode_selected', {
                      mode: option.value,
                      garmentCount: garments.length,
                    });
                  }}
                  style={[
                    styles.groupingChip,
                    planningMode === option.value && styles.groupingChipActive,
                  ]}>
                  <Text
                    style={[
                      styles.groupingChipText,
                      planningMode === option.value && styles.groupingChipTextActive,
                    ]}>
                    {option.label}
                  </Text>
                </Pressable>
              ))}
            </View>

            {planningError ? (
              <View style={styles.card}>
                <Text style={styles.cardTitle}>Load summaries unavailable</Text>
                <Text style={styles.meta}>{planningError}</Text>
              </View>
            ) : null}

            {!planningError && loads.length === 0 ? (
              <View style={styles.card}>
                <Text style={styles.cardTitle}>No valid loads yet</Text>
                <Text style={styles.meta}>
                  FreshCycle could not generate load summaries from the current wardrobe data yet.
                  Try adding more garments with clear care labels.
                </Text>
              </View>
            ) : null}

            {!planningError &&
              loads.map((load) => (
                <View key={load.key} style={[styles.card, styles.loadCard]}>
                  <View style={styles.loadHeader}>
                    <View style={styles.loadHeaderCopy}>
                      <Text style={styles.loadTitle}>{load.title}</Text>
                      <Text style={styles.meta}>{load.description}</Text>
                    </View>
                    <View style={styles.loadBadge}>
                      <Text style={styles.loadBadgeText}>{formatLoadType(load.loadType)}</Text>
                    </View>
                  </View>

                  <View style={styles.metricsRow}>
                    <View style={styles.metric}>
                      <Text style={styles.metricLabel}>Garments</Text>
                      <Text style={styles.metricValue}>{load.garments.length}</Text>
                    </View>
                    <View style={styles.metric}>
                      <Text style={styles.metricLabel}>Unknown temp</Text>
                      <Text style={styles.metricValue}>{load.hasUnknownTemperature ? 'Yes' : 'No'}</Text>
                    </View>
                  </View>

                  <Text style={styles.previewLabel}>Included garments</Text>
                  <Text style={styles.previewValue}>
                    {load.garments.map((garment) => garment.name).join(' • ')}
                  </Text>
                </View>
              ))}
          </>
        ) : null}

        <View style={styles.footerLinks}>
          <Link href={'/' as never} style={styles.linkSecondary}>
            Back to wardrobe
          </Link>
          <Link href={'/add-garment' as never} style={styles.link}>
            Add more garments
          </Link>
        </View>
      </ScrollView>
    </AppScreen>
  );
}

function describeWardrobeError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'auth-required':
        return 'Sign in again before planning a load.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to load garments from.';
      default:
        return 'FreshCycle could not load your wardrobe right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not load your wardrobe right now. Try again in a moment.';
}

function formatLoadType(loadType: PlannedLoadSummary['loadType']) {
  switch (loadType) {
    case 'dry_clean':
      return 'Dry clean';
    case 'hand_wash':
      return 'Hand wash';
    case 'mixed':
      return 'Mixed';
    default:
      return 'Machine wash';
  }
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
  sectionHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 12,
    marginTop: 4,
  },
  groupingBar: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 10,
    marginBottom: 18,
  },
  groupingChip: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 14,
    paddingVertical: 10,
  },
  groupingChipActive: {
    backgroundColor: palette.ink,
    borderColor: palette.ink,
  },
  groupingChipText: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '600',
  },
  groupingChipTextActive: {
    color: palette.canvas,
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
  loadCard: {
    backgroundColor: '#f2ead8',
  },
  cardTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  loadHeader: {
    alignItems: 'flex-start',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  loadHeaderCopy: {
    flex: 1,
    gap: 6,
  },
  loadTitle: {
    color: palette.ink,
    fontSize: 24,
    fontWeight: '700',
  },
  loadBadge: {
    backgroundColor: palette.accentSoft,
    borderRadius: 999,
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  loadBadgeText: {
    color: palette.ink,
    fontSize: 12,
    fontWeight: '700',
    textTransform: 'uppercase',
  },
  metricsRow: {
    flexDirection: 'row',
    gap: 12,
  },
  metric: {
    backgroundColor: '#f7f2e8',
    borderRadius: 16,
    flex: 1,
    gap: 4,
    padding: 14,
  },
  metricLabel: {
    color: palette.inkMuted,
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.5,
    textTransform: 'uppercase',
  },
  metricValue: {
    color: palette.ink,
    fontSize: 20,
    fontWeight: '700',
  },
  previewLabel: {
    color: palette.inkMuted,
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 0.5,
    textTransform: 'uppercase',
  },
  previewValue: {
    color: palette.ink,
    fontSize: 15,
    lineHeight: 22,
  },
  meta: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
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
  primaryLink: {
    color: palette.ink,
    fontSize: 16,
    fontWeight: '700',
    marginTop: 6,
  },
  primaryButton: {
    alignItems: 'center',
    alignSelf: 'flex-start',
    backgroundColor: palette.ink,
    borderRadius: 999,
    marginTop: 8,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  primaryButtonText: {
    color: palette.canvas,
    fontSize: 14,
    fontWeight: '700',
  },
  emptyStateCard: {
    backgroundColor: palette.cardStrong,
  },
  footerLinks: {
    gap: 8,
    marginTop: 8,
  },
});
