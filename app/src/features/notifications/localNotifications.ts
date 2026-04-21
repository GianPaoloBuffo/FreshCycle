import AsyncStorage from '@react-native-async-storage/async-storage';
import * as Notifications from 'expo-notifications';

import { LaundrySchedule } from '@/features/schedules/types';

import { logNotificationError, logNotificationEvent } from './observability';
import { computeUpcomingReminderOccurrences } from './recurrence';
import { savePushToken } from './savePushToken';

type NotificationsClient = Pick<
  typeof Notifications,
  | 'cancelScheduledNotificationAsync'
  | 'getExpoPushTokenAsync'
  | 'getPermissionsAsync'
  | 'requestPermissionsAsync'
  | 'scheduleNotificationAsync'
  | 'setNotificationHandler'
>;

type NotificationStorage = Pick<typeof AsyncStorage, 'getItem' | 'removeItem' | 'setItem'>;

type RegisterPushTokenDeps = {
  notificationsClient?: NotificationsClient;
  projectId?: string;
  savePushTokenImpl?: typeof savePushToken;
};

type LocalScheduleDeps = {
  count?: number;
  from?: Date;
  notificationsClient?: NotificationsClient;
  storage?: NotificationStorage;
};

const NOTIFICATION_ID_PREFIX = 'freshcycle:local-notification-ids:';

export function configureNotificationHandler(
  notificationsClient: Pick<typeof Notifications, 'setNotificationHandler'> = Notifications
) {
  notificationsClient.setNotificationHandler({
    handleNotification: async () => ({
      shouldPlaySound: false,
      shouldSetBadge: false,
      shouldShowBanner: true,
      shouldShowList: true,
    }),
  });

  logNotificationEvent('notification_handler_configured');
}

export async function registerForScheduleNotifications(
  userId: string | null | undefined,
  deps: RegisterPushTokenDeps = {}
) {
  const notificationsClient = deps.notificationsClient ?? Notifications;
  const savePushTokenImpl = deps.savePushTokenImpl ?? savePushToken;

  if (!userId) {
    throw new Error('auth-required');
  }

  const existingPermissions = await notificationsClient.getPermissionsAsync();
  const permissions =
    existingPermissions.status === 'granted'
      ? existingPermissions
      : await notificationsClient.requestPermissionsAsync({
          ios: {
            allowAlert: true,
            allowBadge: false,
            allowSound: false,
          },
        });

  if (permissions.status !== 'granted') {
    logNotificationEvent('notification_permission_denied', {
      status: permissions.status,
    });
    throw new Error('notifications-permission-denied');
  }

  try {
    logNotificationEvent('push_token_capture_started');
    const pushToken = await notificationsClient.getExpoPushTokenAsync(
      deps.projectId ? { projectId: deps.projectId } : undefined
    );

    await savePushTokenImpl(userId, pushToken.data);
    logNotificationEvent('push_token_capture_succeeded', {
      hasPushToken: Boolean(pushToken.data),
    });

    return pushToken.data;
  } catch (error) {
    logNotificationError('push_token_capture_failed', error);
    throw new Error(error instanceof Error ? error.message : 'push-token-capture-failed');
  }
}

export async function scheduleLocalNotificationsForSchedule(
  schedule: LaundrySchedule,
  deps: LocalScheduleDeps = {}
) {
  const notificationsClient = deps.notificationsClient ?? Notifications;
  const storage = deps.storage ?? AsyncStorage;

  await cancelLocalNotificationsForSchedule(schedule.id, {
    notificationsClient,
    storage,
  });

  if (!schedule.reminders_enabled) {
    return [];
  }

  try {
    logNotificationEvent('local_notifications_schedule_started', {
      scheduleId: schedule.id,
      recurrence: schedule.recurrence,
    });

    const occurrences = computeUpcomingReminderOccurrences(schedule.recurrence, {
      count: deps.count ?? 4,
      from: deps.from,
    });
    const notificationIds = [];

    for (const occurrence of occurrences) {
      const notificationId = await notificationsClient.scheduleNotificationAsync({
        content: {
          title: `${schedule.name} is due`,
          body: 'FreshCycle reminder: check this laundry schedule today.',
          data: {
            freshCycleType: 'laundry_schedule',
            scheduleId: schedule.id,
          },
        },
        trigger: {
          type: Notifications.SchedulableTriggerInputTypes.DATE,
          date: occurrence,
        },
      });

      notificationIds.push(notificationId);
    }

    await storage.setItem(notificationStorageKey(schedule.id), JSON.stringify(notificationIds));
    logNotificationEvent('local_notifications_schedule_succeeded', {
      scheduleId: schedule.id,
      notificationCount: notificationIds.length,
    });

    return notificationIds;
  } catch (error) {
    logNotificationError('local_notifications_schedule_failed', error, {
      scheduleId: schedule.id,
      recurrence: schedule.recurrence,
    });
    throw new Error('local-notification-schedule-failed');
  }
}

export async function cancelLocalNotificationsForSchedule(
  scheduleId: string,
  deps: Pick<LocalScheduleDeps, 'notificationsClient' | 'storage'> = {}
) {
  const notificationsClient = deps.notificationsClient ?? Notifications;
  const storage = deps.storage ?? AsyncStorage;

  try {
    logNotificationEvent('local_notifications_cancel_started', {
      scheduleId,
    });

    const notificationIds = await getStoredNotificationIds(scheduleId, storage);
    await Promise.all(
      notificationIds.map((notificationId) =>
        notificationsClient.cancelScheduledNotificationAsync(notificationId)
      )
    );
    await storage.removeItem(notificationStorageKey(scheduleId));

    logNotificationEvent('local_notifications_cancel_succeeded', {
      scheduleId,
      notificationCount: notificationIds.length,
    });
  } catch (error) {
    logNotificationError('local_notifications_cancel_failed', error, {
      scheduleId,
    });
    throw new Error('local-notification-cancel-failed');
  }
}

async function getStoredNotificationIds(scheduleId: string, storage: NotificationStorage) {
  const rawValue = await storage.getItem(notificationStorageKey(scheduleId));

  if (!rawValue) {
    return [];
  }

  try {
    const parsedValue = JSON.parse(rawValue);

    if (Array.isArray(parsedValue)) {
      return parsedValue.filter((value): value is string => typeof value === 'string');
    }

    return [];
  } catch {
    return [];
  }
}

function notificationStorageKey(scheduleId: string) {
  return `${NOTIFICATION_ID_PREFIX}${scheduleId}`;
}
