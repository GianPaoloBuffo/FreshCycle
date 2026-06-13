import { ScanLabelClientResult } from '@/features/scan-label/types';

export type CareLabelScannerProps = {
  accessToken: string | null;
  disabled?: boolean;
  onCancel: () => void;
  onComplete: (result: ScanLabelClientResult) => void;
  onError: (error: unknown) => void;
};
