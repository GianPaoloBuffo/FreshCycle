export type WardrobeGarment = {
  id: string;
  user_id: string;
  name: string;
  category: string | null;
  primary_color: string | null;
  wash_temperature_c: number | null;
  care_instructions: string[];
  label_image_path: string | null;
};

export type WardrobeGroupingMode = 'flat' | 'colour' | 'temperature' | 'category';

export type WardrobeCareMethod = 'machine_wash' | 'hand_wash' | 'dry_clean';

export type WardrobeMachineTempBucket = '30' | '40' | '60' | '90' | 'unknown';

export type WardrobeColourGroup = 'white' | 'light' | 'dark' | 'colour';

export type NormalizedWardrobeGarment = WardrobeGarment & {
  category_key: string;
  category_label: string;
  care_method: WardrobeCareMethod;
  colour_group: WardrobeColourGroup;
  machine_temp_bucket: WardrobeMachineTempBucket;
};

export type WardrobeGroupSection = {
  key: string;
  title: string;
  description: string;
  garments: NormalizedWardrobeGarment[];
};
