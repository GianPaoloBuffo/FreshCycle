import { Link } from 'expo-router';
import { startTransition, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  useWindowDimensions,
  View,
} from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { fetchGarments } from '@/features/wardrobe/fetchGarments';
import { logWardrobeError, logWardrobeEvent } from '@/features/wardrobe/observability';
import { WardrobeGarment } from '@/features/wardrobe/types';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

const env = getAppEnv();

export function HomeScreen() {
  const { authReady, loading, session, signOut } = useAuth();
  const { width } = useWindowDimensions();
  const [garments, setGarments] = useState<WardrobeGarment[]>([]);
  const [isFetching, setIsFetching] = useState(false);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [fetchError, setFetchError] = useState<string | null>(null);
  const userEmail = session?.user.email ?? null;
  const isCompact = width < 640;
  const canLoadWardrobe = authReady && !loading && Boolean(session);
  const lastLoadedAccessTokenRef = useRef<string | null>(null);

  async function loadGarments(accessToken: string, reason: 'initial' | 'refresh') {
    if (reason === 'refresh') {
      setIsRefreshing(true);
      logWardrobeEvent('wardrobe_refresh_started', {
        hasExistingGarments: garments.length > 0,
      });
    } else {
      setIsFetching(true);
      setFetchError(null);
      logWardrobeEvent('wardrobe_fetch_started', {
        hasExistingGarments: garments.length > 0,
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
      });
    } catch (error) {
      const message =
        reason === 'refresh'
          ? 'FreshCycle could not refresh your wardrobe right now. Pull down again in a moment.'
          : describeWardrobeError(error);

      setFetchError(message);
      logWardrobeError(reason === 'refresh' ? 'wardrobe_refresh_failed' : 'wardrobe_fetch_failed', error, {
        reason,
      });
    } finally {
      setIsFetching(false);
      setIsRefreshing(false);
    }
  }

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
          <Text style={styles.eyebrow}>Wardrobe</Text>
          <Text style={styles.title}>Keep your laundry decisions grounded in what you actually own.</Text>
          <Text style={styles.body}>
            Review saved garments, spot missing details quickly, and jump straight into capturing the
            next care label when your wardrobe needs more coverage.
          </Text>
        </View>

        <View style={[styles.card, styles.statusCard]}>
          <View style={styles.statusHeader}>
            <View>
              <Text style={styles.cardTitle}>Account</Text>
              <Text style={styles.meta}>
                {authReady
                  ? userEmail ?? 'Sign in to load your wardrobe.'
                  : 'Configure Supabase auth to unlock wardrobe sync.'}
              </Text>
            </View>
            {authReady && session ? (
              <Pressable onPress={() => void signOut()} style={styles.secondaryButton}>
                <Text style={styles.secondaryButtonText}>Sign out</Text>
              </Pressable>
            ) : null}
          </View>
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
            <Text style={styles.cardTitle}>Sign in to view your wardrobe</Text>
            <Text style={styles.meta}>
              FreshCycle loads garments from your private account, so start by authenticating.
            </Text>
            <Link href="/auth" style={[styles.link, isCompact && styles.linkCompact]}>
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
            <Text style={styles.cardTitle}>Wardrobe unavailable</Text>
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
            <Text style={styles.cardTitle}>Your wardrobe is empty</Text>
            <Text style={styles.meta}>
              Start with your first garment so FreshCycle can build wash-safe loads from real care
              label data.
            </Text>
            <Link href={'/add-garment' as never} style={[styles.primaryLink, isCompact && styles.linkCompact]}>
              Scan your first care label
            </Link>
          </View>
        ) : null}

        {canLoadWardrobe && garments.length > 0 ? (
          <>
            <View style={styles.sectionHeader}>
              <Text style={styles.cardTitle}>Saved garments</Text>
              <Text style={styles.meta}>
                {garments.length} item{garments.length === 1 ? '' : 's'}
              </Text>
            </View>

            {garments.map((garment) => (
              <View key={garment.id} style={styles.card}>
                <View style={styles.cardHeader}>
                  <Text style={styles.garmentName}>{garment.name}</Text>
                  {garment.category ? <Text style={styles.badge}>{garment.category}</Text> : null}
                </View>

                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Primary color</Text>
                  <Text style={styles.detailValue}>{garment.primary_color ?? 'Not set yet'}</Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Wash temperature</Text>
                  <Text style={styles.detailValue}>
                    {garment.wash_temperature_c !== null ? `${garment.wash_temperature_c}°C` : 'Check label'}
                  </Text>
                </View>
                <View style={styles.detailRow}>
                  <Text style={styles.detailLabel}>Care guidance</Text>
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
          <Link href={'/add-garment' as never} style={[styles.link, isCompact && styles.linkCompact]}>
            Add another garment
          </Link>
          <Link href="/auth" style={[styles.linkSecondary, isCompact && styles.linkCompact]}>
            Manage auth
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
        return 'Sign in again before loading your wardrobe.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to load garments from.';
      default:
        return 'FreshCycle could not load your wardrobe right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not load your wardrobe right now. Try again in a moment.';
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
  card: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 20,
    borderWidth: 1,
    gap: 10,
    marginBottom: 16,
    padding: 20,
  },
  featureCard: {
    width: '100%',
    maxWidth: 720,
  },
  statusCard: {
    marginBottom: 20,
  },
  cardTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  cardHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  garmentName: {
    color: palette.ink,
    flex: 1,
    fontSize: 20,
    fontWeight: '700',
  },
  badge: {
    alignSelf: 'flex-start',
    backgroundColor: palette.accentSoft,
    borderRadius: 999,
    color: palette.ink,
    fontSize: 12,
    fontWeight: '700',
    overflow: 'hidden',
    paddingHorizontal: 10,
    paddingVertical: 6,
    textTransform: 'uppercase',
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
  listItem: {
    color: palette.inkMuted,
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
  linkCompact: {
    alignSelf: 'flex-start',
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
  secondaryButton: {
    alignItems: 'center',
    alignSelf: 'flex-start',
    backgroundColor: palette.accentSoft,
    borderRadius: 999,
    marginTop: 8,
    paddingHorizontal: 14,
    paddingVertical: 10,
  },
  secondaryButtonText: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '600',
  },
  statusHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  emptyStateCard: {
    backgroundColor: palette.cardStrong,
  },
  footerLinks: {
    gap: 8,
    marginTop: 8,
  },
});
