import {
  refreshableSettings,
  hiddenColumnsSettings,
  createPersistedStore,
  filteredColumnsSettings,
} from '@@/datatables/types';

import { QuickAction, TableSettings } from './types';

export const TRUNCATE_LENGTH = 32;

const METRIC_COLUMN_IDS = ['cpu', 'memory', 'blockIO'];

/**
 * Older saved table settings predate the metric columns, so rehydration
 * shows them. Rendering those cells used to throw and blank the stack page.
 * Hide them once per table; later toggles in the column menu are kept.
 */
export function hideNewMetricColumnsOnce(
  storageKey: string,
  hiddenColumns: string[],
  setHiddenColumns: (hiddenColumns: string[]) => void
) {
  const flag = `portainer.metricColumnsHiddenOnce.${storageKey}`;
  try {
    if (localStorage.getItem(flag)) {
      return;
    }
    const next = Array.from(new Set([...hiddenColumns, ...METRIC_COLUMN_IDS]));
    if (next.length !== hiddenColumns.length) {
      setHiddenColumns(next);
    }
    localStorage.setItem(flag, '1');
  } catch {
    // Private mode or a blocked storage API should not break the table.
  }
}

export function createStore(storageKey: string) {
  return createPersistedStore<TableSettings>(storageKey, 'name', (set) => ({
    ...hiddenColumnsSettings(set, ['cpu', 'memory', 'blockIO']),
    ...refreshableSettings(set),
    ...filteredColumnsSettings(set),
    truncateContainerName: TRUNCATE_LENGTH,
    setTruncateContainerName(truncateContainerName: number) {
      set({
        truncateContainerName,
      });
    },

    hiddenQuickActions: [] as QuickAction[],
    setHiddenQuickActions: (hiddenQuickActions: QuickAction[]) =>
      set({ hiddenQuickActions }),
  }));
}
