import { useEffect } from 'react';
import { Link } from 'expo-router';
import { StyleSheet, Text, View } from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { logSchedulesEvent } from '@/features/schedules/observability';

export function NewScheduleScreen() {
  useEffect(() => {
    logSchedulesEvent('new_schedule_opened');
  }, []);

  return (
    <AppScreen>
      <View style={styles.hero}>
        <Text style={styles.eyebrow}>New Schedule</Text>
        <Text style={styles.title}>The full schedule builder lands next.</Text>
        <Text style={styles.body}>
          This entry route is live now so the schedules screen has a real navigation target. The
          garment picker, recurrence form, and reminder toggle are the next slice in `GP-36`.
        </Text>
      </View>

      <View style={styles.card}>
        <Text style={styles.cardTitle}>What’s coming next</Text>
        <Text style={styles.meta}>Name a schedule and choose the garments it should track.</Text>
        <Text style={styles.meta}>Pick a simple recurrence such as daily or weekly.</Text>
        <Text style={styles.meta}>Turn reminders on or off before saving.</Text>
      </View>

      <View style={styles.footerLinks}>
        <Link href={'/schedules' as never} style={styles.link}>
          Back to schedules
        </Link>
        <Link href={'/' as never} style={styles.linkSecondary}>
          Back to wardrobe
        </Link>
      </View>
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
