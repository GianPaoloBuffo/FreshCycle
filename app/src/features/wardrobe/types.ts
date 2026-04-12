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
