import * as ImagePicker from 'expo-image-picker';
import { useState } from 'react';
import { ActivityIndicator, Image, Pressable, StyleSheet, Text, View } from 'react-native';

import { CareLabelScannerProps } from '@/components/CareLabelScanner.types';
import { AddGarmentActionError, createSelectedLabelPhoto } from '@/features/add-garment/parseCareLabel';
import { palette } from '@/constants/theme';
import { assessCaptureQuality } from '@/features/scan-label/captureQuality';
import { cropCoverage, DEFAULT_LABEL_CROP } from '@/features/scan-label/cropLabelImage';
import { recognizeCareLabelText } from '@/features/scan-label/recognizeText';
import { scanCareLabelPhoto } from '@/features/scan-label/scanCareLabel';

type ScannerState = 'ready' | 'selecting' | 'processing';

export function CareLabelScanner({ accessToken, disabled, onCancel, onComplete, onError }: CareLabelScannerProps) {
  const [state, setState] = useState<ScannerState>('ready');
  const [previewUri, setPreviewUri] = useState<string | null>(null);

  async function handlePick(source: 'camera' | 'library') {
    try {
      setState('selecting');
      const permission =
        source === 'camera'
          ? await ImagePicker.requestCameraPermissionsAsync()
          : await ImagePicker.requestMediaLibraryPermissionsAsync();

      if (!permission.granted) {
        throw new AddGarmentActionError(
          source === 'camera' ? 'camera-permission-denied' : 'photo-library-permission-denied'
        );
      }

      const picker =
        source === 'camera'
          ? await ImagePicker.launchCameraAsync(getPickerOptions())
          : await ImagePicker.launchImageLibraryAsync(getPickerOptions());
      const asset = picker.assets?.[0];

      if (picker.canceled || !asset) {
        setState('ready');
        return;
      }

      setState('processing');
      const photo = createSelectedLabelPhoto(asset, source);
      setPreviewUri(photo.uri);
      const clientOCR = await recognizeCareLabelText(photo.uri);
      const quality = assessCaptureQuality({
        photo,
        cropCoverage: cropCoverage(DEFAULT_LABEL_CROP),
        stableForMs: 1500,
        ocrText: clientOCR?.text,
        ocrLineCount: clientOCR?.blocks.reduce((total, block) => total + block.lines.length, 0) ?? 0,
      });
      const result = await scanCareLabelPhoto(
        {
          photo,
          clientOCR,
          quality,
        },
        {
          accessToken,
        }
      );

      onComplete(result);
      setState('ready');
    } catch (error) {
      setState('ready');
      onError(error);
    }
  }

  const isBusy = state !== 'ready' || disabled;

  return (
    <View style={styles.shell}>
      <View style={styles.headerRow}>
        <View style={styles.headerCopy}>
          <Text style={styles.eyebrow}>Assisted scanner</Text>
          <Text style={styles.title}>Capture a cropped care label</Text>
        </View>
        <Pressable accessibilityLabel="Close scanner" disabled={isBusy} onPress={onCancel} style={styles.closeButton}>
          <Text style={styles.closeButtonText}>Close</Text>
        </Pressable>
      </View>

      <View style={styles.fallbackPanel}>
        {previewUri ? <Image source={{ uri: previewUri }} style={styles.previewImage} /> : <View style={styles.guideBox} />}
        <Text style={styles.body}>
          Use a clear label-only crop. The app will run on-device OCR when available, then send the cropped image to
          the scan endpoint.
        </Text>
      </View>

      {state === 'processing' && (
        <View style={styles.processingRow}>
          <ActivityIndicator color={palette.ink} />
          <Text style={styles.processingText}>Reading label and preparing confirmation</Text>
        </View>
      )}

      <View style={styles.buttonRow}>
        <Pressable
          accessibilityLabel="Capture care label with device camera"
          disabled={isBusy}
          onPress={() => void handlePick('camera')}
          style={[styles.primaryButton, isBusy && styles.buttonDisabled]}>
          <Text style={styles.primaryButtonText}>Camera capture</Text>
        </Pressable>
        <Pressable
          accessibilityLabel="Choose care label from library"
          disabled={isBusy}
          onPress={() => void handlePick('library')}
          style={[styles.secondaryButton, isBusy && styles.buttonDisabled]}>
          <Text style={styles.secondaryButtonText}>Photo library</Text>
        </Pressable>
      </View>
    </View>
  );
}

function getPickerOptions(): ImagePicker.ImagePickerOptions {
  return {
    allowsEditing: true,
    mediaTypes: ['images'],
    quality: 0.88,
  };
}

const styles = StyleSheet.create({
  shell: {
    backgroundColor: '#ecf1e6',
    borderColor: '#b9c9ae',
    borderRadius: 18,
    borderWidth: 1,
    gap: 16,
    padding: 16,
  },
  headerRow: {
    alignItems: 'flex-start',
    flexDirection: 'row',
    gap: 12,
    justifyContent: 'space-between',
  },
  headerCopy: {
    flex: 1,
    gap: 4,
  },
  eyebrow: {
    color: palette.accent,
    fontSize: 12,
    fontWeight: '700',
    letterSpacing: 1,
    textTransform: 'uppercase',
  },
  title: {
    color: palette.ink,
    fontSize: 20,
    fontWeight: '700',
    lineHeight: 26,
  },
  body: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  fallbackPanel: {
    gap: 12,
  },
  guideBox: {
    aspectRatio: 1.85,
    borderColor: palette.accent,
    borderRadius: 14,
    borderStyle: 'dashed',
    borderWidth: 2,
    backgroundColor: '#f8f4ea',
  },
  previewImage: {
    aspectRatio: 1.85,
    borderRadius: 14,
    width: '100%',
  },
  processingRow: {
    alignItems: 'center',
    flexDirection: 'row',
    gap: 10,
  },
  processingText: {
    color: palette.ink,
    flex: 1,
    fontSize: 14,
    fontWeight: '600',
  },
  buttonRow: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 10,
  },
  primaryButton: {
    alignItems: 'center',
    backgroundColor: palette.ink,
    borderRadius: 999,
    minHeight: 48,
    justifyContent: 'center',
    paddingHorizontal: 18,
  },
  primaryButtonText: {
    color: '#fffaf0',
    fontSize: 15,
    fontWeight: '700',
  },
  secondaryButton: {
    alignItems: 'center',
    borderColor: palette.border,
    borderRadius: 999,
    borderWidth: 1,
    minHeight: 48,
    justifyContent: 'center',
    paddingHorizontal: 18,
  },
  secondaryButtonText: {
    color: palette.ink,
    fontSize: 15,
    fontWeight: '700',
  },
  closeButton: {
    borderColor: palette.border,
    borderRadius: 999,
    borderWidth: 1,
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  closeButtonText: {
    color: palette.ink,
    fontSize: 13,
    fontWeight: '700',
  },
  buttonDisabled: {
    opacity: 0.45,
  },
});
