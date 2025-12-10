/**
 * Listener Types
 *
 * Types for managing C2 listeners
 */

export type ListenerType = 'http' | 'https' | 'tcp' | 'smb';
export type ListenerStatus = 'running' | 'stopped' | 'error';

export interface Listener {
  name: string;
  type: ListenerType;
  bindAddress: string;
  port: number;
  status: ListenerStatus;
  metadata?: {
    createdAt?: Date;
    connections?: number;
    tlsEnabled?: boolean;
    certFile?: string;
    keyFile?: string;
  };
}

export interface ListenerStats {
  totalListeners: number;
  runningListeners: number;
  stoppedListeners: number;
  errorListeners: number;
}

export interface CreateListenerPayload {
  name: string;
  type: ListenerType;
  bindAddress: string;
  port: number;
  tlsEnabled?: boolean;
  certFile?: string;
  keyFile?: string;
  uriPath: string;
  encryptionKey: string;
}
