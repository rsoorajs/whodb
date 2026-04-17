import type { DatabaseType as CompatDatabaseType } from '../config/source-types';

declare module '@graphql' {
  export const DatabaseType: typeof import('../config/source-types').DatabaseType;
  export type DatabaseType = CompatDatabaseType;
  export type LoginCredentials = SourceLoginInput;
  export type StorageUnit = GetStorageUnitsQuery['StorageUnit'][number];
}
