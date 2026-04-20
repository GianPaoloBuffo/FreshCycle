import { Link, useLocalSearchParams } from 'expo-router';
import { startTransition, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { getPlannedLoad } from '@/features/load-planning/planLoads';
import { LoadPlanningMode, PlannedLoadSummary } from '@/features/load-planning/types';
import { logLoadPlanningError, logLoadPlanningEvent } from '@/features/load-planning/observability';
import { fetchGarments } from '@/features/wardrobe/fetchGarments';
import { logWardrobeError, logWardrobeEvent } from '@/features/wardrobe/observability';
import { WardrobeGarment } from '@/features/wardrobe/types';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

const env = getAppEnv();

export function LoadDetailScreen() {
  const { authReady, loading, session } = useAuth();
  const params = useLocalSearchParams<{ loadKey?: string | string[]; mode?: string | string[] }>();
  const [garments, setGarments] = useState<WardrobeGarment[]>([]);
  const [isFetching, setIsFetching] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const [loadDetail, setLoadDetail] = useState<PlannedLoadSummary | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const canLoadWardrobe = authReady && !loading && Boolean(session);
  const lastLoadedAccessTokenRef = useRef<string | null>(null);
  const loadKey = firstParam(params.loadKey);
  const planningMode = parsePlanningMode(firstParam(params.mode));

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
    if (!planningMode || !loadKey) {
      setLoadDetail(null);
      setDetailError('FreshCycle could not determine which load to open.');
      return;
    }

    const nextLoad = getPlannedLoad(garments, planningMode, loadKey);
    if (!nextLoad) {
      setLoadDetail(null);
      setDetailError('That planned load could not be found in the current wardrobe snapshot.');
      logLoadPlanningError('load_detail_failed', new Error('load-not-found'), {
        mode: planningMode,
        loadKey,
        garmentCount: garments.length,
      });
      return;
    }

    startTransition(() => {
      setLoadDetail(nextLoad);
      setDetailError(null);
    });

    logLoadPlanningEvent('load_detail_opened', {
      mode: planningMode,
      loadKey,
      garmentCount: nextLoad.garments.length,
    });
  }, [garments, loadKey, planningMode]);

  async function loadGarments(accessToken: string, reason: 'initial' | 'refresh') {
    if (reason === 'refresh') {
      setIsRefreshing(true);
      logWardrobeEvent('wardrobe_refresh_started', {
        hasExistingGarments: garments.length > 0,
        surface: 'load-detail',
      });
    } else {
      setIsFetching(true);
      setFetchError(null);
      logWardrobeEvent('wardrobe_fetch_started', {
        hasExistingGarments: garments.length > 0,
        surface: 'load-detail',
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
        surface: 'load-detail',
      });
    } catch (error) {
      setFetchError(
        reason === 'refresh'
          ? 'FreshCycle could not refresh your wardrobe right now. Pull down again in a moment.'
          : describeWardrobeError(error)
      );
      logWardrobeError(reason === 'refresh' ? 'wardrobe_refresh_failed' : 'wardrobe_fetch_failed', error, {
        reason,
        surface: 'load-detail',
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
          <Text style={styles.eyebrow}>Load Details</Text>
          <Text style={styles.title}>Inspect exactly what sits inside one planned load.</Text>
          <Text style={styles.body}>
            Review each garment’s care metadata before you commit to a wash batch.
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
            <Text style={styles.cardTitle}>Sign in to inspect planned loads</Text>
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
            <Text style={styles.cardTitle}>Load detail unavailable</Text>
            <Text style={styles.meta}>{fetchError}</Text>
          </View>
        ) : null}

        {canLoadWardrobe && detailError ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Load not available</Text>
            <Text style={styles.meta}>{detailError}</Text>
          </View>
        ) : null}

        {canLoadWardrobe && loadDetail ? (
          <>
            <View style={[styles.card, styles.summaryCard]}>
              <Text style={styles.cardTitle}>{loadDetail.title}</Text>
              <Text style={styles.meta}>{loadDetail.description}</Text>
              <View style={styles.metricsRow}>
                <View style={styles.metric}>
                  <Text style={styles.metricLabel}>Garments</Text>
                  <Text style={styles.metricValue}>{loadDetail.garments.length}</Text>
                </View>
                <View style={styles.metric}>
                  <Text style={styles.metricLabel}>Unknown temp</Text>
                  <Text style={styles.metricValue}>{loadDetail.hasUnknownTemperature ? 'Yes' : 'No'}</Text>
                </View>
              </View>
            </View>

            {loadDetail.garments.map((garment) => (
              <View key={garment.id} style={styles.card}>
                <Text style={styles.garmentName}>{garment.name}</Text>
                <Text style={styles.meta}>{garment.category_label}</Text>

                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Care method</Text>
                  <Text style={styles.detailValue}>{formatCareMethod(garment.care_method)}</Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Temperature ceiling</Text>
                  <Text style={styles.detailValue}>
                    {garment.wash_temperature_c !== null ? `${garment.wash_temperature_c}°C` : 'Unknown'}
                  </Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Colour group</Text>
                  <Text style={styles.detailValue}>{formatColourGroup(garment.colour_group)}</Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Care instructions</Text>
                  <Text style={styles.detailValue}>
                    {garment.care_instructions.length > 0
                      ? garment.care_instructions.join(' • ')
                      : 'No instructions captured yet'}
                  </Text>
                </View>
              </View>
            ))}
          </>
        ) : null}

        <View style={styles.footerLinks}>
          <Link href={'/plan-load' as never} style={styles.linkSecondary}>
            Back to load planner
          </Link>
          <Link href={'/' as never} style={styles.link}>
            Back to wardrobe
          </Link>
        </View>
      </ScrollView>
    </AppScreen>
  );
}

function firstParam(value: string | string[] | undefined) {
  return Array.isArray(value) ? value[0] ?? null : value ?? null;
}

function parsePlanningMode(value: string | null): LoadPlanningMode | null {
  switch (value) {
    case 'smart':
    case 'colour':
    case 'temperature':
    case 'category':
      return value;
    default:
      return null;
  }
}

function describeWardrobeError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'auth-required':
        return 'Sign in again before inspecting that load.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to load garments from.';
      default:
        return 'FreshCycle could not load your wardrobe right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not load your wardrobe right now. Try again in a moment.';
}

function formatCareMethod(careMethod: PlannedLoadSummary['garments'][number]['care_method']) {
  switch (careMethod) {
    case 'dry_clean':
      return 'Dry clean';
    case 'hand_wash':
      return 'Hand wash';
    default:
      return 'Machine wash';
  }
}

function formatColourGroup(colourGroup: PlannedLoadSummary['garments'][number]['colour_group']) {
  switch (colourGroup) {
    case 'white':
      return 'White';
    case 'light':
      return 'Light';
    case 'dark':
      return 'Dark';
    default:
      return 'Colour';
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
  card: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 20,
    borderWidth: 1,
    gap: 10,
    marginBottom: 16,
    padding: 20,
  },
  summaryCard: {
    backgroundColor: '#f2ead8',
  },
  cardTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  garmentName: {
    color: palette.ink,
    fontSize: 22,
    fontWeight: '700',
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
