import { getAppEnv } from '@/lib/env';
import { AddGarmentErrorCode } from './types';

export type SaveGarmentPayload = {
  id: string;
  name: string;
  category: string | null;
  primary_color: string | null;
  wash_temperature_c: number | null;
  care_instructions: string[];
  label_image_path?: string | null;
};

export type SavedGarment = {
  id: string;
  user_id: string;
  name: string;
  category: string | null;
  primary_color: string | null;
  wash_temperature_c: number | null;
  care_instructions: string[];
  label_image_path: string | null;
};

type SaveGarmentDeps = {
  accessToken?: string | null;
  apiBaseUrl?: string | null;
  fetchImpl?: typeof fetch;
};

export async function saveGarment(payload: SaveGarmentPayload, deps: SaveGarmentDeps = {}) {
  const accessToken = deps.accessToken ?? null;
  const apiBaseUrl = deps.apiBaseUrl ?? getAppEnv().apiBaseUrl;
  const fetchImpl = deps.fetchImpl ?? ((input, init) => fetch(input, init));

  if (!accessToken) {
    throw new Error('auth-required');
  }

  if (!apiBaseUrl) {
    throw new Error('api-unavailable');
  }

  const response = await fetchImpl(`${apiBaseUrl.replace(/\/$/, '')}/garments`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${accessToken}`,
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    const errorCode = await extractSaveErrorCode(response);
    throw new Error(errorCode);
  }

  return (await response.json()) as SavedGarment;
}

async function extractSaveErrorCode(response: Response): Promise<AddGarmentErrorCode> {
  if (response.status === 401) {
    return 'auth-required';
  }

  try {
    const body = (await response.json()) as { error?: string };

    switch (body.error) {
      case 'auth_required':
      case 'invalid_token':
        return 'auth-required';
      case 'name_required':
        return 'name-required';
      case 'invalid_wash_temperature':
        return 'invalid-wash-temperature';
      case 'invalid_garment_id':
        return 'invalid-garment-id';
      case 'invalid_label_image_path':
        return 'invalid-label-image-path';
      default:
        return 'save-failed';
    }
  } catch {
    return 'save-failed';
  }
}
