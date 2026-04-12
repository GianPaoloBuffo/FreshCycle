import { Link } from 'expo-router';
import * as ImagePicker from 'expo-image-picker';
import { useEffect, useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import {
  ActivityIndicator,
  Image,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  useWindowDimensions,
  View,
} from 'react-native';

import {
  AddGarmentActionError,
  createSelectedLabelPhoto,
  describeAddGarmentError,
  parseCareLabelPhoto,
} from '@/features/add-garment/parseCareLabel';
import { saveGarment, SavedGarment } from '@/features/add-garment/saveGarment';
import { createSignedLabelImageUrl, uploadLabelImage } from '@/features/add-garment/uploadLabelImage';
import { logAddGarmentError, logAddGarmentEvent } from '@/features/add-garment/observability';
import { ParsedLabelResult, SelectedLabelPhoto } from '@/features/add-garment/types';
import { AppScreen } from '@/components/AppScreen';
import { palette } from '@/constants/theme';
import { useAuth } from '@/hooks/useAuth';

type FlowStatus = 'idle' | 'selecting' | 'processing' | 'ready';
type GarmentReviewFormValues = {
  name: string;
  category: string;
  primaryColor: string;
  washTemperatureC: string;
  careInstructionsText: string;
  fabricNotesText: string;
  rawLabelText: string;
  machineWashable: boolean;
  tumbleDry: boolean;
  dryCleanOnly: boolean;
  ironAllowed: boolean;
  ironTemp: '' | 'low' | 'medium' | 'high';
  bleachAllowed: boolean;
};

type PreparedGarmentPayload = {
  id: string;
  name: string;
  category: string | null;
  primary_color: string | null;
  wash_temperature_c: number | null;
  care_instructions: string[];
  label_image_path: string | null;
};

export function AddGarmentScreen() {
  const { authReady, loading, session } = useAuth();
  const { width } = useWindowDimensions();
  const [selectedPhoto, setSelectedPhoto] = useState<SelectedLabelPhoto | null>(null);
  const [parseResult, setParseResult] = useState<ParsedLabelResult | null>(null);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [reviewMessage, setReviewMessage] = useState<string | null>(null);
  const [preparedPayload, setPreparedPayload] = useState<PreparedGarmentPayload | null>(null);
  const [savedGarment, setSavedGarment] = useState<SavedGarment | null>(null);
  const [savedLabelImageUrl, setSavedLabelImageUrl] = useState<string | null>(null);
  const [status, setStatus] = useState<FlowStatus>('idle');
  const {
    control,
    formState: { errors, isSubmitting },
    handleSubmit,
    reset,
  } = useForm<GarmentReviewFormValues>({
    defaultValues: createInitialReviewDefaults(),
  });

  const canUseCamera = true;
  const isBusy = status === 'selecting' || status === 'processing' || isSubmitting;
  const stackButtons = width < 720;
  const hasCapturedPhoto = Boolean(selectedPhoto);
  const showReviewForm = Boolean(parseResult);

  useEffect(() => {
    if (!parseResult) {
      reset(createInitialReviewDefaults());
      setPreparedPayload(null);
      setReviewMessage(null);
      setSavedGarment(null);
      setSavedLabelImageUrl(null);
      return;
    }

    reset(buildReviewDefaults(parseResult));
    setPreparedPayload(null);
    setReviewMessage(null);
    setSavedGarment(null);
    setSavedLabelImageUrl(null);
  }, [parseResult, reset]);

  function resetFlow() {
    setSelectedPhoto(null);
    setParseResult(null);
    setErrorMessage(null);
    setReviewMessage(null);
    setPreparedPayload(null);
    setSavedGarment(null);
    setSavedLabelImageUrl(null);
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
      setReviewMessage(null);
      setPreparedPayload(null);
      setSavedGarment(null);

      const nextResult = await parseCareLabelPhoto(nextPhoto, {
        accessToken: session?.access_token ?? null,
      });

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

  const submitReview = handleSubmit(async (values) => {
    try {
      if (!selectedPhoto || !session?.user.id) {
        throw new Error('upload-failed');
      }

      logAddGarmentEvent('label_upload_started', {
        source: selectedPhoto.source,
        fileName: selectedPhoto.fileName,
      });

      const uploadedLabel = await uploadLabelImage(selectedPhoto, session.user.id);
      logAddGarmentEvent('label_upload_succeeded', {
        garmentId: uploadedLabel.garmentId,
        objectPath: uploadedLabel.objectPath,
      });

      const payload = buildPreparedGarmentPayload(values, uploadedLabel);
      const garment = await saveGarment(payload, {
        accessToken: session?.access_token ?? null,
      });

      setPreparedPayload(payload);
      setSavedGarment(garment);
      try {
        const signedLabelImageUrl = await createSignedLabelImageUrl(uploadedLabel.objectPath);
        setSavedLabelImageUrl(signedLabelImageUrl);
      } catch (error) {
        setSavedLabelImageUrl(null);
        logAddGarmentError('garment_save_failed', error, {
          reason: 'signed-url-generation-failed',
          garmentId: garment.id,
        });
      }
      setReviewMessage('Garment saved to your FreshCycle wardrobe.');
      logAddGarmentEvent('review_form_submitted', {
        hasCategory: Boolean(payload.category),
        careInstructionCount: payload.care_instructions.length,
        washTemperatureSet: payload.wash_temperature_c !== null,
      });
      logAddGarmentEvent('garment_save_succeeded', {
        garmentId: garment.id,
        hasCategory: Boolean(garment.category),
        hasLabelImagePath: Boolean(garment.label_image_path),
      });
    } catch (error) {
      const code =
        error instanceof Error && error.message === 'auth-required'
          ? 'auth-required'
          : error instanceof Error && error.message === 'upload-failed'
            ? 'upload-failed'
            : 'save-failed';

      if (code === 'upload-failed') {
        logAddGarmentError('label_upload_failed', error, {});
      }

      setReviewMessage(describeAddGarmentError(code));
      logAddGarmentError('garment_save_failed', error, {});
      logAddGarmentError('review_form_failed', error, {});
    }
  });

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
          <>
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

            {showReviewForm && (
              <View style={styles.reviewCard}>
                <Text style={styles.sectionTitle}>Review garment details</Text>
                <Text style={styles.sectionBody}>
                  FreshCycle prefilled these fields from the parser output. Adjust anything that looks off, then we
                  will upload the private label image and save the garment record together.
                </Text>

                <View style={styles.formGrid}>
                  <View style={styles.fieldGroup}>
                    <Text style={styles.fieldLabel}>Garment name</Text>
                    <Controller
                      control={control}
                      name="name"
                      rules={{
                        required: 'Add a garment name before saving.',
                      }}
                      render={({ field: { onBlur, onChange, value } }) => (
                        <TextInput
                          onBlur={onBlur}
                          onChangeText={onChange}
                          placeholder="e.g. Navy Hoodie"
                          placeholderTextColor="#77887a"
                          style={[styles.textInput, errors.name && styles.textInputError]}
                          value={value}
                        />
                      )}
                    />
                    {errors.name && <Text style={styles.validationText}>{errors.name.message}</Text>}
                  </View>

                  <View style={styles.fieldGroup}>
                    <Text style={styles.fieldLabel}>Category</Text>
                    <Controller
                      control={control}
                      name="category"
                      render={({ field: { onBlur, onChange, value } }) => (
                        <TextInput
                          onBlur={onBlur}
                          onChangeText={onChange}
                          placeholder="Tops, Knitwear, Outerwear..."
                          placeholderTextColor="#77887a"
                          style={styles.textInput}
                          value={value}
                        />
                      )}
                    />
                  </View>

                  <View style={styles.fieldGroup}>
                    <Text style={styles.fieldLabel}>Primary color</Text>
                    <Controller
                      control={control}
                      name="primaryColor"
                      render={({ field: { onBlur, onChange, value } }) => (
                        <TextInput
                          onBlur={onBlur}
                          onChangeText={onChange}
                          placeholder="Optional"
                          placeholderTextColor="#77887a"
                          style={styles.textInput}
                          value={value}
                        />
                      )}
                    />
                  </View>

                  <View style={styles.fieldGroup}>
                    <Text style={styles.fieldLabel}>Wash temperature (C)</Text>
                    <Controller
                      control={control}
                      name="washTemperatureC"
                      rules={{
                        validate: (value) => {
                          if (!value.trim()) {
                            return true;
                          }

                          const parsedValue = Number(value);
                          if (!Number.isInteger(parsedValue) || parsedValue < 0 || parsedValue > 95) {
                            return 'Use a whole number between 0 and 95.';
                          }

                          return true;
                        },
                      }}
                      render={({ field: { onBlur, onChange, value } }) => (
                        <TextInput
                          keyboardType="number-pad"
                          onBlur={onBlur}
                          onChangeText={onChange}
                          placeholder="Optional"
                          placeholderTextColor="#77887a"
                          style={[styles.textInput, errors.washTemperatureC && styles.textInputError]}
                          value={value}
                        />
                      )}
                    />
                    {errors.washTemperatureC && (
                      <Text style={styles.validationText}>{errors.washTemperatureC.message}</Text>
                    )}
                  </View>
                </View>

                <View style={styles.toggleGrid}>
                  <ControlledSwitch control={control} name="machineWashable" label="Machine washable" />
                  <ControlledSwitch control={control} name="tumbleDry" label="Tumble dry allowed" />
                  <ControlledSwitch control={control} name="dryCleanOnly" label="Dry clean only" />
                  <ControlledSwitch control={control} name="ironAllowed" label="Iron allowed" />
                  <ControlledSwitch control={control} name="bleachAllowed" label="Bleach allowed" />
                </View>

                <View style={styles.fieldGroup}>
                  <Text style={styles.fieldLabel}>Iron temperature</Text>
                  <Controller
                    control={control}
                    name="ironTemp"
                    render={({ field: { value, onChange } }) => (
                      <View style={[styles.buttonRow, stackButtons && styles.buttonRowStacked]}>
                        {['', 'low', 'medium', 'high'].map((option) => {
                          const label = option === '' ? 'Unset' : option.charAt(0).toUpperCase() + option.slice(1);
                          const isSelected = value === option;

                          return (
                            <Pressable
                              key={option || 'unset'}
                              onPress={() => onChange(option)}
                              style={[
                                styles.choiceChip,
                                isSelected && styles.choiceChipSelected,
                                stackButtons && styles.fullWidthButton,
                              ]}>
                              <Text style={[styles.choiceChipText, isSelected && styles.choiceChipTextSelected]}>
                                {label}
                              </Text>
                            </Pressable>
                          );
                        })}
                      </View>
                    )}
                  />
                </View>

                <View style={styles.fieldGroup}>
                  <Text style={styles.fieldLabel}>Care instructions</Text>
                  <Controller
                    control={control}
                    name="careInstructionsText"
                    render={({ field: { onBlur, onChange, value } }) => (
                      <TextInput
                        multiline
                        numberOfLines={4}
                        onBlur={onBlur}
                        onChangeText={onChange}
                        placeholder="One instruction per line"
                        placeholderTextColor="#77887a"
                        style={[styles.textInput, styles.textArea]}
                        value={value}
                      />
                    )}
                  />
                </View>

                <View style={styles.fieldGroup}>
                  <Text style={styles.fieldLabel}>Fabric notes</Text>
                  <Controller
                    control={control}
                    name="fabricNotesText"
                    render={({ field: { onBlur, onChange, value } }) => (
                      <TextInput
                        multiline
                        numberOfLines={3}
                        onBlur={onBlur}
                        onChangeText={onChange}
                        placeholder="One note per line"
                        placeholderTextColor="#77887a"
                        style={[styles.textInput, styles.textArea]}
                        value={value}
                      />
                    )}
                  />
                </View>

                <View style={styles.fieldGroup}>
                  <Text style={styles.fieldLabel}>Detected label text</Text>
                  <Controller
                    control={control}
                    name="rawLabelText"
                    render={({ field: { onBlur, onChange, value } }) => (
                      <TextInput
                        multiline
                        numberOfLines={4}
                        onBlur={onBlur}
                        onChangeText={onChange}
                        placeholder="Detected OCR text"
                        placeholderTextColor="#77887a"
                        style={[styles.textInput, styles.textArea]}
                        value={value}
                      />
                    )}
                  />
                </View>

                <View style={[styles.buttonRow, stackButtons && styles.buttonRowStacked]}>
                  <Pressable
                    accessibilityLabel="Save garment details"
                    disabled={!session || isBusy}
                    onPress={() => void submitReview()}
                    style={[
                      styles.primaryButton,
                      stackButtons && styles.fullWidthButton,
                      (!session || isBusy) && styles.buttonDisabled,
                    ]}>
                    <Text style={styles.primaryButtonText}>Save garment details</Text>
                  </Pressable>
                  <Pressable
                    accessibilityLabel="Reset garment review defaults"
                    disabled={!parseResult || isBusy}
                    onPress={() => {
                      if (!parseResult) {
                        return;
                      }

                      reset(buildReviewDefaults(parseResult));
                      setPreparedPayload(null);
                      setReviewMessage(null);
                    }}
                    style={[
                      styles.secondaryButton,
                      stackButtons && styles.fullWidthButton,
                      (!parseResult || isBusy) && styles.buttonDisabled,
                    ]}>
                    <Text style={styles.secondaryButtonText}>Reset review fields</Text>
                  </Pressable>
                </View>

                {reviewMessage && <Text style={styles.reviewMessage}>{reviewMessage}</Text>}

                {preparedPayload && (
                  <View style={styles.payloadCard}>
                    <Text style={styles.payloadTitle}>Saved garment payload</Text>
                    <Text style={styles.payloadBody}>
                      This is the garment object FreshCycle just sent to the save endpoint.
                    </Text>
                    <Text style={styles.payloadCode}>{formatPayload(preparedPayload)}</Text>
                  </View>
                )}

                {savedGarment && (
                  <View style={styles.successCard}>
                    <Text style={styles.successTitle}>Garment saved</Text>
                    <Text style={styles.successBody}>
                      Saved as <Text style={styles.successStrong}>{savedGarment.name}</Text> with id {savedGarment.id}.
                    </Text>
                    {savedGarment.label_image_path && (
                      <Text style={styles.successBody}>Stored label path: {savedGarment.label_image_path}</Text>
                    )}
                    {savedLabelImageUrl && (
                      <Image source={{ uri: savedLabelImageUrl }} style={styles.savedLabelImage} />
                    )}
                  </View>
                )}
              </View>
            )}
          </>
        )}
      </ScrollView>
    </AppScreen>
  );
}

function createInitialReviewDefaults(): GarmentReviewFormValues {
  return {
    name: '',
    category: '',
    primaryColor: '',
    washTemperatureC: '',
    careInstructionsText: '',
    fabricNotesText: '',
    rawLabelText: '',
    machineWashable: false,
    tumbleDry: false,
    dryCleanOnly: false,
    ironAllowed: false,
    ironTemp: '',
    bleachAllowed: false,
  };
}

function buildReviewDefaults(parseResult: ParsedLabelResult): GarmentReviewFormValues {
  return {
    name: parseResult.parsed.nameSuggestion,
    category: parseResult.parsed.category,
    primaryColor: parseResult.parsed.primaryColor,
    washTemperatureC:
      parseResult.parsed.washTemperatureC === null ? '' : String(parseResult.parsed.washTemperatureC),
    careInstructionsText: parseResult.parsed.careInstructions.join('\n'),
    fabricNotesText: parseResult.parsed.fabricNotes.join('\n'),
    rawLabelText: parseResult.parsed.rawLabelText,
    machineWashable: parseResult.parsed.machineWashable,
    tumbleDry: parseResult.parsed.tumbleDry,
    dryCleanOnly: parseResult.parsed.dryCleanOnly,
    ironAllowed: parseResult.parsed.ironAllowed,
    ironTemp: parseResult.parsed.ironTemp ?? '',
    bleachAllowed: parseResult.parsed.bleachAllowed,
  };
}

function buildPreparedGarmentPayload(
  values: GarmentReviewFormValues,
  uploadedLabel: { garmentId: string; objectPath: string }
): PreparedGarmentPayload {
  const careInstructions = [
    ...values.careInstructionsText
      .split('\n')
      .map((value) => value.trim())
      .filter(Boolean),
    ...buildInstructionFlags(values),
  ];

  return {
    id: uploadedLabel.garmentId,
    name: values.name.trim(),
    category: emptyToNull(values.category),
    primary_color: emptyToNull(values.primaryColor),
    wash_temperature_c: values.washTemperatureC.trim() ? Number(values.washTemperatureC) : null,
    care_instructions: Array.from(new Set(careInstructions)),
    label_image_path: uploadedLabel.objectPath,
  };
}

function buildInstructionFlags(values: GarmentReviewFormValues) {
  const instructions = [];

  if (values.machineWashable) {
    instructions.push('Machine washable');
  }
  if (values.tumbleDry) {
    instructions.push('Tumble dry allowed');
  }
  if (values.dryCleanOnly) {
    instructions.push('Dry clean only');
  }
  if (values.ironAllowed) {
    instructions.push(values.ironTemp ? `Iron on ${values.ironTemp} heat` : 'Iron allowed');
  }
  if (!values.bleachAllowed) {
    instructions.push('Do not bleach');
  }

  return instructions;
}

function emptyToNull(value: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : null;
}

function formatPayload(payload: PreparedGarmentPayload) {
  return JSON.stringify(payload, null, 2);
}

function ControlledSwitch({
  control,
  label,
  name,
}: {
  control: ReturnType<typeof useForm<GarmentReviewFormValues>>['control'];
  label: string;
  name:
    | 'machineWashable'
    | 'tumbleDry'
    | 'dryCleanOnly'
    | 'ironAllowed'
    | 'bleachAllowed';
}) {
  return (
    <Controller
      control={control}
      name={name}
      render={({ field: { onChange, value } }) => (
        <View style={styles.toggleRow}>
          <Text style={styles.toggleLabel}>{label}</Text>
          <Switch
            onValueChange={onChange}
            thumbColor={value ? '#f6efe2' : '#f6efe2'}
            trackColor={{ false: '#d6cab3', true: palette.accent }}
            value={value}
          />
        </View>
      )}
    />
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
  reviewCard: {
    backgroundColor: '#f7eddc',
    borderColor: palette.border,
    borderRadius: 24,
    borderWidth: 1,
    gap: 14,
    padding: 20,
  },
  formGrid: {
    gap: 14,
  },
  fieldGroup: {
    gap: 8,
  },
  fieldLabel: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '600',
  },
  textInput: {
    backgroundColor: '#fffaf2',
    borderColor: '#d0c1a7',
    borderRadius: 14,
    borderWidth: 1,
    color: palette.ink,
    fontSize: 15,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  textInputError: {
    borderColor: '#b5523f',
  },
  textArea: {
    minHeight: 96,
    textAlignVertical: 'top',
  },
  validationText: {
    color: '#8f3e2f',
    fontSize: 13,
    lineHeight: 18,
  },
  toggleGrid: {
    gap: 12,
  },
  toggleRow: {
    alignItems: 'center',
    backgroundColor: '#efe5d4',
    borderRadius: 16,
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  toggleLabel: {
    color: palette.ink,
    flex: 1,
    fontSize: 15,
    fontWeight: '500',
    paddingRight: 12,
  },
  choiceChip: {
    alignItems: 'center',
    backgroundColor: '#e9ddc8',
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 12,
  },
  choiceChipSelected: {
    backgroundColor: palette.ink,
  },
  choiceChipText: {
    color: palette.ink,
    fontSize: 14,
    fontWeight: '600',
  },
  choiceChipTextSelected: {
    color: '#f7f2e8',
  },
  reviewMessage: {
    color: palette.accent,
    fontSize: 14,
    fontWeight: '600',
    lineHeight: 20,
  },
  payloadCard: {
    backgroundColor: '#e6dcc9',
    borderRadius: 18,
    gap: 8,
    padding: 16,
  },
  payloadTitle: {
    color: palette.ink,
    fontSize: 16,
    fontWeight: '700',
  },
  payloadBody: {
    color: palette.inkMuted,
    fontSize: 14,
    lineHeight: 20,
  },
  payloadCode: {
    color: palette.ink,
    fontFamily: Platform.select({ ios: 'Menlo', android: 'monospace', default: 'monospace' }),
    fontSize: 13,
    lineHeight: 19,
  },
  successCard: {
    backgroundColor: '#dbe8d5',
    borderColor: '#a3be98',
    borderRadius: 18,
    borderWidth: 1,
    gap: 6,
    padding: 16,
  },
  successTitle: {
    color: '#224126',
    fontSize: 16,
    fontWeight: '700',
  },
  successBody: {
    color: '#35523b',
    fontSize: 14,
    lineHeight: 20,
  },
  successStrong: {
    fontWeight: '700',
  },
  savedLabelImage: {
    borderRadius: 18,
    height: 220,
    marginTop: 12,
    width: '100%',
  },
});
