import { Link } from 'expo-router';
import * as ImagePicker from 'expo-image-picker';
import { useState } from 'react';
import {
  ActivityIndicator,
  Image,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  useWindowDimensions,
  View,
} from 'react-native';

import {
  AddGarmentActionError,
  createSelectedLabelPhoto,
  describeAddGarmentError,
  parseCareLabelPhoto,
} from '@/features/add-garment/parseCareLabel';
import { logAddGarmentError, logAddGarmentEvent } from '@/features/add-garment/observability';
import { ParsedLabelResult, SelectedLabelPhoto } from '@/features/add-garment/types';
import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { useAuth } from '@/hooks/useAuth';

type FlowStatus = 'idle' | 'selecting' | 'processing' | 'ready';

export function AddGarmentScreen() {
  const { authReady, loading, session } = useAuth();
  const { width } = useWindowDimensions();
  const [selectedPhoto, setSelectedPhoto] = useState<SelectedLabelPhoto | null>(null);
  const [parseResult, setParseResult] = useState<ParsedLabelResult | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [status, setStatus] = useState<FlowStatus>('idle');

  const canUseCamera = true;
  const isBusy = status === 'selecting' || status === 'processing';
  const stackButtons = width < 720;
  const hasCapturedPhoto = Boolean(selectedPhoto);

  function resetFlow() {
    setSelectedPhoto(null);
    setParseResult(null);
    setErrorMessage(null);
    setStatus('idle');
  }

  async function handleSourceSelection(source: SelectedLabelPhoto['source']) {
    setErrorMessage(null);
    setStatus('selecting');
    setParseResult(null);

    logAddGarmentEvent('label_selection_started', {
      source,
      platform: Platform.OS,
    });

    try {
      const nextPhoto = await pickLabelPhoto(source);

      if (!nextPhoto) {
        setStatus('idle');
        return;
      }

      setSelectedPhoto(nextPhoto);
      setStatus('processing');

      const nextResult = await parseCareLabelPhoto(nextPhoto);

      setParseResult(nextResult);
      setStatus('ready');
      logAddGarmentEvent('label_selection_completed', {
        source,
        durationMs: nextResult.durationMs,
      });
    } catch (error) {
      const message =
        error instanceof AddGarmentActionError
          ? error.message
          : describeAddGarmentError('processing-failed');

      setErrorMessage(message);
      setStatus(selectedPhoto ? 'ready' : 'idle');
      logAddGarmentError('label_selection_failed', error, {
        source,
        platform: Platform.OS,
      });
    }
  }

  return (
    <AppScreen padded={false}>
      <ScrollView contentContainerStyle={styles.scrollContent}>
        <View style={styles.hero}>
          <Text style={styles.eyebrow}>Phase 2 label scanning</Text>
          <Text style={styles.title}>Capture the care label before we parse the garment details.</Text>
          <Text style={styles.body}>
            Start with a fresh camera shot when possible, or fall back to the photo library if the label
            is already saved on your device.
          </Text>
        </View>

        {!authReady && (
          <View style={styles.noticeCard}>
            <Text style={styles.noticeTitle}>Auth config missing</Text>
            <Text style={styles.noticeBody}>
              Add the Supabase Expo environment variables before testing the authenticated garment flow.
            </Text>
          </View>
        )}

        {authReady && loading && (
          <View style={styles.noticeCard}>
            <Text style={styles.noticeTitle}>Checking your session</Text>
            <Text style={styles.noticeBody}>FreshCycle is confirming the authenticated user before capture starts.</Text>
          </View>
        )}

        {authReady && !loading && !session && (
          <View style={styles.noticeCard}>
            <Text style={styles.noticeTitle}>Sign in to continue</Text>
            <Text style={styles.noticeBody}>
              The garment capture flow is scoped to an authenticated wardrobe, so start by signing in.
            </Text>
            <Link href="/auth" style={styles.inlineLink}>
              Open auth screens
            </Link>
          </View>
        )}

        <View style={styles.actionCard}>
          <Text style={styles.sectionTitle}>Choose a care-label photo</Text>
          <Text style={styles.sectionBody}>
            Clear, cropped label photos give the parser the best chance of producing useful garment data.
          </Text>

          <View style={[styles.buttonRow, stackButtons && styles.buttonRowStacked]}>
            <Pressable
              accessibilityLabel="Take a photo with the camera"
              disabled={!session || isBusy || !canUseCamera}
              onPress={() => void handleSourceSelection('camera')}
              style={[
                styles.primaryButton,
                stackButtons && styles.fullWidthButton,
                (!session || isBusy || !canUseCamera) && styles.buttonDisabled,
              ]}>
              <Text style={styles.primaryButtonText}>
                {Platform.OS === 'web' ? 'Use camera or camera upload' : 'Use camera'}
              </Text>
            </Pressable>

            <Pressable
              accessibilityLabel="Choose a photo from the library"
              disabled={!session || isBusy}
              onPress={() => void handleSourceSelection('library')}
              style={[
                styles.secondaryButton,
                stackButtons && styles.fullWidthButton,
                (!session || isBusy) && styles.buttonDisabled,
              ]}>
              <Text style={styles.secondaryButtonText}>Choose from library</Text>
            </Pressable>
          </View>

          <Text style={styles.helperText}>
            {Platform.OS === 'web'
              ? 'On the web, supported browsers can open the device camera from the picker. If not, the same action falls back to the browser file chooser.'
              : 'If camera access is denied, the library path stays available so the flow never dead-ends.'}
          </Text>
        </View>

        {errorMessage && (
          <View style={styles.errorCard}>
            <Text style={styles.errorTitle}>Capture needs attention</Text>
            <Text style={styles.errorBody}>{errorMessage}</Text>
            <View style={[styles.buttonRow, stackButtons && styles.buttonRowStacked]}>
              <Pressable
                accessibilityLabel="Retry capture"
                disabled={isBusy || !session}
                onPress={() => void handleSourceSelection(selectedPhoto?.source ?? 'library')}
                style={[
                  styles.errorActionButton,
                  stackButtons && styles.fullWidthButton,
                  (isBusy || !session) && styles.buttonDisabled,
                ]}>
                <Text style={styles.errorActionButtonText}>Retry</Text>
              </Pressable>
              {hasCapturedPhoto && (
                <Pressable
                  accessibilityLabel="Clear selected label photo"
                  disabled={isBusy}
                  onPress={resetFlow}
                  style={[
                    styles.secondaryButton,
                    stackButtons && styles.fullWidthButton,
                    isBusy && styles.buttonDisabled,
                  ]}>
                  <Text style={styles.secondaryButtonText}>Clear photo</Text>
                </Pressable>
              )}
            </View>
          </View>
        )}

        {selectedPhoto && (
          <View style={styles.previewCard}>
            <Text style={styles.sectionTitle}>Selected label</Text>
            <Image source={{ uri: selectedPhoto.uri }} style={styles.previewImage} />
            <View style={styles.metaGrid}>
              <Text style={styles.metaLine}>Source: {selectedPhoto.source}</Text>
              <Text style={styles.metaLine}>File: {selectedPhoto.fileName ?? 'Unknown image'}</Text>
              <Text style={styles.metaLine}>
                Dimensions: {selectedPhoto.width} x {selectedPhoto.height}
              </Text>
              <Text style={styles.metaLine}>
                Size: {selectedPhoto.fileSize ? `${Math.round(selectedPhoto.fileSize / 1024)} KB` : 'Unknown'}
              </Text>
            </View>
          </View>
        )}

        {status === 'processing' && (
          <View style={styles.processingCard}>
            <ActivityIndicator color={palette.ink} />
            <Text style={styles.processingTitle}>Parsing care label</Text>
            <Text style={styles.processingBody}>
              FreshCycle is validating the upload and preparing structured garment details for review.
            </Text>
          </View>
        )}

        {parseResult && (
          <View style={styles.resultCard}>
            <Text style={styles.sectionTitle}>Processing preview</Text>
            <Text style={styles.resultName}>{parseResult.preview.garmentName}</Text>
            <Text style={styles.resultMeta}>
              {parseResult.preview.suggestedCategory} · {parseResult.preview.confidenceLabel}
            </Text>
            <Text style={styles.resultBody}>{parseResult.preview.careSummary}</Text>
            {parseResult.preview.notes.map((note) => (
              <Text key={note} style={styles.noteItem}>
                {`\u2022 ${note}`}
              </Text>
            ))}
            <Text style={styles.metaFootnote}>
              Last processed in {parseResult.durationMs}ms at {new Date(parseResult.completedAt).toLocaleTimeString()}.
            </Text>
            <View style={[styles.buttonRow, stackButtons && styles.buttonRowStacked]}>
              <Pressable
                accessibilityLabel="Capture a different label photo"
                disabled={isBusy || !session}
                onPress={() => void handleSourceSelection('camera')}
                style={[
                  styles.primaryButton,
                  stackButtons && styles.fullWidthButton,
                  (isBusy || !session) && styles.buttonDisabled,
                ]}>
                <Text style={styles.primaryButtonText}>Retake with camera</Text>
              </Pressable>
              <Pressable
                accessibilityLabel="Choose a different label photo from the library"
                disabled={isBusy || !session}
                onPress={() => void handleSourceSelection('library')}
                style={[
                  styles.secondaryButton,
                  stackButtons && styles.fullWidthButton,
                  (isBusy || !session) && styles.buttonDisabled,
                ]}>
                <Text style={styles.secondaryButtonText}>Pick another photo</Text>
              </Pressable>
            </View>
          </View>
        )}
      </ScrollView>
    </AppScreen>
  );
}

async function pickLabelPhoto(source: SelectedLabelPhoto['source']) {
  if (source === 'camera') {
    if (Platform.OS !== 'web') {
      const permission = await ImagePicker.requestCameraPermissionsAsync();

      if (!permission.granted) {
        throw new AddGarmentActionError('camera-permission-denied');
      }
    }

    const result = await ImagePicker.launchCameraAsync(getPickerOptions());
    return normalizePickerResult(result, source);
  }

  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();

  if (!permission.granted) {
    throw new AddGarmentActionError('photo-library-permission-denied');
  }

  const result = await ImagePicker.launchImageLibraryAsync(getPickerOptions());
  return normalizePickerResult(result, source);
}

function normalizePickerResult(
  result: ImagePicker.ImagePickerResult,
  source: SelectedLabelPhoto['source']
) {
  if ('canceled' in result && result.canceled && !result.assets?.[0]) {
    return null;
  }

  const asset = result.assets?.[0];

  if (!asset) {
    return null;
  }

  return normalizeSelectedPhoto(asset, source);
}

function normalizeSelectedPhoto(
  asset: ImagePicker.ImagePickerAsset | undefined,
  source: SelectedLabelPhoto['source']
) {
  if (!asset) {
    throw new AddGarmentActionError('selection-empty');
  }

  return createSelectedLabelPhoto(asset, source);
}

function getPickerOptions(): ImagePicker.ImagePickerOptions {
  return {
    allowsEditing: true,
    mediaTypes: ['images'],
    quality: 0.8,
  };
}

const styles = StyleSheet.create({
  scrollContent: {
    paddingHorizontal: 24,
    paddingVertical: 20,
    gap: 16,
  },
  hero: {
    gap: 12,
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
    fontSize: 32,
    fontWeight: '700',
    lineHeight: 38,
  },
  body: {
    color: palette.inkMuted,
    fontSize: 16,
    lineHeight: 24,
  },
  noticeCard: {
    backgroundColor: '#fbf7ef',
    borderColor: palette.border,
    borderRadius: 20,
    borderWidth: 1,
    gap: 8,
    padding: 20,
  },
  noticeTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  noticeBody: {
    color: palette.inkMuted,
    fontSize: 15,
    lineHeight: 22,
  },
  inlineLink: {
    color: palette.ink,
    fontSize: 15,
    fontWeight: '600',
    marginTop: 4,
  },
  actionCard: {
    backgroundColor: palette.card,
    borderColor: palette.border,
    borderRadius: 24,
    borderWidth: 1,
    gap: 14,
    padding: 20,
  },
  sectionTitle: {
    color: palette.ink,
    fontSize: 20,
    fontWeight: '600',
  },
  sectionBody: {
    color: palette.inkMuted,
    fontSize: 15,
    lineHeight: 22,
  },
  buttonRow: {
    flexDirection: 'row',
    gap: 12,
    flexWrap: 'wrap',
  },
  buttonRowStacked: {
    flexDirection: 'column',
  },
  primaryButton: {
    alignItems: 'center',
    backgroundColor: palette.ink,
    borderRadius: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  primaryButtonText: {
    color: '#f8f3ea',
    fontSize: 15,
    fontWeight: '600',
  },
  secondaryButton: {
    alignItems: 'center',
    backgroundColor: palette.accentSoft,
    borderRadius: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  fullWidthButton: {
    width: '100%',
  },
  secondaryButtonText: {
    color: palette.ink,
    fontSize: 15,
    fontWeight: '600',
  },
  buttonDisabled: {
    opacity: 0.45,
  },
  helperText: {
    color: palette.inkMuted,
    fontSize: 13,
    lineHeight: 20,
  },
  errorCard: {
    backgroundColor: '#f7ddd6',
    borderColor: '#d48e7f',
    borderRadius: 20,
    borderWidth: 1,
    gap: 8,
    padding: 20,
  },
  errorTitle: {
    color: '#6e2419',
    fontSize: 17,
    fontWeight: '700',
  },
  errorBody: {
    color: '#7a3024',
    fontSize: 15,
    lineHeight: 22,
  },
  errorActionButton: {
    alignItems: 'center',
    backgroundColor: '#7a3024',
    borderRadius: 16,
    paddingHorizontal: 16,
    paddingVertical: 14,
  },
  errorActionButtonText: {
    color: '#fff4ef',
    fontSize: 15,
    fontWeight: '600',
  },
  previewCard: {
    backgroundColor: '#fcf8f0',
    borderColor: palette.border,
    borderRadius: 24,
    borderWidth: 1,
    gap: 14,
    padding: 20,
  },
  previewImage: {
    aspectRatio: 4 / 5,
    backgroundColor: '#e4dac7',
    borderRadius: 18,
    width: '100%',
  },
  metaGrid: {
    gap: 6,
  },
  metaLine: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  processingCard: {
    alignItems: 'center',
    backgroundColor: '#e8ddc7',
    borderRadius: 24,
    gap: 10,
    padding: 24,
  },
  processingTitle: {
    color: palette.ink,
    fontSize: 18,
    fontWeight: '600',
  },
  processingBody: {
    color: palette.inkMuted,
    fontSize: 15,
    lineHeight: 22,
    textAlign: 'center',
  },
  resultCard: {
    backgroundColor: '#f0e6d5',
    borderColor: palette.border,
    borderRadius: 24,
    borderWidth: 1,
    gap: 10,
    padding: 20,
  },
  resultName: {
    color: palette.ink,
    fontSize: 24,
    fontWeight: '700',
  },
  resultMeta: {
    color: palette.accent,
    fontSize: 14,
    fontWeight: '700',
    textTransform: 'uppercase',
  },
  resultBody: {
    color: palette.inkMuted,
    fontSize: 15,
    lineHeight: 22,
  },
  noteItem: {
    color: palette.ink,
    fontSize: 14,
    lineHeight: 20,
  },
  metaFootnote: {
    color: palette.inkMuted,
    fontSize: 12,
    lineHeight: 18,
    marginTop: 4,
  },
});
