import { Link } from 'expo-router';
import { Pressable, StyleSheet, Text, View } from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

const env = getAppEnv();

export function HomeScreen() {
  const { authReady, loading, session, signOut } = useAuth();
  const userEmail = session?.user.email ?? null;

  return (
    <AppScreen>
      <View style={styles.hero}>
        <Text style={styles.eyebrow}>Phase 1 foundation</Text>
        <Text style={styles.title}>FreshCycle starts with a focused mobile shell.</Text>
        <Text style={styles.body}>
          This scaffold sets up Expo Router, TypeScript, shared styling primitives, and config
          seams for the API and Supabase work that follows in GP-8.
        </Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>What is ready now</Text>
        <Text style={styles.listItem}>Expo Router entrypoints for the app shell</Text>
        <Text style={styles.listItem}>A shared theme file for future screens and components</Text>
        <Text style={styles.listItem}>Environment accessors for API and Supabase wiring</Text>
        <Text style={styles.listItem}>Supabase auth client bootstrap with persisted sessions</Text>
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

      <Link href="/auth" style={styles.link}>
        Open auth screens
      </Link>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
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
