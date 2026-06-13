import TextRecognition from '@react-native-ml-kit/text-recognition';

import { ClientOCRResult, OCRFrame } from '@/features/scan-label/types';

export async function recognizeCareLabelText(imageUri: string): Promise<ClientOCRResult | null> {
  const result = await TextRecognition.recognize(imageUri);

  return {
    source: 'mlkit-text-recognition-v2',
    text: result.text,
    averageConfidence: null,
    blocks: result.blocks.map((block) => ({
      text: block.text,
      frame: normalizeFrame(block.frame),
      confidence: null,
      lines: block.lines.map((line) => ({
        text: line.text,
        frame: normalizeFrame(line.frame),
        confidence: null,
        elements: line.elements.map((element) => ({
          text: element.text,
          frame: normalizeFrame(element.frame),
          confidence: null,
        })),
      })),
    })),
  };
}

function normalizeFrame(frame: { left: number; top: number; width: number; height: number } | undefined): OCRFrame | null {
  if (!frame) {
    return null;
  }

  return {
    x: frame.left,
    y: frame.top,
    width: frame.width,
    height: frame.height,
  };
}
