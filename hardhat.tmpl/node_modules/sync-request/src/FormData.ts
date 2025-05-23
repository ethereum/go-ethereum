export interface FormDataEntry {
  key: string;
  value: string | Blob | Buffer;
  fileName?: string;
}
export class FormData {
  private _entries: FormDataEntry[] = [];
  append(key: string, value: string | Blob | Buffer, fileName?: string): void {
    this._entries.push({key, value, fileName});
  }
}

export function getFormDataEntries(fd: FormData): FormDataEntry[] {
  return (fd as any)._entries;
}
