import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { useAuth } from '@/hooks/useAuth';

type AuthMode = 'sign-in' | 'sign-up';

export function AuthScreen() {
  const { authReady, loading, session, signIn, signUp, signOut } = useAuth();
  const [mode, setMode] = useState<AuthMode>('sign-in');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [feedback, setFeedback] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const canSubmit = useMemo(() => {
    return authReady && email.trim().length > 0 && password.length >= 6 && !submitting;
  }, [authReady, email, password, submitting]);

  async function handleSubmit() {
    if (!canSubmit) {
      return;
    }

    setSubmitting(true);
    setFeedback(null);

    const action = mode === 'sign-in' ? signIn : signUp;
    const { error, needsEmailConfirmation } = await action({
      email: email.trim(),
      password,
    });

    if (error) {
      setFeedback(error.message);
      setSubmitting(false);
      return;
    }

    if (mode === 'sign-up' && needsEmailConfirmation) {
      setFeedback('Account created. Check your email to confirm your address before signing in.');
    } else {
      setFeedback(mode === 'sign-in' ? 'Signed in successfully.' : 'Account created successfully.');
    }

    setSubmitting(false);
  }

  return (
    <AppScreen>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={styles.container}>
        <ScrollView
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled">
          <View style={styles.hero}>
            <Text style={styles.eyebrow}>Supabase auth</Text>
            <Text style={styles.title}>Basic email and password auth is ready for Phase 1.</Text>
            <Text style={styles.body}>
              This screen provides the first user-facing auth flow while keeping the rest of the app
              shell stable for later protected routes and API integration.
            </Text>
          </View>

          <View style={styles.modeRow}>
            <Pressable
              onPress={() => setMode('sign-in')}
              style={[styles.modeButton, mode === 'sign-in' && styles.modeButtonActive]}>
              <Text style={[styles.modeText, mode === 'sign-in' && styles.modeTextActive]}>Sign in</Text>
            </Pressable>
            <Pressable
              onPress={() => setMode('sign-up')}
              style={[styles.modeButton, mode === 'sign-up' && styles.modeButtonActive]}>
              <Text style={[styles.modeText, mode === 'sign-up' && styles.modeTextActive]}>Sign up</Text>
            </Pressable>
          </View>

          <View style={styles.card}>
            <Text style={styles.label}>Email</Text>
            <TextInput
              autoCapitalize="none"
              autoComplete="email"
              keyboardType="email-address"
              onChangeText={setEmail}
              placeholder="you@example.com"
              placeholderTextColor={palette.inkMuted}
              style={styles.input}
              value={email}
            />

            <Text style={styles.label}>Password</Text>
            <TextInput
              autoCapitalize="none"
              onChangeText={setPassword}
              placeholder="At least 6 characters"
              placeholderTextColor={palette.inkMuted}
              secureTextEntry
              style={styles.input}
              value={password}
            />

            {!authReady && (
              <Text style={styles.helpText}>
                Add `EXPO_PUBLIC_SUPABASE_URL` and `EXPO_PUBLIC_SUPABASE_KEY` to enable auth.
              </Text>
            )}

            {feedback && <Text style={styles.feedback}>{feedback}</Text>}

            <Pressable
              disabled={!canSubmit}
              onPress={() => void handleSubmit()}
              style={[styles.primaryButton, !canSubmit && styles.primaryButtonDisabled]}>
              {submitting ? (
                <ActivityIndicator color="#f7f2e8" />
              ) : (
                <Text style={styles.primaryButtonText}>
                  {mode === 'sign-in' ? 'Sign in' : 'Create account'}
                </Text>
              )}
            </Pressable>
          </View>

          <View style={styles.card}>
            <Text style={styles.cardTitle}>Session state</Text>
            {loading && <Text style={styles.helpText}>Loading auth session...</Text>}
            {!loading && !session && (
              <Text style={styles.helpText}>No session is active yet on this device.</Text>
            )}
            {!loading && session && (
              <>
                <Text style={styles.helpText}>
                  Signed in as {session.user.email ?? 'an authenticated user'}.
                </Text>
                <Pressable onPress={() => void signOut()} style={styles.secondaryButton}>
                  <Text style={styles.secondaryButtonText}>Sign out</Text>
                </Pressable>
              </>
            )}
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
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
    fontSize: 30,
    fontWeight: '700',
    lineHeight: 36,
  },
  body: {
    color: palette.inkMuted,
    fontSize: 16,
    lineHeight: 24,
  },
  modeRow: {
    flexDirection: 'row',
    gap: 10,
    marginBottom: 16,
  },
  modeButton: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  modeButtonActive: {
    backgroundColor: palette.cardStrong,
    borderColor: palette.accent,
  },
  modeText: {
    color: palette.inkMuted,
    fontSize: 14,
    fontWeight: '600',
  },
  modeTextActive: {
    color: palette.ink,
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
  label: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '600',
  },
  input: {
    backgroundColor: '#fffaf1',
    borderColor: palette.border,
    borderRadius: 14,
    borderWidth: 1,
    color: palette.ink,
    fontSize: 16,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  helpText: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  feedback: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '500',
    lineHeight: 20,
  },
  primaryButton: {
    alignItems: 'center',
    backgroundColor: palette.accent,
    borderRadius: 16,
    justifyContent: 'center',
    minHeight: 48,
    paddingHorizontal: 16,
    paddingVertical: 12,
  },
  primaryButtonDisabled: {
    opacity: 0.45,
  },
  primaryButtonText: {
    color: '#f7f2e8',
    fontSize: 15,
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
});
