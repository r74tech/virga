/**
 * API Client
 *
 * HTTP client for communication with Virga beacon-server
 */

import axios, { type AxiosInstance, type AxiosError, type AxiosRequestConfig } from 'axios';
import { config } from '../config/env';
import {
  ApiError,
  NetworkError,
  TimeoutError,
  AuthenticationError,
  type ErrorResponse,
} from '../types/errors';

/**
 * API client configuration
 */
interface ApiClientConfig {
  baseURL: string;
  timeout: number;
  headers?: Record<string, string>;
}

/**
 * API client class
 */
export class ApiClient {
  private client: AxiosInstance;
  private debugMode: boolean;

  constructor(clientConfig: ApiClientConfig) {
    this.debugMode = config.debugMode;

    this.client = axios.create({
      baseURL: clientConfig.baseURL,
      timeout: clientConfig.timeout,
      headers: {
        'Content-Type': 'application/json',
        ...clientConfig.headers,
      },
      // Ignore HTTPS certificate errors (only for development)
      // In production, use appropriate certificates
      httpsAgent: {
        rejectUnauthorized: false,
      },
    });

    this.setupInterceptors();
  }

  /**
   * Request/response interceptor configuration
   */
  private setupInterceptors() {
    // Request interceptor
    this.client.interceptors.request.use(
      (config) => {
        this.log(`→ ${config.method?.toUpperCase()} ${config.url}`, config.data);

        // Future authentication implementation
        // const token = getAuthToken();
        // if (token) {
        //   config.headers.Authorization = `Bearer ${token}`;
        // }

        return config;
      },
      (error) => {
        this.logError('Request error:', error);
        return Promise.reject(error);
      }
    );

    // Response interceptor
    this.client.interceptors.response.use(
      (response) => {
        this.log(`← ${response.status} ${response.config.url}`, response.data);
        return response;
      },
      (error: AxiosError) => {
        return Promise.reject(this.handleError(error));
      }
    );
  }

  /**
   * Error handling
   */
  private handleError(error: AxiosError): Error {
    this.logError('Response error:', error);

    // Timeout error
    if (error.code === 'ECONNABORTED' || error.code === 'ETIMEDOUT') {
      return new TimeoutError(config.apiTimeout);
    }

    // Network error
    if (!error.response) {
      return new NetworkError(
        error.message || 'Network error occurred',
        error
      );
    }

    // HTTP status code error
    const status = error.response.status;
    const data = error.response.data as ErrorResponse | undefined;

    // Authentication error
    if (status === 401 || status === 403) {
      return new AuthenticationError(
        data?.error || 'Authentication required'
      );
    }

    // API error
    return new ApiError(
      status,
      data?.error || data?.message || error.message,
      data?.details
    );
  }

  /**
   * Debug log output
   */
  private log(message: string, data?: unknown) {
    if (this.debugMode) {
      console.debug(`[API] ${message}`, data);
    }
  }

  /**
   * Error log output
   */
  private logError(message: string, error: unknown) {
    if (this.debugMode) {
      console.error(`[API Error] ${message}`, error);
    }
  }

  /**
   * GET request
   */
  async get<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.get<T>(url, config);
    return response.data;
  }

  /**
   * POST request
   */
  async post<T>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<T> {
    const response = await this.client.post<T>(url, data, config);
    return response.data;
  }

  /**
   * PUT request
   */
  async put<T>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<T> {
    const response = await this.client.put<T>(url, data, config);
    return response.data;
  }

  /**
   * DELETE request
   */
  async delete<T>(url: string, config?: AxiosRequestConfig): Promise<T> {
    const response = await this.client.delete<T>(url, config);
    return response.data;
  }

  /**
   * PATCH request
   */
  async patch<T>(
    url: string,
    data?: unknown,
    config?: AxiosRequestConfig
  ): Promise<T> {
    const response = await this.client.patch<T>(url, data, config);
    return response.data;
  }

  /**
   * File upload (multipart/form-data)
   */
  async upload<T>(
    url: string,
    formData: FormData,
    onUploadProgress?: (progressEvent: { loaded: number; total?: number }) => void
  ): Promise<T> {
    const response = await this.client.post<T>(url, formData, {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      onUploadProgress: onUploadProgress as any,
    });
    return response.data;
  }
}

/**
 * Default API client instance
 */
export const apiClient = new ApiClient({
  baseURL: config.apiBaseURL,
  timeout: config.apiTimeout,
});
