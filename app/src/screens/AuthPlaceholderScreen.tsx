import { StyleSheet, Text, View } from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';

export function AuthPlaceholderScreen() {
  return (
    <AppScreen>
      <View style={styles.card}>
        <Text style={styles.title}>Auth foundation placeholder</Text>
        <Text style={styles.body}>
          This route is reserved for the Phase 1 email/password flow. The next slice can attach
          Supabase client creation, session state, and authenticated navigation without reshaping
          the app shell.
        </Text>
      </View>

      <View style={styles.checklist}>
        <Text style={styles.checkTitle}>Planned follow-up seams</Text>
        <Text style={styles.checkItem}>Supabase client bootstrap in `src/lib`</Text>
        <Text style={styles.checkItem}>Session-aware route guards around app entrypoints</Text>
        <Text style={styles.checkItem}>Shared auth form components in `src/components`</Text>
      </View>
    </AppScreen>
  );
}

const styles = StyleSheet.create({
  card: {
    backgroundColor: palette.cardStrong,
    borderRadius: 24,
    gap: 12,
    marginBottom: 18,
    padding: 22,
  },
  title: {
    color: palette.ink,
    fontSize: 24,
    fontWeight: '700',
  },
  body: {
    color: palette.ink,
    fontSize: 16,
    lineHeight: 24,
  },
  checklist: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 20,
    borderWidth: 1,
    gap: 10,
    padding: 20,
  },
  checkTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  checkItem: {
    color: palette.inkMuted,
    fontSize: 15,
    lineHeight: 22,
  },
});
