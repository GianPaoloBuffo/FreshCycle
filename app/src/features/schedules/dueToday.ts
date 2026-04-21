import { LaundrySchedule } from './types';

const WEEKDAY_INDEX: Record<string, number> = {
  sunday: 0,
  monday: 1,
  tuesday: 2,
  wednesday: 3,
  thursday: 4,
  friday: 5,
  saturday: 6,
};

export function getSchedulesDueToday(schedules: LaundrySchedule[], today: Date = new Date()) {
  return schedules
    .filter((schedule) => isScheduleDueToday(schedule, today))
    .sort((left, right) => left.name.localeCompare(right.name));
}

export function isScheduleDueToday(schedule: LaundrySchedule, today: Date = new Date()) {
  const recurrence = schedule.recurrence.trim().toLowerCase();

  if (recurrence === 'daily') {
    return true;
  }

  if (recurrence.startsWith('weekly:')) {
    const weekday = recurrence.split(':')[1] ?? '';
    return WEEKDAY_INDEX[weekday] === today.getDay();
  }

  if (recurrence === 'fortnightly') {
    const startDate = startOfLocalDay(new Date(schedule.created_at));
    const todayStart = startOfLocalDay(today);
    const elapsedDays = Math.floor((todayStart.getTime() - startDate.getTime()) / 86400000);

    return elapsedDays >= 0 && elapsedDays % 14 === 0;
  }

  return false;
}

function startOfLocalDay(date: Date) {
  const copy = new Date(date);
  copy.setHours(0, 0, 0, 0);
  return copy;
}
