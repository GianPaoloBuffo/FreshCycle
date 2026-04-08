import { Link } from 'expo-router';
import { Pressable, ScrollView, StyleSheet, Text, useWindowDimensions, View } from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

const env = getAppEnv();

export function HomeScreen() {
  const { authReady, loading, session, signOut } = useAuth();
  const { width } = useWindowDimensions();
  const userEmail = session?.user.email ?? null;
  const isCompact = width < 640;

  return (
    <AppScreen>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        <View style={styles.hero}>
          <Text style={styles.eyebrow}>Phase 2 in progress</Text>
          <Text style={styles.title}>FreshCycle is ready to start the garment label capture flow.</Text>
          <Text style={styles.body}>
            The app now supports browser access too, so we can test auth and the current
            add-garment flow from desktop and mobile web before the parser and save flow land.
          </Text>
        </View>

        <View style={[styles.card, styles.featureCard]}>
          <Text style={styles.cardTitle}>What is ready now</Text>
          <Text style={styles.listItem}>Camera and photo library entry points for care-label capture</Text>
          <Text style={styles.listItem}>A processing state and preview surface for parsing feedback</Text>
          <Text style={styles.listItem}>Instrumentation hooks for selection and parsing events</Text>
          <Text style={styles.listItem}>Supabase-backed auth state to gate the garment flow</Text>
          <Text style={styles.listItem}>Responsive browser access for laptop and phone testing</Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardTitle}>Config status</Text>
          <Text style={styles.meta}>API base URL: {env.apiBaseUrl ?? 'not configured'}</Text>
          <Text style={styles.meta}>Supabase URL: {env.supabaseUrl ?? 'not configured'}</Text>
          <Text style={styles.meta}>
            Supabase key: {env.supabaseKey ? 'configured' : 'not configured'}
          </Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.cardTitle}>Auth status</Text>
          {!authReady && (
            <Text style={styles.meta}>
              Add `EXPO_PUBLIC_SUPABASE_URL` and `EXPO_PUBLIC_SUPABASE_KEY` to enable auth.
            </Text>
          )}
          {authReady && loading && <Text style={styles.meta}>Checking your session...</Text>}
          {authReady && !loading && !session && (
            <Text style={styles.meta}>No active session yet. Use the auth screen to sign in.</Text>
          )}
          {authReady && !loading && session && (
            <>
              <Text style={styles.meta}>Signed in as {userEmail ?? 'an authenticated user'}.</Text>
              <Pressable onPress={() => void signOut()} style={styles.secondaryButton}>
                <Text style={styles.secondaryButtonText}>Sign out</Text>
              </Pressable>
            </>
          )}
        </View>

        <Link href={'/add-garment' as never} style={[styles.link, isCompact && styles.linkCompact]}>
          Open add garment flow
        </Link>
        <Link href="/auth" style={[styles.linkSecondary, isCompact && styles.linkCompact]}>
          Open auth screens
        </Link>
      </ScrollView>
    </AppScreen>
  );
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
  featureCard: {
    width: '100%',
    maxWidth: 720,
  },
  cardTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
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
});
