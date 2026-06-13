import { useEffect, useMemo, useRef, useState } from 'react';
import { ActivityIndicator, Image, Pressable, StyleSheet, Text, View } from 'react-native';
import {
  Camera,
  CameraRef,
  useCameraDevice,
  useCameraPermission,
  usePhotoOutput,
} from 'react-native-vision-camera';

import { CareLabelScannerProps } from '@/components/CareLabelScanner.types';
import { AddGarmentActionError } from '@/features/add-garment/parseCareLabel';
import { SelectedLabelPhoto } from '@/features/add-garment/types';
import { palette } from '@/constants/theme';
import { assessCaptureQuality, buildLiveQualityHints } from '@/features/scan-label/captureQuality';
import {
  cropCoverage,
  cropLabelImage,
  DEFAULT_LABEL_CROP,
  nudgeCropRegion,
} from '@/features/scan-label/cropLabelImage';
import { recognizeCareLabelText } from '@/features/scan-label/recognizeText';
import { scanCareLabelPhoto } from '@/features/scan-label/scanCareLabel';
import { CropRegion } from '@/features/scan-label/types';

type ScannerStage = 'camera' | 'crop' | 'processing';

export function CareLabelScanner({ accessToken, disabled, onCancel, onComplete, onError }: CareLabelScannerProps) {
  const cameraRef = useRef<CameraRef>(null);
  const device = useCameraDevice('back');
  const photoOutput = usePhotoOutput({
    quality: 0.9,
    qualityPrioritization: 'balanced',
  });
  const { hasPermission, requestPermission, canRequestPermission } = useCameraPermission();
  const [stage, setStage] = useState<ScannerStage>('camera');
  const [capturedPhoto, setCapturedPhoto] = useState<SelectedLabelPhoto | null>(null);
  const [cropRegion, setCropRegion] = useState<CropRegion>(DEFAULT_LABEL_CROP);
  const [previewReadyAt, setPreviewReadyAt] = useState<number | null>(null);
  const [stableForMs, setStableForMs] = useState(0);

  useEffect(() => {
    if (stage !== 'camera' || !previewReadyAt) {
      return;
    }

    const interval = setInterval(() => {
      setStableForMs(Date.now() - previewReadyAt);
    }, 250);

    return () => clearInterval(interval);
  }, [previewReadyAt, stage]);

  const liveHints = useMemo(() => buildLiveQualityHints(stableForMs), [stableForMs]);
  const captureBlocked = Boolean(disabled) || !hasPermission || stage !== 'camera' || liveHints.some((hint) => hint.blocksCapture);

  async function handleRequestPermission() {
    try {
      const granted = await requestPermission();
      if (!granted) {
        throw new AddGarmentActionError('camera-permission-denied');
      }
    } catch (error) {
      onError(error);
    }
  }

  async function handleCapture() {
    try {
      if (!hasPermission || !device) {
        throw new AddGarmentActionError('camera-unavailable');
      }

      setStage('processing');
      const photo = await photoOutput.capturePhoto(
        {
          enableShutterSound: true,
          flashMode: 'off',
        },
        {}
      );
      const path = await photo.saveToTemporaryFileAsync();
      const nextPhoto: SelectedLabelPhoto = {
        uri: path.startsWith('file://') ? path : `file://${path}`,
        fileName: `care-label-${Math.round(photo.timestamp * 1000)}.jpg`,
        mimeType: 'image/jpeg',
        width: photo.width,
        height: photo.height,
        fileSize: null,
        source: 'camera',
      };
      photo.dispose();

      setCapturedPhoto(nextPhoto);
      setCropRegion(DEFAULT_LABEL_CROP);
      setStage('crop');
    } catch (error) {
      setStage('camera');
      onError(error);
    }
  }

  async function handleUseCrop() {
    if (!capturedPhoto) {
      return;
    }

    try {
      setStage('processing');
      const croppedPhoto = await cropLabelImage(capturedPhoto, cropRegion);
      const clientOCR = await recognizeCareLabelText(croppedPhoto.uri);
      const quality = assessCaptureQuality({
        photo: croppedPhoto,
        cropCoverage: cropCoverage(cropRegion),
        stableForMs,
        ocrText: clientOCR?.text,
        ocrLineCount: clientOCR?.blocks.reduce((total, block) => total + block.lines.length, 0) ?? 0,
      });
      const result = await scanCareLabelPhoto(
        {
          photo: croppedPhoto,
          clientOCR,
          quality,
        },
        {
          accessToken,
        }
      );

      onComplete(result);
      setCapturedPhoto(null);
      setStage('camera');
    } catch (error) {
      setStage(capturedPhoto ? 'crop' : 'camera');
      onError(error);
    }
  }

  function handleRetake() {
    setCapturedPhoto(null);
    setCropRegion(DEFAULT_LABEL_CROP);
    setPreviewReadyAt(null);
    setStableForMs(0);
    setStage('camera');
  }

  if (!hasPermission) {
    return (
      <ScannerShell onCancel={onCancel}>
        <View style={styles.permissionPanel}>
          <Text style={styles.title}>Camera access is needed</Text>
          <Text style={styles.body}>
            FreshCycle uses the camera only to capture the care label crop for OCR and interpretation.
          </Text>
          <Pressable
            accessibilityLabel="Allow camera access"
            disabled={!canRequestPermission || disabled}
            onPress={() => void handleRequestPermission()}
            style={[styles.primaryButton, (!canRequestPermission || disabled) && styles.buttonDisabled]}>
            <Text style={styles.primaryButtonText}>Allow camera</Text>
          </Pressable>
        </View>
      </ScannerShell>
    );
  }

  if (!device) {
    return (
      <ScannerShell onCancel={onCancel}>
        <View style={styles.permissionPanel}>
          <Text style={styles.title}>Camera unavailable</Text>
          <Text style={styles.body}>Choose a label from the photo library below, or retry on a device with a back camera.</Text>
        </View>
      </ScannerShell>
    );
  }

  return (
    <ScannerShell onCancel={stage === 'processing' ? () => undefined : onCancel}>
      {stage === 'camera' && (
        <View style={styles.cameraPanel}>
          <View style={styles.cameraFrame}>
            <Camera
              device={device}
              enableNativeTapToFocusGesture
              isActive={stage === 'camera' && !disabled}
              onPreviewStarted={() => {
                const now = Date.now();
                setPreviewReadyAt(now);
                setStableForMs(0);
              }}
              outputs={[photoOutput]}
              ref={cameraRef}
              resizeMode="cover"
              style={StyleSheet.absoluteFill}
            />
            <View pointerEvents="none" style={styles.cameraScrim} />
            <View pointerEvents="none" style={styles.guideFrame}>
              <View style={styles.guideCornerTopLeft} />
              <View style={styles.guideCornerTopRight} />
              <View style={styles.guideCornerBottomLeft} />
              <View style={styles.guideCornerBottomRight} />
            </View>
          </View>

          <View style={styles.hintStack}>
            {liveHints.map((hint) => (
              <Text key={hint.issue} style={styles.hintText}>
                {hint.message}
              </Text>
            ))}
          </View>

          <Pressable
            accessibilityLabel="Capture care label"
            disabled={captureBlocked}
            onPress={() => void handleCapture()}
            style={[styles.captureButton, captureBlocked && styles.buttonDisabled]}>
            <View style={styles.captureButtonInner} />
          </Pressable>
        </View>
      )}

      {stage === 'crop' && capturedPhoto && (
        <View style={styles.cropPanel}>
          <View style={styles.cropPreview}>
            <Image resizeMode="cover" source={{ uri: capturedPhoto.uri }} style={styles.cropImage} />
            <View
              pointerEvents="none"
              style={[
                styles.cropOverlay,
                {
                  height: `${cropRegion.heightRatio * 100}%`,
                  left: `${cropRegion.originXRatio * 100}%`,
                  top: `${cropRegion.originYRatio * 100}%`,
                  width: `${cropRegion.widthRatio * 100}%`,
                },
              ]}
            />
          </View>

          <View style={styles.cropControls}>
            <Text style={styles.sectionTitle}>Crop label region</Text>
            <View style={styles.controlGrid}>
              <CropButton label="Up" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { originYRatio: -0.03 }))} />
              <CropButton label="Down" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { originYRatio: 0.03 }))} />
              <CropButton label="Left" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { originXRatio: -0.03 }))} />
              <CropButton label="Right" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { originXRatio: 0.03 }))} />
              <CropButton label="Wider" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { widthRatio: 0.04 }))} />
              <CropButton label="Narrower" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { widthRatio: -0.04 }))} />
              <CropButton label="Taller" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { heightRatio: 0.04 }))} />
              <CropButton label="Shorter" onPress={() => setCropRegion((region) => nudgeCropRegion(region, { heightRatio: -0.04 }))} />
            </View>
          </View>

          <View style={styles.buttonRow}>
            <Pressable accessibilityLabel="Use cropped label" onPress={() => void handleUseCrop()} style={styles.primaryButton}>
              <Text style={styles.primaryButtonText}>Use crop</Text>
            </Pressable>
            <Pressable accessibilityLabel="Retake care label" onPress={handleRetake} style={styles.secondaryButton}>
              <Text style={styles.secondaryButtonText}>Retake</Text>
            </Pressable>
          </View>
        </View>
      )}

      {stage === 'processing' && (
        <View style={styles.processingPanel}>
          <ActivityIndicator color={palette.ink} />
          <Text style={styles.processingTitle}>Reading care label</Text>
          <Text style={styles.body}>FreshCycle is cropping the label, running OCR, and asking the backend for structured care instructions.</Text>
        </View>
      )}
    </ScannerShell>
  );
}

function ScannerShell({ children, onCancel }: { children: React.ReactNode; onCancel: () => void }) {
  return (
    <View style={styles.shell}>
      <View style={styles.headerRow}>
        <View style={styles.headerCopy}>
          <Text style={styles.eyebrow}>Assisted scanner</Text>
          <Text style={styles.title}>Align the care label</Text>
        </View>
        <Pressable accessibilityLabel="Close scanner" onPress={onCancel} style={styles.closeButton}>
          <Text style={styles.closeButtonText}>Close</Text>
        </Pressable>
      </View>
      {children}
    </View>
  );
}

function CropButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable accessibilityLabel={`Crop ${label.toLowerCase()}`} onPress={onPress} style={styles.cropButton}>
      <Text style={styles.cropButtonText}>{label}</Text>
    </Pressable>
  );
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
  permissionPanel: {
    backgroundColor: '#f8f4ea',
    borderColor: palette.border,
    borderRadius: 14,
    borderWidth: 1,
    gap: 12,
    padding: 16,
  },
  cameraPanel: {
    alignItems: 'center',
    gap: 14,
  },
  cameraFrame: {
    aspectRatio: 0.72,
    backgroundColor: '#111813',
    borderRadius: 18,
    overflow: 'hidden',
    width: '100%',
  },
  cameraScrim: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: 'rgba(0, 0, 0, 0.08)',
  },
  guideFrame: {
    borderColor: '#f8f4ea',
    borderRadius: 16,
    borderWidth: 2,
    height: '34%',
    left: '8%',
    position: 'absolute',
    top: '32%',
    width: '84%',
  },
  guideCornerTopLeft: {
    borderColor: '#f8f4ea',
    borderLeftWidth: 5,
    borderTopWidth: 5,
    height: 32,
    left: -2,
    position: 'absolute',
    top: -2,
    width: 32,
  },
  guideCornerTopRight: {
    borderColor: '#f8f4ea',
    borderRightWidth: 5,
    borderTopWidth: 5,
    height: 32,
    position: 'absolute',
    right: -2,
    top: -2,
    width: 32,
  },
  guideCornerBottomLeft: {
    borderBottomWidth: 5,
    borderColor: '#f8f4ea',
    borderLeftWidth: 5,
    bottom: -2,
    height: 32,
    left: -2,
    position: 'absolute',
    width: 32,
  },
  guideCornerBottomRight: {
    borderBottomWidth: 5,
    borderColor: '#f8f4ea',
    borderRightWidth: 5,
    bottom: -2,
    height: 32,
    position: 'absolute',
    right: -2,
    width: 32,
  },
  hintStack: {
    alignSelf: 'stretch',
    gap: 6,
  },
  hintText: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '600',
    lineHeight: 20,
  },
  captureButton: {
    alignItems: 'center',
    backgroundColor: '#f8f4ea',
    borderColor: palette.ink,
    borderRadius: 999,
    borderWidth: 3,
    height: 70,
    justifyContent: 'center',
    width: 70,
  },
  captureButtonInner: {
    backgroundColor: palette.ink,
    borderRadius: 999,
    height: 50,
    width: 50,
  },
  cropPanel: {
    gap: 14,
  },
  cropPreview: {
    aspectRatio: 0.72,
    backgroundColor: '#111813',
    borderRadius: 18,
    overflow: 'hidden',
    width: '100%',
  },
  cropImage: {
    height: '100%',
    width: '100%',
  },
  cropOverlay: {
    borderColor: '#f8f4ea',
    borderRadius: 14,
    borderWidth: 3,
    position: 'absolute',
  },
  cropControls: {
    gap: 10,
  },
  sectionTitle: {
    color: palette.ink,
    fontSize: 16,
    fontWeight: '700',
  },
  controlGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: 8,
  },
  cropButton: {
    alignItems: 'center',
    backgroundColor: '#f8f4ea',
    borderColor: palette.border,
    borderRadius: 999,
    borderWidth: 1,
    minHeight: 40,
    minWidth: 82,
    justifyContent: 'center',
    paddingHorizontal: 12,
  },
  cropButtonText: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '700',
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
  processingPanel: {
    alignItems: 'center',
    backgroundColor: '#f8f4ea',
    borderColor: palette.border,
    borderRadius: 14,
    borderWidth: 1,
    gap: 10,
    padding: 18,
  },
  processingTitle: {
    color: palette.ink,
    fontSize: 17,
    fontWeight: '700',
  },
  buttonDisabled: {
    opacity: 0.45,
  },
});
