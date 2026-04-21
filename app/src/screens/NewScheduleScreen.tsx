import { Link, useRouter } from 'expo-router';
import { useEffect, useRef, useState } from 'react';
import { Controller, useForm, useWatch } from 'react-hook-form';
import {
  ActivityIndicator,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
} from 'react-native';

import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import {
  buildCreateSchedulePayload,
  createInitialScheduleFormValues,
  formatRecurrenceLabel,
  recurrenceOptions,
  validateScheduleDraft,
} from '@/features/schedules/form';
import { saveSchedule } from '@/features/schedules/saveSchedule';
import { logSchedulesError, logSchedulesEvent } from '@/features/schedules/observability';
import { ScheduleFormValues } from '@/features/schedules/types';
import { fetchGarments } from '@/features/wardrobe/fetchGarments';
import { WardrobeGarment } from '@/features/wardrobe/types';
import {
  registerForScheduleNotifications,
  scheduleLocalNotificationsForSchedule,
} from '@/features/notifications/localNotifications';
import { useAuth } from '@/hooks/useAuth';
import { getAppEnv } from '@/lib/env';

export function NewScheduleScreen() {
  const router = useRouter();
  const { authReady, loading, session } = useAuth();
  const env = getAppEnv();
  const [garments, setGarments] = useState<WardrobeGarment[]>([]);
  const [isFetchingGarments, setIsFetchingGarments] = useState(false);
  const [isRefreshingGarments, setIsRefreshingGarments] = useState(false);
  const [garmentsError, setGarmentsError] = useState<string | null>(null);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);
  const lastLoadedAccessTokenRef = useRef<string | null>(null);
  const {
    control,
    formState: { errors, isSubmitting },
    getValues,
    handleSubmit,
    setError,
    setValue,
  } = useForm<ScheduleFormValues>({
    defaultValues: createInitialScheduleFormValues(),
  });
  const selectedGarmentIds = useWatch({
    control,
    name: 'garmentIds',
  });
  const selectedRecurrence = useWatch({
    control,
    name: 'recurrence',
  });

  useEffect(() => {
    logSchedulesEvent('new_schedule_opened');
  }, []);

  useEffect(() => {
    if (!authReady || loading || !session) {
      setGarments([]);
      setGarmentsError(null);
      setIsFetchingGarments(false);
      setIsRefreshingGarments(false);
      lastLoadedAccessTokenRef.current = null;
      return;
    }

    const accessToken = session.access_token ?? null;
    if (!accessToken || lastLoadedAccessTokenRef.current === accessToken) {
      return;
    }

    lastLoadedAccessTokenRef.current = accessToken;
    void loadGarments(accessToken, 'initial');
  }, [authReady, loading, session]);

  async function loadGarments(accessToken: string, reason: 'initial' | 'refresh') {
    if (reason === 'refresh') {
      setIsRefreshingGarments(true);
    } else {
      setIsFetchingGarments(true);
      setGarmentsError(null);
    }

    logSchedulesEvent('new_schedule_garments_fetch_started', {
      reason,
      hasExistingGarments: garments.length > 0,
    });

    try {
      const nextGarments = await fetchGarments({
        accessToken,
      });

      setGarments(nextGarments);
      setGarmentsError(null);
      logSchedulesEvent('new_schedule_garments_fetch_succeeded', {
        reason,
        garmentCount: nextGarments.length,
      });
    } catch (error) {
      setGarmentsError(
        reason === 'refresh'
          ? 'FreshCycle could not refresh your garments right now. Try again in a moment.'
          : describeGarmentsError(error)
      );
      logSchedulesError('new_schedule_garments_fetch_failed', error, {
        reason,
      });
    } finally {
      setIsFetchingGarments(false);
      setIsRefreshingGarments(false);
    }
  }

  function toggleGarmentSelection(garmentId: string) {
    const currentSelection = getValues('garmentIds');
    const nextSelection = currentSelection.includes(garmentId)
      ? currentSelection.filter((currentId) => currentId !== garmentId)
      : [...currentSelection, garmentId];

    setValue('garmentIds', nextSelection, {
      shouldDirty: true,
      shouldTouch: true,
      shouldValidate: true,
    });

    if (nextSelection.length > 0) {
      setSubmitError(null);
    }
  }

  const submitForm = handleSubmit(async (values) => {
    setSubmitError(null);
    setSuccessMessage(null);

    const validationErrors = validateScheduleDraft(values);
    if (Object.keys(validationErrors).length > 0) {
      if (validationErrors.name) {
        setError('name', {
          message: validationErrors.name,
          type: 'manual',
        });
      }

      if (validationErrors.recurrence) {
        setError('recurrence', {
          message: validationErrors.recurrence,
          type: 'manual',
        });
      }

      if (validationErrors.garmentIds) {
        setError('garmentIds', {
          message: validationErrors.garmentIds,
          type: 'manual',
        });
      }

      logSchedulesEvent('new_schedule_validation_failed', {
        hasNameError: Boolean(validationErrors.name),
        hasRecurrenceError: Boolean(validationErrors.recurrence),
        hasGarmentError: Boolean(validationErrors.garmentIds),
      });
      return;
    }

    try {
      logSchedulesEvent('new_schedule_submission_started', {
        garmentCount: values.garmentIds.length,
        remindersEnabled: values.remindersEnabled,
        recurrence: values.recurrence,
      });

      const savedSchedule = await saveSchedule(buildCreateSchedulePayload(values), {
        accessToken: session?.access_token ?? null,
      });

      let reminderMessage = savedSchedule.reminders_enabled
        ? 'Reminder setup is pending.'
        : 'Reminders are disabled for this schedule.';

      if (savedSchedule.reminders_enabled) {
        try {
          await registerForScheduleNotifications(session?.user.id, {});
          const notificationIds = await scheduleLocalNotificationsForSchedule(savedSchedule);
          reminderMessage =
            notificationIds.length > 0
              ? `${notificationIds.length} local reminder${notificationIds.length === 1 ? '' : 's'} scheduled.`
              : 'No local reminders were scheduled yet.';
        } catch (error) {
          reminderMessage = describeNotificationSetupError(error);
          logSchedulesError('new_schedule_submission_failed', error, {
            reason: 'notification-setup-failed',
            scheduleId: savedSchedule.id,
          });
        }
      }

      setSuccessMessage(
        `Saved "${savedSchedule.name}" with ${formatRecurrenceLabel(savedSchedule.recurrence)} cadence. ${reminderMessage}`
      );
      logSchedulesEvent('new_schedule_submission_succeeded', {
        scheduleId: savedSchedule.id,
        garmentCount: savedSchedule.garment_ids.length,
        remindersEnabled: savedSchedule.reminders_enabled,
      });
    } catch (error) {
      const nextError = describeSaveError(error);
      setSubmitError(nextError);
      logSchedulesError('new_schedule_submission_failed', error, {
        garmentCount: values.garmentIds.length,
        recurrence: values.recurrence,
      });
    }
  });

  return (
    <AppScreen>
      <ScrollView
        contentContainerStyle={styles.scrollContent}
        refreshControl={
          session ? (
            <RefreshControl
              refreshing={isRefreshingGarments}
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
          <Text style={styles.eyebrow}>New Schedule</Text>
          <Text style={styles.title}>Turn a repeat laundry job into a routine you can trust.</Text>
          <Text style={styles.body}>
            Give the schedule a name, choose the garments it should track, pick a simple cadence,
            and decide whether reminders should be enabled before saving.
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
            <Text style={styles.cardTitle}>Sign in to create schedules</Text>
            <Text style={styles.meta}>
              FreshCycle stores schedules inside your private account, so start by authenticating.
            </Text>
            <Link href={'/auth' as never} style={styles.link}>
              Open auth screens
            </Link>
          </View>
        ) : null}

        {session && isFetchingGarments && garments.length === 0 ? (
          <View style={styles.card}>
            <ActivityIndicator color={palette.accent} />
            <Text style={styles.meta}>Loading your garments...</Text>
          </View>
        ) : null}

        {session && garmentsError ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Garments unavailable</Text>
            <Text style={styles.meta}>{garmentsError}</Text>
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

        {session && !isFetchingGarments && garments.length === 0 && !garmentsError ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Add garments before creating a schedule</Text>
            <Text style={styles.meta}>
              Schedules need at least one saved garment, so capture a care label first and then come
              back to build the recurring plan.
            </Text>
            <Link href={'/add-garment' as never} style={styles.primaryLink}>
              Add your first garment
            </Link>
          </View>
        ) : null}

        {session && garments.length > 0 ? (
          <>
            <View style={styles.card}>
              <Text style={styles.cardTitle}>Schedule details</Text>

              <Controller
                control={control}
                name="name"
                render={({ field: { onBlur, onChange, value } }) => (
                  <View style={styles.formField}>
                    <Text style={styles.fieldLabel}>Schedule name</Text>
                    <TextInput
                      autoCapitalize="sentences"
                      onBlur={onBlur}
                      onChangeText={onChange}
                      placeholder="Weekly towels"
                      placeholderTextColor={palette.inkMuted}
                      style={[styles.input, errors.name && styles.inputError]}
                      value={value}
                    />
                    {errors.name?.message ? (
                      <Text style={styles.fieldError}>{errors.name.message}</Text>
                    ) : null}
                  </View>
                )}
              />

              <View style={styles.formField}>
                <Text style={styles.fieldLabel}>Recurrence</Text>
                <View style={styles.optionStack}>
                  {recurrenceOptions.map((option) => {
                    const isSelected = selectedRecurrence === option.value;

                    return (
                      <Pressable
                        key={option.value}
                        onPress={() =>
                          setValue('recurrence', option.value, {
                            shouldDirty: true,
                            shouldTouch: true,
                            shouldValidate: true,
                          })
                        }
                        style={[
                          styles.optionCard,
                          isSelected && styles.optionCardSelected,
                          errors.recurrence?.message && styles.optionCardError,
                        ]}>
                        <Text style={styles.optionTitle}>{option.label}</Text>
                        <Text style={styles.optionDescription}>{option.description}</Text>
                      </Pressable>
                    );
                  })}
                </View>
                {errors.recurrence?.message ? (
                  <Text style={styles.fieldError}>{errors.recurrence.message}</Text>
                ) : null}
              </View>

              <Controller
                control={control}
                name="remindersEnabled"
                render={({ field: { onChange, value } }) => (
                  <View style={styles.toggleRow}>
                    <View style={styles.toggleCopy}>
                      <Text style={styles.fieldLabel}>Reminders</Text>
                      <Text style={styles.meta}>
                        {value
                          ? 'FreshCycle should schedule reminders once notifications are wired in.'
                          : 'Save the schedule without reminder prompts for now.'}
                      </Text>
                    </View>
                    <Switch
                      onValueChange={onChange}
                      thumbColor={value ? palette.ink : '#f5f1e8'}
                      trackColor={{
                        false: '#d8ccb7',
                        true: palette.accentSoft,
                      }}
                      value={value}
                    />
                  </View>
                )}
              />
            </View>

            <View style={styles.card}>
              <View style={styles.sectionHeader}>
                <Text style={styles.cardTitle}>Tracked garments</Text>
                <Text style={styles.meta}>
                  {selectedGarmentIds.length} selected
                </Text>
              </View>
              <Text style={styles.meta}>
                Choose the garments this schedule should keep in view.
              </Text>

              <View style={styles.optionStack}>
                {garments.map((garment) => {
                  const isSelected = selectedGarmentIds.includes(garment.id);

                  return (
                    <Pressable
                      key={garment.id}
                      onPress={() => toggleGarmentSelection(garment.id)}
                      style={[
                        styles.garmentCard,
                        isSelected && styles.garmentCardSelected,
                        errors.garmentIds?.message && styles.optionCardError,
                      ]}>
                      <View style={styles.garmentCardHeader}>
                        <Text style={styles.garmentName}>{garment.name}</Text>
                        <Text style={styles.badge}>{isSelected ? 'Selected' : 'Tap to add'}</Text>
                      </View>
                      <Text style={styles.meta}>
                        {garment.category ?? 'Uncategorized'} •{' '}
                        {garment.wash_temperature_c !== null
                          ? `${garment.wash_temperature_c}°C`
                          : 'Check label'}
                      </Text>
                    </Pressable>
                  );
                })}
              </View>

              {errors.garmentIds?.message ? (
                <Text style={styles.fieldError}>{errors.garmentIds.message}</Text>
              ) : null}
            </View>

            {submitError ? (
              <View style={styles.card}>
                <Text style={styles.cardTitle}>Schedule not saved</Text>
                <Text style={styles.meta}>{submitError}</Text>
              </View>
            ) : null}

            {successMessage ? (
              <View style={[styles.card, styles.successCard]}>
                <Text style={styles.cardTitle}>Schedule saved</Text>
                <Text style={styles.meta}>{successMessage}</Text>
                <Pressable onPress={() => router.replace('/schedules' as never)} style={styles.primaryButton}>
                  <Text style={styles.primaryButtonText}>Back to schedules</Text>
                </Pressable>
              </View>
            ) : null}

            <Pressable disabled={isSubmitting} onPress={() => void submitForm()} style={styles.primaryButton}>
              <Text style={styles.primaryButtonText}>
                {isSubmitting ? 'Saving schedule...' : 'Save schedule'}
              </Text>
            </Pressable>
          </>
        ) : null}

        <View style={styles.footerLinks}>
          <Link href={'/schedules' as never} style={styles.link}>
            Back to schedules
          </Link>
          <Link href={'/' as never} style={styles.linkSecondary}>
            Back to wardrobe
          </Link>
        </View>
      </ScrollView>
    </AppScreen>
  );
}

function describeGarmentsError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'auth-required':
        return 'Sign in again before building a schedule.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to load garments from.';
      default:
        return 'FreshCycle could not load your garments right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not load your garments right now. Try again in a moment.';
}

function describeSaveError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'auth-required':
        return 'Sign in again before saving the schedule.';
      case 'api-unavailable':
        return 'Set `EXPO_PUBLIC_API_BASE_URL` so FreshCycle knows where to save schedules.';
      case 'name-required':
        return 'Give the schedule a name before saving.';
      case 'garments-required':
        return 'Select at least one garment before saving.';
      case 'invalid-garment-id':
        return 'Choose only garments from your own wardrobe before saving.';
      case 'recurrence-invalid':
        return 'Choose one of the supported recurrence options before saving.';
      case 'not-ready':
        return 'The schedule builder is ready, but the schedules API arrives later in Phase 4. Save will start working once GP-39 lands.';
      default:
        return 'FreshCycle could not save the schedule right now. Try again in a moment.';
    }
  }

  return 'FreshCycle could not save the schedule right now. Try again in a moment.';
}

function describeNotificationSetupError(error: unknown) {
  if (error instanceof Error) {
    switch (error.message) {
      case 'notifications-permission-denied':
        return 'Schedule saved, but notification permission was denied. You can enable reminders later from device settings.';
      case 'auth-required':
        return 'Schedule saved, but FreshCycle could not link reminders to your account session.';
      case 'push-token-save-failed':
        return 'Schedule saved, but FreshCycle could not store this device token for reminders.';
      case 'local-notification-schedule-failed':
        return 'Schedule saved, but local reminders could not be scheduled on this device.';
      default:
        return 'Schedule saved, but reminder setup could not finish on this device.';
    }
  }

  return 'Schedule saved, but reminder setup could not finish on this device.';
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
  formField: {
    gap: 8,
  },
  fieldLabel: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '700',
  },
  input: {
    backgroundColor: '#f7f2e8',
    borderColor: palette.border,
    borderRadius: 16,
    borderWidth: 1,
    color: palette.ink,
    fontSize: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  inputError: {
    borderColor: '#b86b5f',
  },
  fieldError: {
    color: '#9b4334',
    fontSize: 14,
    lineHeight: 20,
  },
  optionStack: {
    gap: 10,
  },
  optionCard: {
    backgroundColor: '#f7f2e8',
    borderColor: palette.border,
    borderRadius: 18,
    borderWidth: 1,
    gap: 6,
    padding: 16,
  },
  optionCardSelected: {
    backgroundColor: '#e3ecd9',
    borderColor: palette.accent,
  },
  optionCardError: {
    borderColor: '#b86b5f',
  },
  optionTitle: {
    color: palette.ink,
    fontSize: 16,
    fontWeight: '700',
  },
  optionDescription: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  toggleRow: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 16,
    justifyContent: 'space-between',
  },
  toggleCopy: {
    flex: 1,
    gap: 4,
  },
  garmentCard: {
    backgroundColor: '#f7f2e8',
    borderColor: palette.border,
    borderRadius: 18,
    borderWidth: 1,
    gap: 8,
    padding: 16,
  },
  garmentCardSelected: {
    backgroundColor: '#e3ecd9',
    borderColor: palette.accent,
  },
  garmentCardHeader: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  garmentName: {
    color: palette.ink,
    flex: 1,
    fontSize: 18,
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
  successCard: {
    backgroundColor: '#e3ecd9',
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
    opacity: 1,
    paddingHorizontal: 20,
    paddingVertical: 14,
  },
  primaryButtonText: {
    color: palette.canvas,
    fontSize: 16,
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
