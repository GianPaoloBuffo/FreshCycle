import { describe, expect, it, vi } from 'vitest';

vi.mock('expo-notifications', () => ({
  SchedulableTriggerInputTypes: {
    DATE: 'date',
  },
}));

vi.mock('@react-native-async-storage/async-storage', () => ({
  default: {
    getItem: vi.fn(),
    removeItem: vi.fn(),
    setItem: vi.fn(),
  },
}));

vi.mock('@/lib/supabase', () => ({
  supabase: {
    from: vi.fn(),
  },
}));

import {
  cancelLocalNotificationsForSchedule,
  registerForScheduleNotifications,
  scheduleLocalNotificationsForSchedule,
} from './localNotifications';

const schedule = {
  id: 'schedule-123',
  user_id: 'user-123',
  name: 'Weekly towels',
  recurrence: 'daily',
  garment_ids: ['garment-1'],
  reminders_enabled: true,
  created_at: '2026-04-20T09:00:00.000Z',
};

describe('registerForScheduleNotifications', () => {
  it('captures and persists an Expo push token after permission is granted', async () => {
    const savePushTokenImpl = vi.fn().mockResolvedValue(undefined);
    const token = await registerForScheduleNotifications('user-123', {
      notificationsClient: {
        getPermissionsAsync: vi.fn().mockResolvedValue({ status: 'granted' }),
        requestPermissionsAsync: vi.fn(),
        getExpoPushTokenAsync: vi.fn().mockResolvedValue({ data: 'ExponentPushToken[abc]' }),
      } as never,
      savePushTokenImpl,
    });

    expect(token).toBe('ExponentPushToken[abc]');
    expect(savePushTokenImpl).toHaveBeenCalledWith('user-123', 'ExponentPushToken[abc]');
  });

  it('fails without capturing a token when permission is denied', async () => {
    await expect(
      registerForScheduleNotifications('user-123', {
        notificationsClient: {
          getPermissionsAsync: vi.fn().mockResolvedValue({ status: 'undetermined' }),
          requestPermissionsAsync: vi.fn().mockResolvedValue({ status: 'denied' }),
          getExpoPushTokenAsync: vi.fn(),
        } as never,
      })
    ).rejects.toThrow('notifications-permission-denied');
  });
});

describe('scheduleLocalNotificationsForSchedule', () => {
  it('schedules concrete local notifications and stores their identifiers', async () => {
    const scheduleNotificationAsync = vi
      .fn()
      .mockResolvedValueOnce('notification-1')
      .mockResolvedValueOnce('notification-2');
    const setItem = vi.fn().mockResolvedValue(undefined);

    const notificationIds = await scheduleLocalNotificationsForSchedule(schedule, {
      count: 2,
      from: new Date('2026-04-20T08:00:00'),
      notificationsClient: {
        cancelScheduledNotificationAsync: vi.fn(),
        scheduleNotificationAsync,
      } as never,
      storage: {
        getItem: vi.fn().mockResolvedValue(null),
        removeItem: vi.fn().mockResolvedValue(undefined),
        setItem,
      },
    });

    expect(notificationIds).toEqual(['notification-1', 'notification-2']);
    expect(scheduleNotificationAsync).toHaveBeenCalledTimes(2);
    expect(scheduleNotificationAsync).toHaveBeenCalledWith(
      expect.objectContaining({
        content: expect.objectContaining({
          data: {
            freshCycleType: 'laundry_schedule',
            scheduleId: 'schedule-123',
          },
        }),
        trigger: expect.objectContaining({
          type: 'date',
        }),
      })
    );
    expect(setItem).toHaveBeenCalledWith(
      'freshcycle:local-notification-ids:schedule-123',
      JSON.stringify(['notification-1', 'notification-2'])
    );
  });

  it('cancels existing notifications before replacing them', async () => {
    const cancelScheduledNotificationAsync = vi.fn().mockResolvedValue(undefined);

    await scheduleLocalNotificationsForSchedule(schedule, {
      count: 0,
      notificationsClient: {
        cancelScheduledNotificationAsync,
        scheduleNotificationAsync: vi.fn(),
      } as never,
      storage: {
        getItem: vi.fn().mockResolvedValue(JSON.stringify(['old-1', 'old-2'])),
        removeItem: vi.fn().mockResolvedValue(undefined),
        setItem: vi.fn().mockResolvedValue(undefined),
      },
    });

    expect(cancelScheduledNotificationAsync).toHaveBeenCalledWith('old-1');
    expect(cancelScheduledNotificationAsync).toHaveBeenCalledWith('old-2');
  });

  it('only cancels notifications when reminders are disabled', async () => {
    const cancelScheduledNotificationAsync = vi.fn().mockResolvedValue(undefined);
    const scheduleNotificationAsync = vi.fn();

    const notificationIds = await scheduleLocalNotificationsForSchedule(
      {
        ...schedule,
        reminders_enabled: false,
      },
      {
        notificationsClient: {
          cancelScheduledNotificationAsync,
          scheduleNotificationAsync,
        } as never,
        storage: {
          getItem: vi.fn().mockResolvedValue(JSON.stringify(['old-1'])),
          removeItem: vi.fn().mockResolvedValue(undefined),
          setItem: vi.fn().mockResolvedValue(undefined),
        },
      }
    );

    expect(notificationIds).toEqual([]);
    expect(cancelScheduledNotificationAsync).toHaveBeenCalledWith('old-1');
    expect(scheduleNotificationAsync).not.toHaveBeenCalled();
  });
});

describe('cancelLocalNotificationsForSchedule', () => {
  it('cancels stored notifications and clears linkage', async () => {
    const cancelScheduledNotificationAsync = vi.fn().mockResolvedValue(undefined);
    const removeItem = vi.fn().mockResolvedValue(undefined);

    await cancelLocalNotificationsForSchedule('schedule-123', {
      notificationsClient: {
        cancelScheduledNotificationAsync,
      } as never,
      storage: {
        getItem: vi.fn().mockResolvedValue(JSON.stringify(['notification-1'])),
        removeItem,
        setItem: vi.fn(),
      },
    });

    expect(cancelScheduledNotificationAsync).toHaveBeenCalledWith('notification-1');
    expect(removeItem).toHaveBeenCalledWith('freshcycle:local-notification-ids:schedule-123');
  });
});
