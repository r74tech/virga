export interface SettingsSectionProps<T> {
  settings: T;
  onUpdate: (settings: Partial<T>) => Promise<void>;
  onSave: () => void;
}
